// Package situation assembles the bounded runtime context agents need to act.
package situation

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	// DefaultSectionLimit is the MVP bound for list sections inside agent context.
	DefaultSectionLimit = 8
	// ProvenanceSource identifies the local daemon context assembler.
	ProvenanceSource = "daemon.situation"

	defaultMaxSpawnDepth       = 1
	defaultMaxActiveTaskLeases = 1
	inboxPreviewLimit          = 180
)

// WorkspaceResolver resolves persisted workspaces into runtime snapshots.
type WorkspaceResolver interface {
	Resolve(ctx context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error)
}

// AgentResolver resolves one agent definition for a workspace.
type AgentResolver interface {
	ResolveAgent(name string, resolved *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)
}

// SkillRegistry resolves the active skill set for a workspace.
type SkillRegistry interface {
	ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)
	ForAgent(
		ctx context.Context,
		resolved *workspacepkg.ResolvedWorkspace,
		agentName string,
	) ([]*skillspkg.Skill, error)
}

// TaskStore is the narrowed task read surface required by agent context.
type TaskStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskRuns(ctx context.Context, query taskpkg.RunQuery) ([]taskpkg.Run, error)
	ListTaskEvents(ctx context.Context, query taskpkg.EventQuery) ([]taskpkg.Event, error)
	ListTaskEventRecords(ctx context.Context, query taskpkg.EventRecordQuery) ([]taskpkg.EventRecord, error)
	GetExecutionProfile(ctx context.Context, taskID string) (taskpkg.ExecutionProfile, error)
	GetRunReview(ctx context.Context, reviewID string) (taskpkg.RunReview, error)
	LookupRunReviewBySession(ctx context.Context, sessionID string) (taskpkg.RunReview, error)
	ListRunReviews(ctx context.Context, query taskpkg.RunReviewQuery) ([]taskpkg.RunReview, error)
}

// NetworkReader is the narrowed network read surface required by agent context.
type NetworkReader interface {
	ListPeers(ctx context.Context, workspaceID string, channel string) ([]network.PeerInfo, error)
	Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error)
}

// CoordinatorConfigResolver reads the safe coordinator limits for a workspace.
type CoordinatorConfigResolver interface {
	ResolveCoordinatorConfig(ctx context.Context, workspaceID string) (aghconfig.CoordinatorConfig, error)
}

// SoulSnapshotStore loads immutable Soul snapshots for compact context projection.
type SoulSnapshotStore interface {
	GetSoulSnapshot(ctx context.Context, id string) (soul.Snapshot, error)
}

// Deps wires situation context to daemon-owned services. Function fields are
// evaluated at render time so daemon boot can install the provider before late
// runtime services are available.
type Deps struct {
	Now func() time.Time

	SectionLimit int

	WorkspaceResolver     WorkspaceResolver
	WorkspaceResolverFunc func() WorkspaceResolver
	AgentResolver         AgentResolver
	AgentResolverFunc     func() AgentResolver
	SkillRegistry         SkillRegistry
	SkillRegistryFunc     func() SkillRegistry
	TaskStore             TaskStore
	TaskStoreFunc         func() TaskStore
	Network               NetworkReader
	NetworkFunc           func() NetworkReader
	CoordinatorConfig     CoordinatorConfigResolver
	CoordinatorConfigFunc func() CoordinatorConfigResolver
	SoulSnapshots         SoulSnapshotStore
	SoulSnapshotsFunc     func() SoulSnapshotStore
}

// Service assembles contract.AgentContextPayload and renders prompt sections.
type Service struct {
	now          func() time.Time
	sectionLimit int

	workspaceResolver     WorkspaceResolver
	workspaceResolverFunc func() WorkspaceResolver
	agentResolver         AgentResolver
	agentResolverFunc     func() AgentResolver
	skillRegistry         SkillRegistry
	skillRegistryFunc     func() SkillRegistry
	taskStore             TaskStore
	taskStoreFunc         func() TaskStore
	network               NetworkReader
	networkFunc           func() NetworkReader
	coordinatorConfig     CoordinatorConfigResolver
	coordinatorConfigFunc func() CoordinatorConfigResolver
	soulSnapshots         SoulSnapshotStore
	soulSnapshotsFunc     func() SoulSnapshotStore
}

// NewService constructs a deterministic situation context assembler.
func NewService(deps Deps) *Service {
	now := deps.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	limit := deps.SectionLimit
	if limit <= 0 {
		limit = DefaultSectionLimit
	}

	return &Service{
		now:                   now,
		sectionLimit:          limit,
		workspaceResolver:     deps.WorkspaceResolver,
		workspaceResolverFunc: deps.WorkspaceResolverFunc,
		agentResolver:         deps.AgentResolver,
		agentResolverFunc:     deps.AgentResolverFunc,
		skillRegistry:         deps.SkillRegistry,
		skillRegistryFunc:     deps.SkillRegistryFunc,
		taskStore:             deps.TaskStore,
		taskStoreFunc:         deps.TaskStoreFunc,
		network:               deps.Network,
		networkFunc:           deps.NetworkFunc,
		coordinatorConfig:     deps.CoordinatorConfig,
		coordinatorConfigFunc: deps.CoordinatorConfigFunc,
		soulSnapshots:         deps.SoulSnapshots,
		soulSnapshotsFunc:     deps.SoulSnapshotsFunc,
	}
}

