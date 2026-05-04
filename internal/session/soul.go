package session

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var (
	// ErrSoulRefreshConflict reports that a Soul refresh cannot run because
	// another session-scoped operation or an active task run is in progress.
	ErrSoulRefreshConflict = errors.New("session: soul refresh conflict")
	// ErrSoulRefreshDigestConflict reports a stale body-level expected digest.
	ErrSoulRefreshDigestConflict = errors.New("session: soul refresh expected digest conflict")
)

// SoulSnapshotStore is the durable storage needed by session Soul integration.
type SoulSnapshotStore interface {
	UpsertSoulSnapshot(ctx context.Context, snapshot soul.Snapshot) (soul.Snapshot, error)
	GetSoulSnapshot(ctx context.Context, id string) (soul.Snapshot, error)
	UpdateSessionSoulSnapshot(ctx context.Context, update store.SessionSoulSnapshotUpdate) error
}

// SoulRunActivityChecker reports whether a session currently owns an active run.
type SoulRunActivityChecker interface {
	HasActiveRunForSession(ctx context.Context, sessionID string, now time.Time) (bool, error)
}

// SoulRefreshResult is the internal result returned after a session Soul refresh.
type SoulRefreshResult struct {
	SessionID        string
	AgentName        string
	SoulSnapshotID   string
	SoulDigest       string
	ParentSoulDigest string
	Snapshot         *soul.Snapshot
	Soul             *soul.ResolvedSoul
	ConfigProvenance soul.ConfigProvenance
	RefreshedAt      time.Time
}

// RefreshSoul explicitly refreshes the session's resolved Soul snapshot.
func (m *Manager) RefreshSoul(ctx context.Context, id string) (SoulRefreshResult, error) {
	return m.RefreshSoulWithExpectedDigest(ctx, id, "")
}

// RefreshSoulWithExpectedDigest explicitly refreshes a session Soul snapshot after service-owned CAS.
func (m *Manager) RefreshSoulWithExpectedDigest(
	ctx context.Context,
	id string,
	expectedDigest string,
) (SoulRefreshResult, error) {
	if m == nil {
		return SoulRefreshResult{}, errors.New("session: manager is required")
	}
	if ctx == nil {
		return SoulRefreshResult{}, errors.New("session: soul refresh context is required")
	}
	session, err := m.lookup(id)
	if err != nil {
		return SoulRefreshResult{}, err
	}

	release, ok := m.tryAcquireSoulLock(session.ID)
	if !ok {
		return SoulRefreshResult{}, fmt.Errorf("%w: session %q soul lock is busy", ErrSoulRefreshConflict, session.ID)
	}
	defer release()

	info := session.Info()
	if info == nil {
		return SoulRefreshResult{}, fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	}
	if err := validateSoulRefreshExpectedDigest(info, expectedDigest); err != nil {
		return SoulRefreshResult{}, err
	}
	return m.refreshSoulLocked(ctx, session, info)
}

func (m *Manager) refreshSoulLocked(
	ctx context.Context,
	session *Session,
	info *Info,
) (SoulRefreshResult, error) {
	if err := m.ensureSoulRefreshAllowed(ctx, info, m.now()); err != nil {
		return SoulRefreshResult{}, err
	}

	workspaceSnapshot, artifacts, err := m.resolveSoulRefreshTarget(ctx, info)
	if err != nil {
		return SoulRefreshResult{}, err
	}
	resolved, err := m.resolveSoul(ctx, artifacts, &workspaceSnapshot)
	if err != nil {
		return SoulRefreshResult{}, fmt.Errorf("session: resolve soul for refresh %q: %w", info.ID, err)
	}

	durableCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.soulRefreshTimeout)
	defer cancel()

	refreshedAt := m.now().UTC()
	if active, activeErr := m.hasActiveSoulRun(durableCtx, info.ID, refreshedAt); activeErr != nil {
		return SoulRefreshResult{}, activeErr
	} else if active {
		return SoulRefreshResult{}, fmt.Errorf("%w: session %q has an active task run", ErrSoulRefreshConflict, info.ID)
	}

	snapshot, err := m.persistResolvedSoul(
		durableCtx,
		workspaceSnapshot.ID,
		info.AgentName,
		&resolved,
		workspaceSnapshot.Config.Agents.Soul,
		"session_refresh",
		refreshedAt,
	)
	if err != nil {
		return SoulRefreshResult{}, err
	}
	if err := m.storeSoulRefresh(durableCtx, session, info, snapshot, refreshedAt); err != nil {
		return SoulRefreshResult{}, err
	}
	provenance, err := soul.NewConfigProvenance(workspaceSnapshot.Config.Agents.Soul, "session_refresh")
	if err != nil {
		return SoulRefreshResult{}, err
	}
	return newSoulRefreshResult(info, snapshot, &resolved, provenance, refreshedAt), nil
}

