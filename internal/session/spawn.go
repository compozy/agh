package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

const (
	// DefaultSpawnMaxChildren is the MVP per-parent active child cap.
	DefaultSpawnMaxChildren = 5
	// DefaultSpawnMaxDepth is the MVP maximum child depth under a root session.
	DefaultSpawnMaxDepth = 1
	// DefaultSpawnRole is used when an agent omits the advisory child role.
	DefaultSpawnRole = "worker"
)

var (
	// ErrSpawnValidation reports a structurally invalid spawn request.
	ErrSpawnValidation = errors.New("session: spawn validation failed")
	// ErrSpawnPermissionDenied reports a failed permission narrowing check.
	ErrSpawnPermissionDenied = errors.New("session: spawn permission denied")
	// ErrSpawnLimitExceeded reports a depth, child, or workspace cap violation.
	ErrSpawnLimitExceeded = errors.New("session: spawn limit exceeded")
)

// SpawnOpts defines the safe child-session creation request accepted by the manager.
type SpawnOpts struct {
	ParentSessionID  string
	AgentName        string
	Provider         string
	Name             string
	Workspace        string
	WorkspacePath    string
	Channel          string
	PromptOverlay    string
	SpawnRole        string
	TTL              time.Duration
	AutoStopOnParent bool
	PermissionPolicy store.SessionPermissionPolicy
	IdempotencyKey   string
}

type permissionCategory struct {
	name   string
	values func(store.SessionPermissionPolicy) []string
}

var knownPermissionCategories = []permissionCategory{
	{name: "tools", values: func(p store.SessionPermissionPolicy) []string { return p.Tools }},
	{name: "skills", values: func(p store.SessionPermissionPolicy) []string { return p.Skills }},
	{name: "mcp_servers", values: func(p store.SessionPermissionPolicy) []string { return p.MCPServers }},
	{name: "workspace_paths", values: func(p store.SessionPermissionPolicy) []string { return p.WorkspacePaths }},
	{name: "network_channels", values: func(p store.SessionPermissionPolicy) []string { return p.NetworkChannels }},
	{name: "sandbox_profiles", values: func(p store.SessionPermissionPolicy) []string {
		return p.SandboxProfiles
	}},
}

// Spawn creates a bounded child session after enforcing lineage, TTL, caps,
// workspace bounds, and permission narrowing.
func (m *Manager) Spawn(ctx context.Context, opts SpawnOpts) (*Session, error) {
	if m == nil {
		return nil, errors.New("session: manager is required")
	}
	if ctx == nil {
		return nil, errors.New("session: spawn context is required")
	}

	m.spawnMu.Lock()
	defer m.spawnMu.Unlock()

	normalized, parent, lineage, err := m.prepareSpawn(ctx, opts)
	if err != nil {
		return nil, err
	}
	workspaceRef, workspacePath := spawnWorkspaceCreateRefs(parent)

	child, err := m.Create(ctx, CreateOpts{
		AgentName:        normalized.AgentName,
		Provider:         normalized.Provider,
		Name:             normalized.Name,
		Workspace:        workspaceRef,
		WorkspacePath:    workspacePath,
		Channel:          spawnChannel(normalized, parent),
		PromptOverlay:    normalized.PromptOverlay,
		Type:             SessionTypeSpawned,
		Lineage:          lineage,
		ParentSoulDigest: strings.TrimSpace(parent.SoulDigest),
	})
	if err != nil {
		return nil, err
	}
	if hookErr := m.dispatchSpawnCreated(ctx, parent, child.Info()); hookErr != nil {
		return child, fmt.Errorf("session: dispatch spawn created hooks for %q: %w", child.ID, hookErr)
	}
	return child, nil
}