// ContextForStartup assembles the bounded context available before the agent driver starts.
func (s *Service) ContextForStartup(
	ctx context.Context,
	startup session.StartupPromptContext,
	agent aghconfig.AgentDef,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) (contract.AgentContextPayload, error) {
	if s == nil {
		return contract.AgentContextPayload{}, nil
	}
	if err := checkContext(ctx); err != nil {
		return contract.AgentContextPayload{}, err
	}

	workspaceSnapshot, err := s.resolveWorkspace(ctx, startup.WorkspaceID, startup.Workspace, resolvedWorkspace)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}

	agentName := firstTrimmed(startup.AgentName, agent.Name)
	resolvedAgent, agentDef, err := s.resolveAgent(agentName, startup.Provider, agent, workspaceSnapshot)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}

	payload := contract.AgentContextPayload{
		Self: contract.AgentIdentityPayload{
			SessionID: strings.TrimSpace(startup.SessionID),
			AgentName: agentName,
			Provider:  firstTrimmed(resolvedAgent.Provider, startup.Provider, agent.Provider),
			Model:     firstTrimmed(resolvedAgent.Model, agent.Model),
		},
		Workspace:    workspacePayload(workspaceSnapshot, startup.WorkspaceID, startup.Workspace),
		Session:      startupSessionPayload(startup),
		Capabilities: s.capabilitiesSection(ctx, workspaceSnapshot, agentDef),
		Limits:       s.limits(ctx, workspacePayload(workspaceSnapshot, startup.WorkspaceID, startup.Workspace).ID),
		Provenance:   s.provenance(),
	}

	normalized := contract.NormalizeAgentContextPayload(&payload)
	return normalized, nil
}

// ContextForSession assembles the bounded context for an active session.
func (s *Service) ContextForSession(
	ctx context.Context,
	info *session.Info,
) (contract.AgentContextPayload, error) {
	if s == nil {
		return contract.AgentContextPayload{}, nil
	}
	if info == nil {
		return contract.AgentContextPayload{}, errors.New("situation: session info is required")
	}
	if err := checkContext(ctx); err != nil {
		return contract.AgentContextPayload{}, err
	}

	workspaceSnapshot, err := s.resolveWorkspace(ctx, info.WorkspaceID, info.Workspace, nil)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}
	resolvedAgent, agentDef, err := s.resolveAgent(
		info.AgentName,
		info.Provider,
		aghconfig.AgentDef{},
		workspaceSnapshot,
	)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}

	workspaceSection := workspacePayload(workspaceSnapshot, info.WorkspaceID, info.Workspace)
	payload := contract.AgentContextPayload{
		Self: contract.AgentIdentityPayload{
			SessionID: strings.TrimSpace(info.ID),
			AgentName: strings.TrimSpace(info.AgentName),
			Provider:  firstTrimmed(info.Provider, resolvedAgent.Provider, agentDef.Provider),
			Model:     firstTrimmed(resolvedAgent.Model, agentDef.Model),
		},
		Workspace:    workspaceSection,
		Session:      sessionPayload(info),
		Soul:         s.soulPayload(ctx, info),
		Capabilities: s.capabilitiesSection(ctx, workspaceSnapshot, agentDef),
		Limits:       s.limits(ctx, workspaceSection.ID),
		Provenance:   s.provenance(),
	}

	taskContext, channelContext, activeChannel, err := s.taskAndChannelContext(ctx, info.ID, workspaceSnapshot)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}
	payload.Task = taskContext
	payload.CoordinationChannel = channelContext

	networkChannel := firstTrimmed(activeChannel, info.Channel)
	inbox, peers, err := s.networkSections(
		ctx,
		info.ID,
		workspaceSection.ID,
		networkChannel,
		coordinationChannelID(channelContext),
	)
	if err != nil {
		return contract.AgentContextPayload{}, err
	}
	payload.InboxSummary = inbox
	payload.PeerRoster = peers

	normalized := contract.NormalizeAgentContextPayload(&payload)
	return normalized, nil
}

// PromptSection implements the legacy workspace-scoped prompt provider seam.
func (s *Service) PromptSection(
	ctx context.Context,
	workspace *workspacepkg.ResolvedWorkspace,
) (string, error) {
	payload, err := s.ContextForStartup(ctx, session.StartupPromptContext{
		WorkspaceID: workspaceID(workspace),
		Workspace:   workspaceRoot(workspace),
		SessionType: session.SessionTypeUser,
	}, aghconfig.AgentDef{}, workspace)
	if err != nil {
		return "", err
	}
	return RenderPrompt(&payload)
}

// PromptStartupSection renders the startup prompt section with full startup metadata.
func (s *Service) PromptStartupSection(
	ctx context.Context,
	startup session.StartupPromptContext,
	agent aghconfig.AgentDef,
	workspace *workspacepkg.ResolvedWorkspace,
) (string, error) {
	payload, err := s.ContextForStartup(ctx, startup, agent, workspace)
	if err != nil {
		return "", err
	}
	return RenderPrompt(&payload)
}