func (m *Manager) ensureSoulRefreshAllowed(ctx context.Context, info *Info, now time.Time) error {
	if info.State != StateActive {
		return fmt.Errorf("%w: %s", ErrSessionNotActive, info.ID)
	}
	if active, activeErr := m.hasActiveSoulRun(ctx, info.ID, now); activeErr != nil {
		return activeErr
	} else if active {
		return fmt.Errorf("%w: session %q has an active task run", ErrSoulRefreshConflict, info.ID)
	}
	return nil
}

func (m *Manager) storeSoulRefresh(
	ctx context.Context,
	session *Session,
	info *Info,
	snapshot *soul.Snapshot,
	refreshedAt time.Time,
) error {
	update := store.SessionSoulSnapshotUpdate{
		ID:               info.ID,
		ParentSoulDigest: info.ParentSoulDigest,
		UpdatedAt:        refreshedAt,
	}
	if snapshot != nil {
		update.SoulSnapshotID = snapshot.ID
		update.SoulDigest = snapshot.Digest
	}
	if m.soulStore != nil {
		if err := m.soulStore.UpdateSessionSoulSnapshot(ctx, update); err != nil {
			return fmt.Errorf("session: update soul snapshot for %q: %w", info.ID, err)
		}
	}

	session.updateSoulSnapshot(snapshot, info.ParentSoulDigest, refreshedAt)
	if err := m.writeMeta(session); err != nil {
		return err
	}
	return nil
}

func newSoulRefreshResult(
	info *Info,
	snapshot *soul.Snapshot,
	resolved *soul.ResolvedSoul,
	provenance soul.ConfigProvenance,
	refreshedAt time.Time,
) SoulRefreshResult {
	result := SoulRefreshResult{
		SessionID:        info.ID,
		AgentName:        info.AgentName,
		ParentSoulDigest: info.ParentSoulDigest,
		Snapshot:         cloneSoulSnapshotPointer(snapshot),
		Soul:             cloneResolvedSoulPointer(resolved),
		ConfigProvenance: provenance,
		RefreshedAt:      refreshedAt,
	}
	if snapshot != nil {
		result.SoulSnapshotID = snapshot.ID
		result.SoulDigest = snapshot.Digest
	}
	return result
}

func validateSoulRefreshExpectedDigest(info *Info, expectedDigest string) error {
	if info == nil {
		return errors.New("session: session info is required")
	}
	expected := strings.TrimSpace(expectedDigest)
	if expected == "" {
		return nil
	}
	current := strings.TrimSpace(info.SoulDigest)
	if current == "" {
		return fmt.Errorf("%w: expected_digest was provided but session %q has no Soul snapshot",
			ErrSoulRefreshDigestConflict,
			info.ID,
		)
	}
	if current != expected {
		return fmt.Errorf("%w: expected_digest does not match current session Soul digest",
			ErrSoulRefreshDigestConflict,
		)
	}
	return nil
}

// WithSoulClaimLock runs fn while holding the session Soul lock used by refresh.
func (m *Manager) WithSoulClaimLock(ctx context.Context, sessionID string, fn func() error) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: soul claim lock context is required")
	}
	if fn == nil {
		return errors.New("session: soul claim lock function is required")
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return errors.New("session: session id is required")
	}
	lock := m.sessionSoulLock(target)
	select {
	case <-lock:
		defer releaseSoulLock(lock)
		return fn()
	case <-ctx.Done():
		return fmt.Errorf("session: acquire soul claim lock for %q: %w", target, ctx.Err())
	}
}