func (m *Manager) prepareSpawn(
	ctx context.Context,
	opts SpawnOpts,
) (SpawnOpts, *Info, *store.SessionLineage, error) {
	normalized, err := normalizeSpawnOpts(opts)
	if err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	parent, err := m.spawnParent(ctx, normalized.ParentSessionID)
	if err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	if err := validateSpawnWorkspace(parent, normalized); err != nil {
		return SpawnOpts{}, nil, nil, err
	}

	lineage, err := m.spawnLineage(ctx, parent, normalized)
	if err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	if err := ValidatePermissionSubset(parent.Lineage.PermissionPolicy, normalized.PermissionPolicy); err != nil {
		return SpawnOpts{}, nil, nil, err
	}

	normalized, lineage, err = m.dispatchSpawnPreCreate(ctx, parent, normalized, lineage)
	if err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	if err := validateSpawnWorkspace(parent, normalized); err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	if err := ValidatePermissionSubset(parent.Lineage.PermissionPolicy, normalized.PermissionPolicy); err != nil {
		return SpawnOpts{}, nil, nil, err
	}
	return normalized, parent, lineage, nil
}

func normalizeSpawnOpts(opts SpawnOpts) (SpawnOpts, error) {
	normalized := opts
	normalized.ParentSessionID = strings.TrimSpace(normalized.ParentSessionID)
	normalized.AgentName = strings.TrimSpace(normalized.AgentName)
	normalized.Provider = strings.TrimSpace(normalized.Provider)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Workspace = strings.TrimSpace(normalized.Workspace)
	normalized.WorkspacePath = strings.TrimSpace(normalized.WorkspacePath)
	normalized.Channel = strings.TrimSpace(normalized.Channel)
	normalized.PromptOverlay = strings.TrimSpace(normalized.PromptOverlay)
	normalized.SpawnRole = normalizeSpawnRole(normalized.SpawnRole)
	normalized.PermissionPolicy = store.NormalizeSessionPermissionPolicy(normalized.PermissionPolicy)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)

	switch {
	case normalized.ParentSessionID == "":
		return SpawnOpts{}, spawnValidation("parent_session_id is required")
	case normalized.AgentName == "":
		return SpawnOpts{}, spawnValidation("agent_name is required")
	case normalized.TTL <= 0:
		return SpawnOpts{}, spawnValidation("ttl is required and must be positive")
	case isCoordinatorSpawnRole(normalized.SpawnRole):
		return SpawnOpts{}, spawnValidation("coordinator spawn role is not supported in MVP")
	default:
		return normalized, nil
	}
}

func (m *Manager) spawnParent(ctx context.Context, parentID string) (*Info, error) {
	parent, err := m.Status(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("%w: parent session %q: %w", ErrSpawnValidation, parentID, err)
	}
	if parent == nil {
		return nil, fmt.Errorf("%w: parent session %q returned nil status", ErrSpawnValidation, parentID)
	}
	if parent.State != StateActive {
		return nil, fmt.Errorf("%w: parent session %q is %q", ErrSpawnValidation, parent.ID, parent.State)
	}
	parent.Lineage = store.NormalizeSessionLineage(parent.ID, parent.Lineage)
	return parent, nil
}

func validateSpawnWorkspace(parent *Info, opts SpawnOpts) error {
	if parent == nil {
		return spawnValidation("parent session is required")
	}
	if opts.Workspace != "" && opts.Workspace != parent.WorkspaceID && opts.Workspace != parent.Workspace {
		return fmt.Errorf(
			"%w: child workspace %q is outside parent workspace %q",
			ErrSpawnPermissionDenied,
			opts.Workspace,
			parent.WorkspaceID,
		)
	}
	if opts.WorkspacePath != "" && opts.WorkspacePath != parent.Workspace {
		return fmt.Errorf(
			"%w: child workspace_path %q is outside parent workspace %q",
			ErrSpawnPermissionDenied,
			opts.WorkspacePath,
			parent.Workspace,
		)
	}
	return nil
}