// Augment prefixes a live prompt with fresh situation context.
func (s *Service) Augment(
	ctx context.Context,
	sess *session.Session,
	message string,
) (string, error) {
	if s == nil || sess == nil {
		return message, nil
	}
	payload, err := s.ContextForSession(ctx, sess.Info())
	if err != nil {
		return "", err
	}
	rendered, err := RenderPrompt(&payload)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(rendered) == "" {
		return message, nil
	}
	if strings.TrimSpace(message) == "" {
		return rendered, nil
	}
	return rendered + "\n\n" + message, nil
}

func (s *Service) resolveWorkspace(
	ctx context.Context,
	workspaceID string,
	rootDir string,
	provided *workspacepkg.ResolvedWorkspace,
) (*workspacepkg.ResolvedWorkspace, error) {
	if provided != nil {
		clone := *provided
		return &clone, nil
	}

	target := firstTrimmed(workspaceID, rootDir)
	if resolver := s.workspaceResolverValue(); resolver != nil && target != "" {
		resolved, err := resolver.Resolve(ctx, target)
		if err == nil {
			return &resolved, nil
		}
		if isContextError(err) {
			return nil, err
		}
	}

	if strings.TrimSpace(workspaceID) == "" && strings.TrimSpace(rootDir) == "" {
		return nil, nil
	}
	return &workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      strings.TrimSpace(workspaceID),
			RootDir: strings.TrimSpace(rootDir),
		},
	}, nil
}

func (s *Service) resolveAgent(
	name string,
	providerOverride string,
	provided aghconfig.AgentDef,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) (aghconfig.ResolvedAgent, aghconfig.AgentDef, error) {
	agent := provided
	agentName := firstTrimmed(name, agent.Name)
	if resolver := s.agentResolverValue(); resolver != nil && agentName != "" {
		resolved, err := resolver.ResolveAgent(agentName, workspaceSnapshot)
		switch {
		case err == nil:
			agent = resolved
		case isContextError(err):
			return aghconfig.ResolvedAgent{}, aghconfig.AgentDef{}, err
		case !errors.Is(err, workspacepkg.ErrAgentNotAvailable) && agent.Name == "":
			return aghconfig.ResolvedAgent{}, aghconfig.AgentDef{}, err
		}
	}
	if strings.TrimSpace(agent.Name) == "" {
		agent.Name = agentName
	}
	if agentName == "" &&
		strings.TrimSpace(providerOverride) == "" &&
		strings.TrimSpace(agent.Provider) == "" &&
		strings.TrimSpace(agent.Model) == "" {
		return aghconfig.ResolvedAgent{}, agent, nil
	}

	if workspaceSnapshot != nil {
		provider := firstTrimmed(providerOverride, agent.Provider, workspaceSnapshot.Config.Defaults.Provider)
		model := strings.TrimSpace(agent.Model)
		if provider != "" && model == "" {
			if providerConfig, err := workspaceSnapshot.Config.ResolveProvider(provider); err == nil {
				model = strings.TrimSpace(providerConfig.Models.Default)
			}
		}
		return aghconfig.ResolvedAgent{
			Name:     strings.TrimSpace(agent.Name),
			Provider: provider,
			Model:    model,
		}, agent, nil
	}

	return aghconfig.ResolvedAgent{
		Name:     strings.TrimSpace(agent.Name),
		Provider: firstTrimmed(providerOverride, agent.Provider),
		Model:    strings.TrimSpace(agent.Model),
	}, agent, nil
}

func (s *Service) capabilitiesSection(
	ctx context.Context,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
	agent aghconfig.AgentDef,
) contract.AgentCapabilitySectionPayload {
	capabilities := make([]contract.AgentCapabilityPayload, 0)
	if catalog := agent.Capabilities; catalog != nil {
		for _, capability := range catalog.Capabilities {
			id := strings.TrimSpace(capability.ID)
			if id == "" {
				continue
			}
			capabilities = append(capabilities, contract.AgentCapabilityPayload{
				ID:      id,
				Summary: strings.TrimSpace(capability.Summary),
				Source:  "agent",
			})
		}
	}

	if registry := s.skillRegistryValue(); registry != nil {
		skills, err := registry.ForWorkspace(ctx, workspaceSnapshot)
		if strings.TrimSpace(agent.Name) != "" {
			skills, err = registry.ForAgent(ctx, workspaceSnapshot, agent.Name)
		}
		if err == nil {
			for _, skill := range skills {
				if skill == nil || !skill.Enabled {
					continue
				}
				name := strings.TrimSpace(skill.Meta.Name)
				if name == "" {
					continue
				}
				capabilities = append(capabilities, contract.AgentCapabilityPayload{
					ID:      "skill:" + name,
					Summary: strings.TrimSpace(skill.Meta.Description),
					Source:  "skill",
				})
			}
		}
	}

	slices.SortStableFunc(capabilities, func(left, right contract.AgentCapabilityPayload) int {
		if left.Source != right.Source {
			return strings.Compare(left.Source, right.Source)
		}
		return strings.Compare(left.ID, right.ID)
	})
	return contract.AgentCapabilitySectionPayload{
		Section:      sectionMeta(len(capabilities), s.limit()),
		Capabilities: boundedCapabilities(capabilities, s.limit()),
	}
}