func (m *Manager) prepareSessionStartSoul(
	ctx context.Context,
	spec *sessionStartSpec,
	artifacts AgentArtifacts,
	now time.Time,
) error {
	if spec == nil {
		return errors.New("session: start spec is required")
	}
	if spec.startAction == "resume" {
		return m.prepareResumeSoul(ctx, spec)
	}

	resolved, err := m.resolveSoul(ctx, artifacts, &spec.workspace)
	if err != nil {
		return fmt.Errorf("session: resolve soul for agent %q: %w", spec.agentName, err)
	}
	snapshot, err := m.persistResolvedSoul(
		ctx,
		spec.workspace.ID,
		spec.agentName,
		&resolved,
		spec.workspace.Config.Agents.Soul,
		"session_start",
		now,
	)
	if err != nil {
		return err
	}
	spec.applySoulSnapshot(snapshot)
	return nil
}

func (m *Manager) prepareResumeSoul(ctx context.Context, spec *sessionStartSpec) error {
	if spec == nil || strings.TrimSpace(spec.soulSnapshotID) == "" {
		return nil
	}
	if m.soulStore == nil {
		return errors.New("session: soul snapshot store is required to resume session soul")
	}
	snapshot, err := m.soulStore.GetSoulSnapshot(ctx, spec.soulSnapshotID)
	if err != nil {
		return fmt.Errorf("session: load soul snapshot %q: %w", spec.soulSnapshotID, err)
	}
	if spec.soulDigest != "" && strings.TrimSpace(snapshot.Digest) != spec.soulDigest {
		return fmt.Errorf("session: soul snapshot %q digest mismatch", spec.soulSnapshotID)
	}
	spec.soulSnapshot = cloneSoulSnapshotPointer(&snapshot)
	return nil
}

func (m *Manager) resolveSoul(
	ctx context.Context,
	artifacts AgentArtifacts,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) (soul.ResolvedSoul, error) {
	if workspaceSnapshot == nil {
		return soul.ResolvedSoul{}, errors.New("session: resolved workspace is required for soul")
	}
	config := sessionSoulConfig(workspaceSnapshot.Config.Agents.Soul)
	if strings.TrimSpace(artifacts.SoulBody) != "" {
		return soul.Parse(ctx, soul.ParseRequest{
			SourcePath:    strings.TrimSpace(artifacts.SoulSourcePath),
			WorkspaceRoot: m.soulSourceRoot(strings.TrimSpace(artifacts.SoulSourcePath), workspaceSnapshot),
			Content:       []byte(artifacts.SoulBody),
			Config:        config,
		})
	}
	if artifacts.PackageOwned {
		return soul.Empty(config, strings.TrimSpace(artifacts.SoulSourcePath))
	}
	agentPath := soulAgentPath(artifacts.Agent, workspaceSnapshot)
	return soul.Resolve(ctx, soul.ResolveRequest{
		AgentPath:     agentPath,
		WorkspaceRoot: m.soulSourceRoot(agentPath, workspaceSnapshot),
		Config:        config,
	})
}

func (m *Manager) soulSourceRoot(
	sourcePath string,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) string {
	workspaceRoot := ""
	if workspaceSnapshot != nil {
		workspaceRoot = strings.TrimSpace(workspaceSnapshot.RootDir)
	}
	trimmedSource := strings.TrimSpace(sourcePath)
	if trimmedSource == "" || !filepath.IsAbs(trimmedSource) {
		return workspaceRoot
	}

	for _, root := range trustedSoulSourceRoots(m, workspaceSnapshot) {
		if pathWithinRoot(root, trimmedSource) {
			return strings.TrimSpace(root)
		}
	}
	return workspaceRoot
}

func trustedSoulSourceRoots(
	manager *Manager,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) []string {
	roots := make([]string, 0, 3)
	if workspaceSnapshot != nil {
		if root := strings.TrimSpace(workspaceSnapshot.RootDir); root != "" {
			roots = append(roots, root)
		}
		for _, root := range workspaceSnapshot.AdditionalDirs {
			if trimmed := strings.TrimSpace(root); trimmed != "" {
				roots = append(roots, trimmed)
			}
		}
	}
	if manager != nil {
		if home := strings.TrimSpace(manager.homePaths.HomeDir); home != "" {
			roots = append(roots, home)
		}
	}
	return roots
}