func (m *Manager) spawnLineage(
	ctx context.Context,
	parent *Info,
	opts SpawnOpts,
) (*store.SessionLineage, error) {
	parentLineage := store.NormalizeSessionLineage(parent.ID, parent.Lineage)
	budget := effectiveSpawnBudget(parentLineage.SpawnBudget)
	childDepth := parentLineage.SpawnDepth + 1
	if childDepth > budget.MaxDepth {
		return nil, fmt.Errorf(
			"%w: child depth %d exceeds max_depth %d",
			ErrSpawnLimitExceeded,
			childDepth,
			budget.MaxDepth,
		)
	}
	if err := m.validateSpawnCaps(ctx, parent, parentLineage, budget); err != nil {
		return nil, err
	}

	rootID := strings.TrimSpace(parentLineage.RootSessionID)
	if rootID == "" {
		rootID = parent.ID
	}
	ttlExpiresAt := m.now().UTC().Add(opts.TTL)
	budget.TTLSeconds = durationSecondsCeil(opts.TTL)
	return store.NormalizeSessionLineage("", &store.SessionLineage{
		ParentSessionID:  parent.ID,
		RootSessionID:    rootID,
		SpawnDepth:       childDepth,
		SpawnRole:        opts.SpawnRole,
		TTLExpiresAt:     &ttlExpiresAt,
		AutoStopOnParent: opts.AutoStopOnParent,
		SpawnBudget:      budget,
		PermissionPolicy: opts.PermissionPolicy,
	}), nil
}

func (m *Manager) validateSpawnCaps(
	ctx context.Context,
	parent *Info,
	parentLineage *store.SessionLineage,
	budget store.SessionSpawnBudget,
) error {
	infos, err := m.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("%w: count child sessions for %q: %w", ErrSpawnValidation, parent.ID, err)
	}
	activeChildren := 0
	activeInWorkspace := 0
	rootID := strings.TrimSpace(parentLineage.RootSessionID)
	if rootID == "" {
		rootID = parent.ID
	}
	for _, info := range infos {
		if info == nil || info.Lineage == nil || !isLiveSpawnState(info.State) {
			continue
		}
		lineage := store.NormalizeSessionLineage(info.ID, info.Lineage)
		if lineage.ParentSessionID == parent.ID {
			activeChildren++
		}
		if budget.MaxActivePerWorkspace > 0 &&
			lineage.RootSessionID == rootID &&
			info.WorkspaceID == parent.WorkspaceID {
			activeInWorkspace++
		}
	}
	if activeChildren >= budget.MaxChildren {
		return fmt.Errorf(
			"%w: parent %q has %d active children, max_children %d",
			ErrSpawnLimitExceeded,
			parent.ID,
			activeChildren,
			budget.MaxChildren,
		)
	}
	if budget.MaxActivePerWorkspace > 0 && activeInWorkspace >= budget.MaxActivePerWorkspace {
		return fmt.Errorf(
			"%w: workspace %q has %d active spawned sessions, max_active_per_workspace %d",
			ErrSpawnLimitExceeded,
			parent.WorkspaceID,
			activeInWorkspace,
			budget.MaxActivePerWorkspace,
		)
	}
	return nil
}