func (s *Service) limits(ctx context.Context, workspaceID string) contract.AgentLimitsPayload {
	limits := contract.AgentLimitsPayload{
		MaxChildren:         aghconfig.DefaultCoordinatorMaxChildren,
		MaxSpawnDepth:       defaultMaxSpawnDepth,
		MaxActiveTaskLeases: defaultMaxActiveTaskLeases,
		ContextSectionLimit: s.limit(),
	}
	if resolver := s.coordinatorConfigValue(); resolver != nil {
		cfg, err := resolver.ResolveCoordinatorConfig(ctx, strings.TrimSpace(workspaceID))
		if err == nil && cfg.MaxChildren > 0 {
			limits.MaxChildren = cfg.MaxChildren
		}
	}
	return limits
}

func (s *Service) taskAndChannelContext(
	ctx context.Context,
	sessionID string,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) (contract.AgentTaskContextPayload, contract.AgentCoordinationChannelContextPayload, string, error) {
	store := s.taskStoreValue()
	if store == nil || strings.TrimSpace(sessionID) == "" {
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}

	runs, err := store.ListTaskRuns(ctx, taskpkg.RunQuery{SessionID: strings.TrimSpace(sessionID)})
	if err != nil {
		if isContextError(err) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
		}
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}
	run, ok := selectActiveRun(runs)
	if !ok {
		return s.reviewBindingTaskAndChannelContext(ctx, store, sessionID, workspaceSnapshot)
	}

	taskRecord, err := store.GetTask(ctx, run.TaskID)
	if err != nil {
		if isContextError(err) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
		}
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}

	bundle, err := s.sessionContextBundle(ctx, taskRecord, run, workspaceSnapshot, strings.TrimSpace(sessionID))
	if err != nil {
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
	}

	channel := coordinationChannelPayload(taskRecord, run)
	lease := contract.TaskRunLeaseSummaryPayload{
		TaskID:                strings.TrimSpace(run.TaskID),
		RunID:                 strings.TrimSpace(run.ID),
		Status:                run.Status.Normalize(),
		SessionID:             strings.TrimSpace(run.SessionID),
		ClaimedBy:             cloneActorIdentity(run.ClaimedBy),
		CoordinationChannelID: channel.ID,
	}
	if channel.ID != "" {
		lease.CoordinationChannel = &channel
	}

	taskContext := contract.AgentTaskContextPayload{
		Available: true,
		Task:      taskReferencePayload(taskRecord),
		Lease:     &lease,
		Bundle:    bundle,
	}
	channelContext := contract.AgentCoordinationChannelContextPayload{
		Available: channel.ID != "",
		Channel:   &channel,
	}
	if !channelContext.Available {
		channelContext.Channel = nil
	}
	return taskContext, channelContext, firstTrimmed(channel.Channel, channel.ID), nil
}

func (s *Service) reviewBindingTaskAndChannelContext(
	ctx context.Context,
	store TaskStore,
	sessionID string,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
) (contract.AgentTaskContextPayload, contract.AgentCoordinationChannelContextPayload, string, error) {
	review, err := store.LookupRunReviewBySession(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		if errors.Is(err, taskpkg.ErrRunReviewNotFound) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
		}
		if isContextError(err) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
		}
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}
	taskRecord, err := store.GetTask(ctx, review.TaskID)
	if err != nil {
		if isContextError(err) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
		}
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}
	run, err := store.GetTaskRun(ctx, review.RunID)
	if err != nil {
		if isContextError(err) {
			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
		}
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}
	if strings.TrimSpace(run.TaskID) != strings.TrimSpace(taskRecord.ID) {
		slog.Warn(
			"situation: skip review-bound context for mismatched task and run",
			"session_id", strings.TrimSpace(sessionID),
			"review_id", strings.TrimSpace(review.ReviewID),
			"task_id", strings.TrimSpace(taskRecord.ID),
			"run_id", strings.TrimSpace(run.ID),
			"run_task_id", strings.TrimSpace(run.TaskID),
		)
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
	}
	bundle, err := s.sessionContextBundle(ctx, taskRecord, run, workspaceSnapshot, strings.TrimSpace(sessionID))
	if err != nil {
		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
	}

	channel := coordinationChannelPayload(taskRecord, run)
	if reviewerChannelID := strings.TrimSpace(review.ReviewerChannelID); reviewerChannelID != "" {
		channel.ID = reviewerChannelID
		channel.Channel = reviewerChannelID
		channel.DisplayName = reviewerChannelID
		channel = contract.NormalizeCoordinationChannelPayload(channel)
	}

	taskContext := contract.AgentTaskContextPayload{
		Available: true,
		Task:      taskReferencePayload(taskRecord),
		Bundle:    bundle,
	}
	channelContext := contract.AgentCoordinationChannelContextPayload{
		Available: channel.ID != "",
		Channel:   &channel,
	}
	if !channelContext.Available {
		channelContext.Channel = nil
	}
	return taskContext, channelContext, firstTrimmed(review.ReviewerChannelID, channel.Channel, channel.ID), nil
}