func pathWithinRoot(root string, sourcePath string) bool {
	trimmedRoot := strings.TrimSpace(root)
	trimmedSource := strings.TrimSpace(sourcePath)
	if trimmedRoot == "" || trimmedSource == "" {
		return false
	}
	absRoot, err := filepath.Abs(filepath.Clean(trimmedRoot))
	if err != nil {
		return false
	}
	sourceForRoot := filepath.Clean(trimmedSource)
	if !filepath.IsAbs(sourceForRoot) {
		sourceForRoot = filepath.Join(absRoot, sourceForRoot)
	}
	absSource, err := filepath.Abs(sourceForRoot)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absSource)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (m *Manager) persistResolvedSoul(
	ctx context.Context,
	workspaceID string,
	agentName string,
	resolved *soul.ResolvedSoul,
	config aghconfig.SoulConfig,
	source string,
	now time.Time,
) (*soul.Snapshot, error) {
	if resolved == nil {
		return nil, errors.New("session: resolved soul is required")
	}
	config = sessionSoulConfig(config)
	if !resolved.Present || !resolved.Active {
		return nil, nil
	}
	if !resolved.Valid {
		return nil, fmt.Errorf("session: resolved soul for %q is invalid", agentName)
	}
	if m.soulStore == nil {
		return nil, errors.New("session: soul snapshot store is required")
	}
	provenance, err := soul.NewConfigProvenance(config, source)
	if err != nil {
		return nil, err
	}
	snapshot, err := soul.SnapshotFromResolved(
		newID("soul"),
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(agentName),
		resolved,
		provenance,
		now.UTC(),
	)
	if err != nil {
		return nil, err
	}
	persisted, err := m.soulStore.UpsertSoulSnapshot(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("session: persist soul snapshot for %q: %w", agentName, err)
	}
	return &persisted, nil
}

func (m *Manager) resolveSoulRefreshTarget(
	ctx context.Context,
	info *Info,
) (workspacepkg.ResolvedWorkspace, AgentArtifacts, error) {
	if info == nil {
		return workspacepkg.ResolvedWorkspace{}, AgentArtifacts{}, errors.New("session: session info is required")
	}
	resolvedWorkspace, err := m.resolveSoulRefreshWorkspace(ctx, info)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, AgentArtifacts{}, err
	}
	artifacts, err := m.resolveWorkspaceAgentArtifacts(info.AgentName, &resolvedWorkspace)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, AgentArtifacts{}, fmt.Errorf(
			"session: resolve workspace agent %q for soul refresh: %w",
			info.AgentName,
			err,
		)
	}
	return resolvedWorkspace, artifacts, nil
}

func (m *Manager) resolveSoulRefreshWorkspace(
	ctx context.Context,
	info *Info,
) (workspacepkg.ResolvedWorkspace, error) {
	target := firstNonEmpty(strings.TrimSpace(info.WorkspaceID), strings.TrimSpace(info.Workspace))
	if m.workspace != nil && target != "" {
		resolved, err := m.workspace.Resolve(ctx, target)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("session: resolve workspace for soul refresh: %w", err)
		}
		return resolved, nil
	}
	if strings.TrimSpace(info.Workspace) == "" {
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: workspace is required for soul refresh")
	}
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      strings.TrimSpace(info.WorkspaceID),
			RootDir: strings.TrimSpace(info.Workspace),
		},
		Config: aghconfig.DefaultWithHome(m.homePaths),
	}, nil
}

func (m *Manager) hasActiveSoulRun(ctx context.Context, sessionID string, now time.Time) (bool, error) {
	if m == nil || m.soulRunChecker == nil {
		return false, nil
	}
	active, err := m.soulRunChecker.HasActiveRunForSession(ctx, strings.TrimSpace(sessionID), now.UTC())
	if err != nil {
		return false, fmt.Errorf("session: check active task runs for soul refresh: %w", err)
	}
	return active, nil
}

func sessionSoulConfig(config aghconfig.SoulConfig) aghconfig.SoulConfig {
	if config == (aghconfig.SoulConfig{}) {
		return aghconfig.DefaultSoulConfig()
	}
	return config
}

func (s *sessionStartSpec) applySoulSnapshot(snapshot *soul.Snapshot) {
	if s == nil {
		return
	}
	s.soulSnapshot = cloneSoulSnapshotPointer(snapshot)
	s.soulSnapshotID = ""
	s.soulDigest = ""
	if snapshot != nil {
		s.soulSnapshotID = strings.TrimSpace(snapshot.ID)
		s.soulDigest = strings.TrimSpace(snapshot.Digest)
	}
}