func (m *Manager) dispatchSpawnPreCreate(
	ctx context.Context,
	parent *Info,
	opts SpawnOpts,
	lineage *store.SessionLineage,
) (SpawnOpts, *store.SessionLineage, error) {
	payload := hookspkg.SpawnPreCreatePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSpawnPreCreate,
			Timestamp: m.now().UTC(),
		},
		SpawnContext:      spawnHookContext(parent, nil, lineage, opts.AgentName, opts.SpawnRole),
		ParentPermissions: hookPermissionSetFromPolicy(parent.Lineage.PermissionPolicy),
		ChildPermissions:  hookPermissionSetFromPolicy(opts.PermissionPolicy),
	}
	result, err := m.hooks.spawn().DispatchSpawnPreCreate(ctx, payload)
	if err != nil {
		return SpawnOpts{}, nil, fmt.Errorf("%w: %w", ErrSpawnPermissionDenied, err)
	}
	if result.Denied {
		reason := strings.TrimSpace(result.DenyReason)
		if reason == "" {
			reason = "spawn denied by hook"
		}
		return SpawnOpts{}, nil, fmt.Errorf("%w: %s", ErrSpawnPermissionDenied, reason)
	}

	opts.AgentName = strings.TrimSpace(result.AgentName)
	opts.SpawnRole = normalizeSpawnRole(result.SpawnRole)
	opts.TTL = time.Duration(result.TTLSeconds) * time.Second
	opts.PermissionPolicy = policyFromHookPermissionSet(result.ChildPermissions)
	normalized, err := normalizeSpawnOpts(opts)
	if err != nil {
		return SpawnOpts{}, nil, err
	}
	lineage, err = m.spawnLineage(ctx, parent, normalized)
	if err != nil {
		return SpawnOpts{}, nil, err
	}
	return normalized, lineage, nil
}

func (m *Manager) dispatchSpawnCreated(ctx context.Context, parent *Info, child *Info) error {
	if parent == nil || child == nil || child.Lineage == nil {
		return nil
	}
	lineage := store.NormalizeSessionLineage(child.ID, child.Lineage)
	payload := hookspkg.SpawnCreatedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSpawnCreated,
			Timestamp: m.now().UTC(),
		},
		SpawnContext:      spawnHookContext(parent, child, lineage, child.AgentName, lineage.SpawnRole),
		ParentPermissions: hookPermissionSetFromPolicy(parent.Lineage.PermissionPolicy),
		ChildPermissions:  hookPermissionSetFromPolicy(lineage.PermissionPolicy),
	}
	_, err := m.hooks.spawn().DispatchSpawnCreated(ctx, payload)
	return err
}

func spawnHookContext(
	parent *Info,
	child *Info,
	lineage *store.SessionLineage,
	agentName string,
	spawnRole string,
) hookspkg.SpawnContext {
	ctx := hookspkg.SpawnContext{
		AgentName:        strings.TrimSpace(agentName),
		SpawnRole:        strings.TrimSpace(spawnRole),
		ParentSessionID:  strings.TrimSpace(lineage.ParentSessionID),
		RootSessionID:    strings.TrimSpace(lineage.RootSessionID),
		SpawnDepth:       lineage.SpawnDepth,
		AutoStopOnParent: lineage.AutoStopOnParent,
		TTLSeconds:       lineage.SpawnBudget.TTLSeconds,
	}
	if parent != nil {
		ctx.WorkspaceID = strings.TrimSpace(parent.WorkspaceID)
		ctx.Workspace = strings.TrimSpace(parent.Workspace)
		ctx.ParentSoulDigest = strings.TrimSpace(parent.SoulDigest)
	}
	if child != nil {
		ctx.ChildSessionID = strings.TrimSpace(child.ID)
		ctx.WorkspaceID = strings.TrimSpace(child.WorkspaceID)
		ctx.Workspace = strings.TrimSpace(child.Workspace)
		ctx.SoulSnapshotID = strings.TrimSpace(child.SoulSnapshotID)
		ctx.SoulDigest = strings.TrimSpace(child.SoulDigest)
		if value := strings.TrimSpace(child.ParentSoulDigest); value != "" {
			ctx.ParentSoulDigest = value
		}
	}
	return ctx
}

// ValidatePermissionSubset fails closed unless every child permission atom is
// present in the corresponding known parent category.
func ValidatePermissionSubset(parent store.SessionPermissionPolicy, child store.SessionPermissionPolicy) error {
	normalizedParent := store.NormalizeSessionPermissionPolicy(parent)
	normalizedChild := store.NormalizeSessionPermissionPolicy(child)
	for _, category := range knownPermissionCategories {
		if err := validatePermissionAtoms(
			category.name,
			category.values(normalizedParent),
			category.values(normalizedChild),
		); err != nil {
			return fmt.Errorf("%w: %w", ErrSpawnPermissionDenied, err)
		}
	}
	return nil
}