func (s *Service) sessionContextBundle(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
	sessionID string,
) (*taskpkg.ContextBundle, error) {
	bundle, err := s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, nil)
	if err == nil {
		return &bundle, nil
	}
	if isContextError(err) {
		return nil, err
	}

	slog.Warn(
		"situation: skip task context bundle enrichment",
		"session_id", strings.TrimSpace(sessionID),
		"task_id", strings.TrimSpace(taskRecord.ID),
		"run_id", strings.TrimSpace(run.ID),
		"error", safeTaskContextText(err.Error(), 240),
	)
	return nil, nil
}

func (s *Service) networkSections(
	ctx context.Context,
	sessionID string,
	workspaceID string,
	channel string,
	activeCoordinationChannelID string,
) (contract.AgentInboxSummaryPayload, contract.AgentPeerRosterPayload, error) {
	reader := s.networkValue()
	if reader == nil {
		return contract.AgentInboxSummaryPayload{}, contract.AgentPeerRosterPayload{}, nil
	}

	inbox := contract.AgentInboxSummaryPayload{Section: emptySectionMeta(s.limit())}
	if strings.TrimSpace(sessionID) != "" {
		envelopes, err := reader.Inbox(ctx, strings.TrimSpace(sessionID))
		if err != nil {
			if isContextError(err) {
				return contract.AgentInboxSummaryPayload{}, contract.AgentPeerRosterPayload{}, err
			}
		} else {
			inbox = inboxSummary(envelopes, s.limit(), activeCoordinationChannelID)
		}
	}

	peers := contract.AgentPeerRosterPayload{Section: emptySectionMeta(s.limit())}
	if strings.TrimSpace(channel) != "" {
		peerInfos, err := reader.ListPeers(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(channel))
		if err != nil {
			if isContextError(err) {
				return contract.AgentInboxSummaryPayload{}, contract.AgentPeerRosterPayload{}, err
			}
		} else {
			peers = peerRoster(peerInfos, strings.TrimSpace(sessionID), s.limit())
		}
	}

	return inbox, peers, nil
}

func (s *Service) provenance() contract.AgentContextProvenancePayload {
	return contract.AgentContextProvenancePayload{
		GeneratedAt: s.now().UTC(),
		Source:      ProvenanceSource,
	}
}

func (s *Service) limit() int {
	if s == nil || s.sectionLimit <= 0 {
		return DefaultSectionLimit
	}
	return s.sectionLimit
}

func (s *Service) workspaceResolverValue() WorkspaceResolver {
	if s == nil {
		return nil
	}
	if s.workspaceResolverFunc != nil {
		return s.workspaceResolverFunc()
	}
	return s.workspaceResolver
}

func (s *Service) agentResolverValue() AgentResolver {
	if s == nil {
		return nil
	}
	if s.agentResolverFunc != nil {
		return s.agentResolverFunc()
	}
	return s.agentResolver
}

func (s *Service) skillRegistryValue() SkillRegistry {
	if s == nil {
		return nil
	}
	if s.skillRegistryFunc != nil {
		return s.skillRegistryFunc()
	}
	return s.skillRegistry
}

func (s *Service) taskStoreValue() TaskStore {
	if s == nil {
		return nil
	}
	if s.taskStoreFunc != nil {
		return s.taskStoreFunc()
	}
	return s.taskStore
}

func (s *Service) networkValue() NetworkReader {
	if s == nil {
		return nil
	}
	if s.networkFunc != nil {
		return s.networkFunc()
	}
	return s.network
}

func (s *Service) coordinatorConfigValue() CoordinatorConfigResolver {
	if s == nil {
		return nil
	}
	if s.coordinatorConfigFunc != nil {
		return s.coordinatorConfigFunc()
	}
	return s.coordinatorConfig
}

func (s *Service) soulSnapshotsValue() SoulSnapshotStore {
	if s == nil {
		return nil
	}
	if s.soulSnapshotsFunc != nil {
		return s.soulSnapshotsFunc()
	}
	return s.soulSnapshots
}

func (s *Service) soulPayload(ctx context.Context, info *session.Info) contract.AgentSoulSectionPayload {
	if info == nil || strings.TrimSpace(info.SoulSnapshotID) == "" {
		return contract.AgentSoulSectionPayload{}
	}
	store := s.soulSnapshotsValue()
	if store == nil {
		return contract.AgentSoulSectionPayload{
			Present:    true,
			Active:     true,
			Valid:      true,
			SnapshotID: strings.TrimSpace(info.SoulSnapshotID),
			Digest:     strings.TrimSpace(info.SoulDigest),
		}
	}
	snapshot, err := store.GetSoulSnapshot(ctx, info.SoulSnapshotID)
	if err != nil {
		if isContextError(err) {
			return contract.AgentSoulSectionPayload{}
		}
		return contract.AgentSoulSectionPayload{
			Present:    true,
			Active:     true,
			Valid:      false,
			SnapshotID: strings.TrimSpace(info.SoulSnapshotID),
			Digest:     strings.TrimSpace(info.SoulDigest),
		}
	}
	return soulPayloadFromSnapshot(&snapshot)
}