func (s *Session) updateSoulSnapshot(snapshot *soul.Snapshot, parentSoulDigest string, updatedAt time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SoulSnapshotID = ""
	s.SoulDigest = ""
	if snapshot != nil {
		s.SoulSnapshotID = strings.TrimSpace(snapshot.ID)
		s.SoulDigest = strings.TrimSpace(snapshot.Digest)
	}
	s.ParentSoulDigest = strings.TrimSpace(parentSoulDigest)
	if !updatedAt.IsZero() {
		s.UpdatedAt = updatedAt.UTC()
	}
}

func soulAgentPath(agentDef aghconfig.AgentDef, workspaceSnapshot *workspacepkg.ResolvedWorkspace) string {
	if sourcePath := strings.TrimSpace(agentDef.SourcePath); sourcePath != "" {
		return sourcePath
	}
	if workspaceSnapshot != nil {
		for _, candidate := range workspaceSnapshot.Agents {
			if strings.TrimSpace(candidate.Name) == strings.TrimSpace(agentDef.Name) &&
				strings.TrimSpace(candidate.SourcePath) != "" {
				return strings.TrimSpace(candidate.SourcePath)
			}
		}
		if root := strings.TrimSpace(workspaceSnapshot.RootDir); root != "" && strings.TrimSpace(agentDef.Name) != "" {
			return filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, agentDef.Name, "AGENT.md")
		}
	}
	return ""
}

func cloneSoulSnapshotPointer(snapshot *soul.Snapshot) *soul.Snapshot {
	if snapshot == nil {
		return nil
	}
	clone := *snapshot
	clone.ProfileJSON = append([]byte(nil), snapshot.ProfileJSON...)
	return &clone
}

func cloneResolvedSoulPointer(resolved *soul.ResolvedSoul) *soul.ResolvedSoul {
	if resolved == nil {
		return nil
	}
	clone := *resolved
	clone.Profile.Tone = append([]string(nil), resolved.Profile.Tone...)
	clone.Profile.Principles = append([]string(nil), resolved.Profile.Principles...)
	clone.Profile.Constraints = append([]string(nil), resolved.Profile.Constraints...)
	clone.Profile.Collaboration = append([]string(nil), resolved.Profile.Collaboration...)
	clone.Profile.MemoryPolicy = append([]string(nil), resolved.Profile.MemoryPolicy...)
	clone.Profile.Tags = append([]string(nil), resolved.Profile.Tags...)
	clone.Compact.Tone = append([]string(nil), resolved.Compact.Tone...)
	clone.Compact.Principles = append([]string(nil), resolved.Compact.Principles...)
	clone.ReadModel.Frontmatter.Tone = append([]string(nil), resolved.ReadModel.Frontmatter.Tone...)
	clone.ReadModel.Frontmatter.Principles = append([]string(nil), resolved.ReadModel.Frontmatter.Principles...)
	clone.ReadModel.Frontmatter.Constraints = append([]string(nil), resolved.ReadModel.Frontmatter.Constraints...)
	clone.ReadModel.Frontmatter.Collaboration = append([]string(nil), resolved.ReadModel.Frontmatter.Collaboration...)
	clone.ReadModel.Frontmatter.MemoryPolicy = append([]string(nil), resolved.ReadModel.Frontmatter.MemoryPolicy...)
	clone.ReadModel.Frontmatter.Tags = append([]string(nil), resolved.ReadModel.Frontmatter.Tags...)
	clone.ReadModel.Diagnostics = append([]soul.Diagnostic(nil), resolved.ReadModel.Diagnostics...)
	clone.Diagnostics = append([]soul.Diagnostic(nil), resolved.Diagnostics...)
	return &clone
}

func (m *Manager) tryAcquireSoulLock(sessionID string) (func(), bool) {
	lock := m.sessionSoulLock(sessionID)
	select {
	case <-lock:
		return func() { releaseSoulLock(lock) }, true
	default:
		return nil, false
	}
}

func (m *Manager) sessionSoulLock(sessionID string) chan struct{} {
	target := strings.TrimSpace(sessionID)
	m.soulLocksMu.Lock()
	defer m.soulLocksMu.Unlock()
	if m.soulLocks == nil {
		m.soulLocks = make(map[string]chan struct{})
	}
	lock, ok := m.soulLocks[target]
	if ok {
		return lock
	}
	lock = make(chan struct{}, 1)
	lock <- struct{}{}
	m.soulLocks[target] = lock
	return lock
}

func releaseSoulLock(lock chan struct{}) {
	if lock == nil {
		return
	}
	select {
	case lock <- struct{}{}:
	default:
	}
}