func validatePermissionAtoms(category string, parent []string, child []string) error {
	allowed := make(map[string]struct{}, len(parent))
	for _, atom := range parent {
		trimmed := strings.TrimSpace(atom)
		if trimmed == "" {
			return fmt.Errorf("parent %s includes a blank permission atom", category)
		}
		allowed[trimmed] = struct{}{}
	}
	for _, atom := range child {
		trimmed := strings.TrimSpace(atom)
		if trimmed == "" {
			return fmt.Errorf("child %s includes a blank permission atom", category)
		}
		if _, ok := allowed[trimmed]; !ok {
			return fmt.Errorf("child %s permission atom %q widens parent permissions", category, trimmed)
		}
	}
	return nil
}

func effectiveSpawnBudget(budget store.SessionSpawnBudget) store.SessionSpawnBudget {
	normalized := budget
	if normalized.MaxChildren <= 0 {
		normalized.MaxChildren = DefaultSpawnMaxChildren
	}
	if normalized.MaxDepth <= 0 {
		normalized.MaxDepth = DefaultSpawnMaxDepth
	}
	return normalized
}

func hookPermissionSetFromPolicy(policy store.SessionPermissionPolicy) *hookspkg.PermissionSet {
	normalized := store.NormalizeSessionPermissionPolicy(policy)
	return &hookspkg.PermissionSet{
		Tools:           append([]string(nil), normalized.Tools...),
		Skills:          append([]string(nil), normalized.Skills...),
		MCPServers:      append([]string(nil), normalized.MCPServers...),
		WorkspacePaths:  append([]string(nil), normalized.WorkspacePaths...),
		NetworkChannels: append([]string(nil), normalized.NetworkChannels...),
		SandboxProfiles: append([]string(nil), normalized.SandboxProfiles...),
	}
}

func policyFromHookPermissionSet(src *hookspkg.PermissionSet) store.SessionPermissionPolicy {
	if src == nil {
		return store.NormalizeSessionPermissionPolicy(store.SessionPermissionPolicy{})
	}
	return store.NormalizeSessionPermissionPolicy(store.SessionPermissionPolicy{
		Tools:           append([]string(nil), src.Tools...),
		Skills:          append([]string(nil), src.Skills...),
		MCPServers:      append([]string(nil), src.MCPServers...),
		WorkspacePaths:  append([]string(nil), src.WorkspacePaths...),
		NetworkChannels: append([]string(nil), src.NetworkChannels...),
		SandboxProfiles: append([]string(nil), src.SandboxProfiles...),
	})
}

func spawnChannel(opts SpawnOpts, parent *Info) string {
	if opts.Channel != "" {
		return opts.Channel
	}
	if parent == nil {
		return ""
	}
	return strings.TrimSpace(parent.Channel)
}

func spawnWorkspaceCreateRefs(parent *Info) (string, string) {
	if parent == nil {
		return "", ""
	}
	if workspaceID := strings.TrimSpace(parent.WorkspaceID); workspaceID != "" {
		return workspaceID, ""
	}
	return "", strings.TrimSpace(parent.Workspace)
}

func normalizeSpawnRole(role string) string {
	trimmed := strings.TrimSpace(role)
	if trimmed == "" {
		return DefaultSpawnRole
	}
	return trimmed
}

func isCoordinatorSpawnRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), string(SessionTypeCoordinator))
}

func isLiveSpawnState(state State) bool {
	switch state {
	case StateStarting, StateActive, StateStopping:
		return true
	default:
		return false
	}
}

func durationSecondsCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	seconds := int64(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}

func spawnValidation(message string) error {
	return fmt.Errorf("%w: %s", ErrSpawnValidation, strings.TrimSpace(message))
}