func startupSessionPayload(startup session.StartupPromptContext) contract.AgentSessionPayload {
	return contract.AgentSessionPayload{
		ID:        strings.TrimSpace(startup.SessionID),
		Name:      strings.TrimSpace(startup.SessionName),
		Type:      startup.SessionType,
		State:     session.StateStarting,
		Channel:   strings.TrimSpace(startup.Channel),
		CreatedAt: startup.CreatedAt.UTC(),
		UpdatedAt: startup.UpdatedAt.UTC(),
	}
}

func sessionPayload(info *session.Info) contract.AgentSessionPayload {
	if info == nil {
		return contract.AgentSessionPayload{}
	}
	return contract.AgentSessionPayload{
		ID:        strings.TrimSpace(info.ID),
		Name:      strings.TrimSpace(info.Name),
		Type:      info.Type,
		State:     info.State,
		Channel:   strings.TrimSpace(info.Channel),
		Lineage:   contract.SessionLineagePayloadFromStore(info.Lineage),
		CreatedAt: info.CreatedAt.UTC(),
		UpdatedAt: info.UpdatedAt.UTC(),
	}
}

func soulPayloadFromSnapshot(snapshot *soul.Snapshot) contract.AgentSoulSectionPayload {
	if snapshot == nil || strings.TrimSpace(snapshot.ID) == "" {
		return contract.AgentSoulSectionPayload{}
	}
	profile, err := snapshot.ProfileEnvelope()
	if err != nil {
		return contract.AgentSoulSectionPayload{
			Present:    true,
			Active:     false,
			Valid:      false,
			SnapshotID: strings.TrimSpace(snapshot.ID),
			Digest:     strings.TrimSpace(snapshot.Digest),
		}
	}
	return contract.AgentSoulSectionPayload{
		Enabled:          profile.Compact.Enabled,
		Present:          profile.Compact.Present,
		Active:           profile.Compact.Active,
		Valid:            profile.Valid,
		ValidationStatus: soulValidationStatus(profile.Present, profile.Active, profile.Valid),
		SnapshotID:       strings.TrimSpace(snapshot.ID),
		Digest:           firstTrimmed(profile.Compact.Digest, snapshot.Digest),
		ConfigDigest:     strings.TrimSpace(profile.ConfigProvenance.Digest),
		SourcePath:       firstTrimmed(profile.Compact.SourcePath, snapshot.SourcePath),
		Role:             strings.TrimSpace(profile.Compact.Role),
		Tone:             append([]string(nil), profile.Compact.Tone...),
		Principles:       append([]string(nil), profile.Compact.Principles...),
		Truncated:        profile.Compact.Truncated || snapshot.Truncated,
		MaxBytes:         profile.Compact.MaxBytes,
		MaxBodyBytes:     profile.Compact.MaxBodyBytes,
	}
}

func soulValidationStatus(present bool, active bool, valid bool) contract.AuthoredValidationStatus {
	switch {
	case !present:
		return contract.AuthoredValidationMissing
	case !valid:
		return contract.AuthoredValidationInvalid
	case !active:
		return contract.AuthoredValidationInactive
	default:
		return contract.AuthoredValidationValid
	}
}

func workspacePayload(
	workspaceSnapshot *workspacepkg.ResolvedWorkspace,
	workspaceID string,
	rootDir string,
) contract.AgentWorkspacePayload {
	if workspaceSnapshot != nil {
		return contract.AgentWorkspacePayload{
			ID:      strings.TrimSpace(workspaceSnapshot.ID),
			Name:    strings.TrimSpace(workspaceSnapshot.Name),
			RootDir: strings.TrimSpace(workspaceSnapshot.RootDir),
		}
	}
	return contract.AgentWorkspacePayload{
		ID:      strings.TrimSpace(workspaceID),
		RootDir: strings.TrimSpace(rootDir),
	}
}

func workspaceID(workspace *workspacepkg.ResolvedWorkspace) string {
	if workspace == nil {
		return ""
	}
	return strings.TrimSpace(workspace.ID)
}

func workspaceRoot(workspace *workspacepkg.ResolvedWorkspace) string {
	if workspace == nil {
		return ""
	}
	return strings.TrimSpace(workspace.RootDir)
}

func taskReferencePayload(taskRecord taskpkg.Task) *contract.TaskReferencePayload {
	return &contract.TaskReferencePayload{
		ID:          strings.TrimSpace(taskRecord.ID),
		Identifier:  strings.TrimSpace(taskRecord.Identifier),
		Title:       strings.TrimSpace(taskRecord.Title),
		Status:      taskRecord.Status,
		Priority:    taskRecord.Priority,
		Owner:       cloneOwnership(taskRecord.Owner),
		Scope:       taskRecord.Scope,
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
	}
}

func coordinationChannelPayload(taskRecord taskpkg.Task, run taskpkg.Run) contract.CoordinationChannelPayload {
	metadata := runMetadata(run.Metadata)
	channelID := firstTrimmed(run.CoordinationChannelID, metadata["coordination_channel_id"], run.NetworkChannel)
	channelName := firstTrimmed(run.NetworkChannel, channelID)
	lastActivity := latestTime(
		run.QueuedAt,
		run.ClaimedAt,
		run.StartedAt,
		taskRecord.UpdatedAt,
	)
	return contract.NormalizeCoordinationChannelPayload(contract.CoordinationChannelPayload{
		ID:          channelID,
		Channel:     channelName,
		DisplayName: firstTrimmed(channelName, channelID),
		Purpose:     "task_run_coordination",
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
		TaskID:      strings.TrimSpace(taskRecord.ID),
		RunID:       strings.TrimSpace(run.ID),
		WorkflowID:  firstTrimmed(metadata["workflow_id"]),
		LastActivityAt: optionalTimePtr(
			lastActivity,
		),
	})
}

func inboxSummary(
	envelopes []network.Envelope,
	limit int,
	activeCoordinationChannelID string,
) contract.AgentInboxSummaryPayload {
	items := make([]contract.AgentInboxItemPayload, 0, len(envelopes))
	for _, envelope := range envelopes {
		metadata, ok := coordinationMetadataFromEnvelope(envelope)
		if !ok {
			continue
		}
		if active := strings.TrimSpace(activeCoordinationChannelID); active != "" &&
			strings.TrimSpace(metadata.CoordinationChannelID) != active {
			continue
		}
		items = append(items, contract.AgentInboxItemPayload{
			MessageID: strings.TrimSpace(envelope.ID),
			ChannelID: firstTrimmed(
				metadata.CoordinationChannelID,
				envelope.Channel,
			),
			Kind:      metadata.MessageKind,
			Metadata:  metadata,
			Preview:   envelopePreview(envelope),
			Timestamp: envelopeTimestamp(envelope),
		})
	}
	slices.SortStableFunc(items, func(left, right contract.AgentInboxItemPayload) int {
		if !left.Timestamp.Equal(right.Timestamp) {
			if left.Timestamp.After(right.Timestamp) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.MessageID, right.MessageID)
	})

	return contract.AgentInboxSummaryPayload{
		Section:     sectionMeta(len(items), limit),
		UnreadCount: len(items),
		Items:       boundedInbox(items, limit),
	}
}

func peerRoster(peers []network.PeerInfo, sessionID string, limit int) contract.AgentPeerRosterPayload {
	roster := make([]contract.AgentPeerSummaryPayload, 0, len(peers))
	for _, peer := range peers {
		if peer.SessionID != nil && strings.TrimSpace(*peer.SessionID) == strings.TrimSpace(sessionID) {
			continue
		}
		roster = append(roster, contract.AgentPeerSummaryPayload{
			PeerID:       strings.TrimSpace(peer.PeerID),
			SessionID:    peerSessionID(peer),
			DisplayName:  peerDisplayName(peer),
			ChannelID:    strings.TrimSpace(peer.Channel),
			Capabilities: peerCapabilities(peer),
		})
	}
	slices.SortStableFunc(roster, func(left, right contract.AgentPeerSummaryPayload) int {
		if left.ChannelID != right.ChannelID {
			return strings.Compare(left.ChannelID, right.ChannelID)
		}
		return strings.Compare(left.PeerID, right.PeerID)
	})
	return contract.AgentPeerRosterPayload{
		Section: sectionMeta(len(roster), limit),
		Peers:   boundedPeers(roster, limit),
	}
}

func selectActiveRun(runs []taskpkg.Run) (taskpkg.Run, bool) {
	active := make([]taskpkg.Run, 0, len(runs))
	for _, run := range runs {
		if activeRunRank(run.Status) < 0 {
			continue
		}
		active = append(active, run)
	}
	if len(active) == 0 {
		return taskpkg.Run{}, false
	}
	slices.SortStableFunc(active, func(left, right taskpkg.Run) int {
		leftRank := activeRunRank(left.Status)
		rightRank := activeRunRank(right.Status)
		if leftRank != rightRank {
			return leftRank - rightRank
		}
		if leftTime, rightTime := runActivityTime(left), runActivityTime(right); !leftTime.Equal(rightTime) {
			if leftTime.After(rightTime) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.ID, right.ID)
	})
	return active[0], true
}

func activeRunRank(status taskpkg.RunStatus) int {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusRunning:
		return 0
	case taskpkg.TaskRunStatusStarting:
		return 1
	case taskpkg.TaskRunStatusClaimed:
		return 2
	case taskpkg.TaskRunStatusQueued:
		return 3
	default:
		return -1
	}
}

func runActivityTime(run taskpkg.Run) time.Time {
	return latestTime(run.QueuedAt, run.ClaimedAt, run.StartedAt, run.EndedAt)
}

func runMetadata(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	values := make(map[string]string, len(decoded))
	for key, rawValue := range decoded {
		var stringValue string
		if err := json.Unmarshal(rawValue, &stringValue); err == nil {
			values[strings.TrimSpace(key)] = strings.TrimSpace(stringValue)
		}
	}
	return values
}

func coordinationMetadataFromEnvelope(
	envelope network.Envelope,
) (contract.CoordinationMessageMetadataPayload, bool) {
	for _, key := range []string{"coordination", "coordination_metadata", "agh_coordination", "metadata"} {
		if raw, ok := envelope.Ext[key]; ok {
			if metadata, decodeOK := decodeCoordinationMetadata(raw); decodeOK {
				return metadata, true
			}
		}
	}
	if len(envelope.Ext) > 0 {
		raw, err := json.Marshal(envelope.Ext)
		if err == nil {
			if metadata, ok := decodeCoordinationMetadata(raw); ok {
				return metadata, true
			}
		}
	}
	return contract.CoordinationMessageMetadataPayload{}, false
}

func decodeCoordinationMetadata(raw json.RawMessage) (contract.CoordinationMessageMetadataPayload, bool) {
	var metadata contract.CoordinationMessageMetadataPayload
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return contract.CoordinationMessageMetadataPayload{}, false
	}
	return metadata, true
}

func envelopePreview(envelope network.Envelope) string {
	body, err := envelope.DecodeBody()
	if err != nil {
		return ""
	}
	var preview string
	switch typed := body.(type) {
	case network.SayBody:
		preview = typed.Text
	case network.TraceBody:
		preview = typed.Message
	case network.CapabilityBody:
		preview = typed.Capability.Summary
	case network.ReceiptBody:
		if typed.Detail != nil {
			preview = *typed.Detail
		}
	case network.GreetBody:
		preview = typed.Summary
	}
	return truncateRunes(singleLine(preview), inboxPreviewLimit)
}

func envelopeTimestamp(envelope network.Envelope) time.Time {
	if envelope.TS <= 0 {
		return time.Time{}
	}
	return time.Unix(envelope.TS, 0).UTC()
}

func peerSessionID(peer network.PeerInfo) string {
	if peer.SessionID == nil {
		return ""
	}
	return strings.TrimSpace(*peer.SessionID)
}

func peerDisplayName(peer network.PeerInfo) string {
	if peer.PeerCard.DisplayName != nil {
		if display := strings.TrimSpace(*peer.PeerCard.DisplayName); display != "" {
			return display
		}
	}
	return strings.TrimSpace(peer.PeerID)
}

func peerCapabilities(peer network.PeerInfo) []string {
	values := make([]string, 0, len(peer.PeerCard.Capabilities)+len(peer.CapabilityCatalog))
	seen := make(map[string]struct{}, cap(values))
	for _, capability := range peer.PeerCard.Capabilities {
		addStringSet(&values, seen, capability)
	}
	if peer.CapabilityCatalogKnown {
		for _, capability := range peer.CapabilityCatalog {
			addStringSet(&values, seen, capability.ID)
		}
	}
	slices.Sort(values)
	return values
}

func addStringSet(values *[]string, seen map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	if _, exists := seen[trimmed]; exists {
		return
	}
	seen[trimmed] = struct{}{}
	*values = append(*values, trimmed)
}

func sectionMeta(total int, limit int) contract.AgentContextSectionMetaPayload {
	normalizedLimit := limit
	if normalizedLimit <= 0 {
		normalizedLimit = DefaultSectionLimit
	}
	return contract.AgentContextSectionMetaPayload{
		Limit:     normalizedLimit,
		Returned:  min(total, normalizedLimit),
		Truncated: total > normalizedLimit,
	}
}

func emptySectionMeta(limit int) contract.AgentContextSectionMetaPayload {
	return sectionMeta(0, limit)
}

func boundedCapabilities(
	values []contract.AgentCapabilityPayload,
	limit int,
) []contract.AgentCapabilityPayload {
	if len(values) == 0 {
		return []contract.AgentCapabilityPayload{}
	}
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func boundedInbox(values []contract.AgentInboxItemPayload, limit int) []contract.AgentInboxItemPayload {
	if len(values) == 0 {
		return []contract.AgentInboxItemPayload{}
	}
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func boundedPeers(values []contract.AgentPeerSummaryPayload, limit int) []contract.AgentPeerSummaryPayload {
	if len(values) == 0 {
		return []contract.AgentPeerSummaryPayload{}
	}
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func cloneActorIdentity(value *taskpkg.ActorIdentity) *taskpkg.ActorIdentity {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneOwnership(value *taskpkg.Ownership) *taskpkg.Ownership {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func coordinationChannelID(context contract.AgentCoordinationChannelContextPayload) string {
	if context.Channel == nil {
		return ""
	}
	return strings.TrimSpace(context.Channel.ID)
}

func latestTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		value = value.UTC()
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}
	return latest
}

func optionalTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	clone := value.UTC()
	return &clone
}

func firstTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("situation: context is required")
	}
	return ctx.Err()
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func singleLine(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	return strings.Join(fields, " ")
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || value == "" || utf8.RuneCountInString(value) <= limit {
		return value
	}
	if limit <= len("...") {
		return strings.Repeat(".", limit)
	}
	var builder strings.Builder
	builder.Grow(len(value))
	count := 0
	for _, r := range value {
		if count == limit-len("...") {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return strings.TrimSpace(builder.String()) + "..."
}
