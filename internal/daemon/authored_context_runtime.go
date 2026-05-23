package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/compozy/agh/internal/acp"
	core "github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/heartbeat"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/soul"
)

const (
	taskRecoverySessionMissing = "missing"
)

const (
	authoredContextRuntimeValidKey = "valid"
)

type authoredContextDeps struct {
	SoulAuthoring      core.SoulAuthoringService
	SoulRefresher      core.SoulRefresher
	HeartbeatAuthoring core.HeartbeatAuthoringService
	HeartbeatStatus    core.HeartbeatStatusService
	HeartbeatWake      core.HeartbeatWakeService
	SessionHealth      core.SessionHealthReader
	WakeEvents         core.HeartbeatWakeEventReader
}

type apiHeartbeatWakePrompter struct {
	ctx      context.Context
	sessions SessionManager
	logger   *slog.Logger
}

type heartbeatWakeHealthReader struct {
	reader heartbeat.SessionHealthReader
}

func authoredContextRuntimeDeps(ctx context.Context, state *bootState, sessions SessionManager) authoredContextDeps {
	var deps authoredContextDeps
	if state == nil {
		return deps
	}
	deps.SoulAuthoring = soulAuthoringServiceDependency(state.registry, state.logger)
	deps.SoulRefresher = soulRefresherDependency(sessions)
	deps.HeartbeatAuthoring = heartbeatAuthoringServiceDependency(state.registry, state.logger)
	deps.SessionHealth = sessionHealthReaderDependency(sessions)
	deps.HeartbeatStatus = heartbeatStatusServiceDependency(
		state.registry,
		sessions,
		agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		state.logger,
	)
	deps.HeartbeatWake = heartbeatWakeServiceDependency(
		ctx,
		state.registry,
		sessions,
		state.cfg.Agents.Heartbeat,
		state.logger,
	)
	deps.WakeEvents = heartbeatWakeEventReaderDependency(state.registry)
	deps.SoulAuthoring = hookSoulAuthoringService(deps.SoulAuthoring, state.notifier)
	deps.HeartbeatAuthoring = hookHeartbeatAuthoringService(deps.HeartbeatAuthoring, state.notifier)
	deps.HeartbeatStatus = hookHeartbeatStatusService(deps.HeartbeatStatus, state.notifier)
	deps.HeartbeatWake = hookHeartbeatWakeService(deps.HeartbeatWake, state.notifier)
	return deps
}

func soulAuthoringServiceDependency(store any, logger *slog.Logger) core.SoulAuthoringService {
	authoringStore, ok := store.(soul.AuthoringStore)
	if !ok {
		return nil
	}
	service, err := soul.NewManagedSoulAuthoringService(authoringStore)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create soul authoring service", err)
		return nil
	}
	return service
}

func soulRefresherDependency(sessions SessionManager) core.SoulRefresher {
	refresher, ok := sessions.(core.SoulRefresher)
	if !ok {
		return nil
	}
	return refresher
}

func heartbeatAuthoringServiceDependency(store any, logger *slog.Logger) core.HeartbeatAuthoringService {
	authoringStore, ok := store.(heartbeat.AuthoringStore)
	if !ok {
		return nil
	}
	service, err := heartbeat.NewManagedHeartbeatAuthoringService(authoringStore)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat authoring service", err)
		return nil
	}
	return service
}

func heartbeatStatusServiceDependency(
	store any,
	sessions SessionManager,
	policyResolver heartbeat.PolicyResolver,
	logger *slog.Logger,
) core.HeartbeatStatusService {
	statusStore, ok := store.(heartbeat.StatusStore)
	if !ok {
		return nil
	}
	options := make([]heartbeat.StatusOption, 0, 2)
	if reader, ok := sessions.(heartbeat.SessionHealthReader); ok {
		options = append(options, heartbeat.WithHeartbeatStatusSessionHealthReader(reader))
	}
	if policyResolver != nil {
		options = append(options, heartbeat.WithHeartbeatStatusPolicyResolver(policyResolver))
	}
	service, err := heartbeat.NewManagedHeartbeatStatusService(statusStore, options...)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat status service", err)
		return nil
	}
	return service
}

func heartbeatWakeServiceDependency(
	ctx context.Context,
	store any,
	sessions SessionManager,
	config aghconfig.HeartbeatConfig,
	logger *slog.Logger,
) core.HeartbeatWakeService {
	wakeStore, ok := store.(heartbeat.WakeStore)
	if !ok {
		return nil
	}
	healthReader, ok := sessions.(heartbeat.SessionHealthReader)
	if !ok {
		return nil
	}
	service, err := heartbeat.NewManagedWakeService(
		wakeStore,
		heartbeatWakeHealthReader{reader: healthReader},
		&apiHeartbeatWakePrompter{ctx: authoredContextLifecycle(ctx), sessions: sessions, logger: logger},
		config,
	)
	if err != nil {
		logAuthoredContextDependencyError(logger, "daemon: create heartbeat wake service", err)
		return nil
	}
	return service
}

func authoredContextLifecycle(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func sessionHealthReaderDependency(sessions SessionManager) core.SessionHealthReader {
	reader, ok := sessions.(core.SessionHealthReader)
	if !ok {
		return nil
	}
	return reader
}

func (r heartbeatWakeHealthReader) GetSessionHealth(
	ctx context.Context,
	sessionID string,
) (heartbeat.SessionHealth, error) {
	if r.reader == nil {
		return heartbeat.SessionHealth{}, heartbeat.ErrSessionHealthNotFound
	}
	health, err := r.reader.GetSessionHealth(ctx, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return heartbeat.SessionHealth{}, fmt.Errorf("%w: %s", heartbeat.ErrSessionHealthNotFound, sessionID)
		}
		return heartbeat.SessionHealth{}, err
	}
	return health, nil
}

func heartbeatWakeEventReaderDependency(store any) core.HeartbeatWakeEventReader {
	reader, ok := store.(core.HeartbeatWakeEventReader)
	if !ok {
		return nil
	}
	return reader
}

func (p *apiHeartbeatWakePrompter) PromptHeartbeatWake(
	ctx context.Context,
	req heartbeat.SyntheticWakePromptRequest,
) (heartbeat.SyntheticWakePromptResult, error) {
	if p == nil || p.sessions == nil {
		return heartbeat.SyntheticWakePromptResult{}, errors.New("daemon: api heartbeat prompter requires sessions")
	}
	synthetic, ok := p.sessions.(schedulerSyntheticPrompter)
	if !ok {
		return heartbeat.SyntheticWakePromptResult{}, errors.New(
			"daemon: api heartbeat prompter requires synthetic prompt support",
		)
	}
	events, err := synthetic.PromptSynthetic(ctx, req.SessionID, session.SyntheticPromptOpts{
		Message: req.Message,
		TurnID:  req.TurnID,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:               req.SyntheticCorrelation.TaskID,
			TaskRunID:            req.SyntheticCorrelation.TaskRunID,
			WorkflowID:           req.SyntheticCorrelation.WorkflowID,
			ClaimTokenHash:       req.SyntheticCorrelation.ClaimTokenHash,
			CoordinatorSessionID: req.SyntheticCorrelation.CoordinatorSessionID,
			Reason:               heartbeat.SyntheticReasonHeartbeatWake,
			Summary:              req.Summary,
			WakeEventID:          req.WakeEventID,
			PolicySnapshotID:     req.PolicySnapshotID,
			PolicyDigest:         req.PolicyDigest,
			ConfigDigest:         req.ConfigDigest,
		},
		SkipIfBusy: true,
	})
	if err != nil {
		if errors.Is(err, session.ErrPromptInProgress) {
			return heartbeat.SyntheticWakePromptResult{}, heartbeat.ErrSyntheticPromptBusy
		}
		return heartbeat.SyntheticWakePromptResult{}, err
	}
	p.drainEvents(req.SessionID, req.WakeEventID, events)
	return heartbeat.SyntheticWakePromptResult{SyntheticPromptID: req.TurnID}, nil
}

func (p *apiHeartbeatWakePrompter) drainEvents(sessionID string, wakeEventID string, events <-chan acp.AgentEvent) {
	if events == nil {
		return
	}
	drainCtx := context.Background()
	if p != nil && p.ctx != nil {
		drainCtx = p.ctx
	}
	go func() {
		for {
			select {
			case <-drainCtx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if event.Type == acp.EventTypeError && p != nil && p.logger != nil {
					p.logger.Warn(
						"api.heartbeat_wake.agent_error",
						"session_id", sessionID,
						"wake_event_id", wakeEventID,
					)
				}
			}
		}
	}()
}

func logAuthoredContextDependencyError(logger *slog.Logger, message string, err error) {
	if logger == nil || err == nil {
		return
	}
	logger.Warn(message, "error", err)
}

func hookSoulAuthoringService(
	next core.SoulAuthoringService,
	hooks *hooksNotifier,
) core.SoulAuthoringService {
	if next == nil {
		return nil
	}
	return hookedSoulAuthoringService{next: next, hooks: hooks}
}

type hookedSoulAuthoringService struct {
	next  core.SoulAuthoringService
	hooks *hooksNotifier
}

func (s hookedSoulAuthoringService) Validate(
	ctx context.Context,
	req soul.ValidateRequest,
) (soul.ValidateResult, error) {
	if s.next == nil {
		return soul.ValidateResult{}, errors.New("daemon: soul authoring service is not configured")
	}
	result, err := s.next.Validate(ctx, req)
	if err == nil {
		s.dispatchSoulSnapshotResolved(ctx, req.Target, &result)
	}
	return result, err
}

func (s hookedSoulAuthoringService) Put(ctx context.Context, req soul.PutRequest) (soul.MutationResult, error) {
	if s.next == nil {
		return soul.MutationResult{}, errors.New("daemon: soul authoring service is not configured")
	}
	result, err := s.next.Put(ctx, req)
	if err == nil {
		s.dispatchSoulMutationAfter(ctx, &result)
	}
	return result, err
}

func (s hookedSoulAuthoringService) Delete(
	ctx context.Context,
	req soul.DeleteRequest,
) (soul.MutationResult, error) {
	if s.next == nil {
		return soul.MutationResult{}, errors.New("daemon: soul authoring service is not configured")
	}
	result, err := s.next.Delete(ctx, req)
	if err == nil {
		s.dispatchSoulMutationAfter(ctx, &result)
	}
	return result, err
}

func (s hookedSoulAuthoringService) History(
	ctx context.Context,
	req soul.HistoryRequest,
) (soul.HistoryResult, error) {
	if s.next == nil {
		return soul.HistoryResult{}, errors.New("daemon: soul authoring service is not configured")
	}
	return s.next.History(ctx, req)
}

func (s hookedSoulAuthoringService) Rollback(
	ctx context.Context,
	req soul.RollbackRequest,
) (soul.MutationResult, error) {
	if s.next == nil {
		return soul.MutationResult{}, errors.New("daemon: soul authoring service is not configured")
	}
	result, err := s.next.Rollback(ctx, req)
	if err == nil {
		s.dispatchSoulMutationAfter(ctx, &result)
	}
	return result, err
}

func (s hookedSoulAuthoringService) dispatchSoulSnapshotResolved(
	ctx context.Context,
	target soul.AuthoringTarget,
	result *soul.ValidateResult,
) {
	if s.hooks == nil || result == nil {
		return
	}
	config, configErr := soul.NewConfigProvenance(target.Config, target.ConfigSource)
	if configErr != nil {
		logAuthoredContextDependencyError(s.hooks.logger, "daemon: resolve soul hook config provenance", configErr)
	}
	payload := hookspkg.AgentSoulSnapshotResolvedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentSoulSnapshotResolved,
			Timestamp: s.hooks.timestamp(),
		},
		AuthoredContextProvenance: hookspkg.AuthoredContextProvenance{
			WorkspaceID:      strings.TrimSpace(target.WorkspaceID),
			AgentName:        strings.TrimSpace(target.AgentName),
			SourcePath:       strings.TrimSpace(result.Soul.SourcePath),
			Digest:           strings.TrimSpace(result.Soul.Digest),
			ConfigDigest:     strings.TrimSpace(config.Digest),
			ValidationStatus: authoredValidationStatus(result.Soul.Present, result.Soul.Active, result.Soul.Valid),
			Valid:            result.Soul.Valid,
			Active:           result.Soul.Active,
			Reason:           firstSoulDiagnosticCode(result.Soul.Diagnostics),
		},
	}
	if _, err := s.hooks.DispatchAgentSoulSnapshotResolved(ctx, payload); err != nil {
		logAuthoredContextDependencyError(s.hooks.logger, "daemon: dispatch soul snapshot hook", err)
	}
}

func (s hookedSoulAuthoringService) dispatchSoulMutationAfter(ctx context.Context, result *soul.MutationResult) {
	if s.hooks == nil || result == nil {
		return
	}
	revision := result.Revision
	payload := hookspkg.AgentSoulMutationAfterPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentSoulMutationAfter,
			Timestamp: s.hooks.timestamp(),
		},
		AuthoredContextProvenance: hookspkg.AuthoredContextProvenance{
			WorkspaceID:      strings.TrimSpace(revision.WorkspaceID),
			AgentName:        strings.TrimSpace(revision.AgentName),
			SourcePath:       strings.TrimSpace(revision.SourcePath),
			SnapshotID:       strings.TrimSpace(result.Snapshot.ID),
			Digest:           strings.TrimSpace(result.Soul.Digest),
			ValidationStatus: authoredValidationStatus(result.Soul.Present, result.Soul.Active, result.Soul.Valid),
			Valid:            result.Soul.Valid,
			Active:           result.Soul.Active,
			Reason:           firstSoulDiagnosticCode(result.Soul.Diagnostics),
		},
		AuthoredMutationProvenance: hookspkg.AuthoredMutationProvenance{
			ActorKind:  strings.TrimSpace(revision.ActorKind),
			ActorID:    strings.TrimSpace(revision.ActorID),
			OriginKind: strings.TrimSpace(revision.OriginKind),
			OriginRef:  strings.TrimSpace(revision.OriginRef),
		},
		RevisionID:     strings.TrimSpace(revision.ID),
		Action:         string(revision.Action),
		PreviousDigest: strings.TrimSpace(revision.PreviousDigest),
		NewDigest:      strings.TrimSpace(revision.NewDigest),
	}
	if _, err := s.hooks.DispatchAgentSoulMutationAfter(ctx, payload); err != nil {
		logAuthoredContextDependencyError(s.hooks.logger, "daemon: dispatch soul mutation hook", err)
	}
}

func hookHeartbeatAuthoringService(
	next core.HeartbeatAuthoringService,
	hooks *hooksNotifier,
) core.HeartbeatAuthoringService {
	if next == nil {
		return nil
	}
	return hookedHeartbeatAuthoringService{next: next, hooks: hooks}
}

type hookedHeartbeatAuthoringService struct {
	next  core.HeartbeatAuthoringService
	hooks *hooksNotifier
}

func (s hookedHeartbeatAuthoringService) Validate(
	ctx context.Context,
	req heartbeat.ValidateRequest,
) (heartbeat.ValidateResult, error) {
	if s.next == nil {
		return heartbeat.ValidateResult{}, errors.New("daemon: heartbeat authoring service is not configured")
	}
	result, err := s.next.Validate(ctx, req)
	if err == nil {
		s.dispatchHeartbeatPolicyResolved(ctx, req.Target, &result.Policy, "")
	}
	return result, err
}

func (s hookedHeartbeatAuthoringService) Put(
	ctx context.Context,
	req heartbeat.PutRequest,
) (heartbeat.MutationResult, error) {
	if s.next == nil {
		return heartbeat.MutationResult{}, errors.New("daemon: heartbeat authoring service is not configured")
	}
	return s.next.Put(ctx, req)
}

func (s hookedHeartbeatAuthoringService) Delete(
	ctx context.Context,
	req heartbeat.DeleteRequest,
) (heartbeat.MutationResult, error) {
	if s.next == nil {
		return heartbeat.MutationResult{}, errors.New("daemon: heartbeat authoring service is not configured")
	}
	return s.next.Delete(ctx, req)
}

func (s hookedHeartbeatAuthoringService) History(
	ctx context.Context,
	req heartbeat.HistoryRequest,
) (heartbeat.HistoryResult, error) {
	if s.next == nil {
		return heartbeat.HistoryResult{}, errors.New("daemon: heartbeat authoring service is not configured")
	}
	return s.next.History(ctx, req)
}

func (s hookedHeartbeatAuthoringService) Rollback(
	ctx context.Context,
	req heartbeat.RollbackRequest,
) (heartbeat.MutationResult, error) {
	if s.next == nil {
		return heartbeat.MutationResult{}, errors.New("daemon: heartbeat authoring service is not configured")
	}
	return s.next.Rollback(ctx, req)
}

func (s hookedHeartbeatAuthoringService) dispatchHeartbeatPolicyResolved(
	ctx context.Context,
	target heartbeat.AuthoringTarget,
	policy *heartbeat.ResolvedPolicy,
	snapshotID string,
) {
	dispatchHeartbeatPolicyResolved(ctx, s.hooks, target.WorkspaceID, target.AgentName, policy, snapshotID)
}

func hookHeartbeatStatusService(
	next core.HeartbeatStatusService,
	hooks *hooksNotifier,
) core.HeartbeatStatusService {
	if next == nil {
		return nil
	}
	return hookedHeartbeatStatusService{next: next, hooks: hooks}
}

type hookedHeartbeatStatusService struct {
	next  core.HeartbeatStatusService
	hooks *hooksNotifier
}

func (s hookedHeartbeatStatusService) Inspect(
	ctx context.Context,
	req heartbeat.InspectRequest,
) (heartbeat.InspectResult, error) {
	if s.next == nil {
		return heartbeat.InspectResult{}, errors.New("daemon: heartbeat status service is not configured")
	}
	result, err := s.next.Inspect(ctx, req)
	if err == nil {
		snapshotID := ""
		if result.Snapshot != nil {
			snapshotID = result.Snapshot.ID
		}
		dispatchHeartbeatPolicyResolved(
			ctx,
			s.hooks,
			req.Target.WorkspaceID,
			result.AgentName,
			&result.Policy,
			snapshotID,
		)
	}
	return result, err
}

func (s hookedHeartbeatStatusService) Status(
	ctx context.Context,
	req heartbeat.StatusRequest,
) (heartbeat.StatusResult, error) {
	if s.next == nil {
		return heartbeat.StatusResult{}, errors.New("daemon: heartbeat status service is not configured")
	}
	result, err := s.next.Status(ctx, req)
	if err == nil {
		policy := heartbeat.ResolvedPolicy{
			Enabled:          result.Enabled,
			Present:          result.Present,
			Active:           result.Active,
			Valid:            result.Valid,
			SourcePath:       result.SourcePath,
			Digest:           result.Digest,
			ConfigDigest:     result.ConfigDigest,
			Summary:          result.Summary,
			ConfigProvenance: result.ConfigProvenance,
			Preferences:      result.Preferences,
			Diagnostics:      result.Diagnostics,
		}
		dispatchHeartbeatPolicyResolved(
			ctx,
			s.hooks,
			req.Target.WorkspaceID,
			result.AgentName,
			&policy,
			result.SnapshotID,
		)
	}
	return result, err
}

func hookHeartbeatWakeService(next core.HeartbeatWakeService, hooks *hooksNotifier) core.HeartbeatWakeService {
	if next == nil {
		return nil
	}
	return hookedHeartbeatWakeService{next: next, hooks: hooks}
}

type hookedHeartbeatWakeService struct {
	next  core.HeartbeatWakeService
	hooks *hooksNotifier
}

func (s hookedHeartbeatWakeService) Wake(
	ctx context.Context,
	req heartbeat.WakeRequest,
) (heartbeat.WakeDecision, error) {
	if s.next == nil {
		return heartbeat.WakeDecision{}, errors.New("daemon: heartbeat wake service is not configured")
	}
	s.dispatchWakeBefore(ctx, req)
	decision, err := s.next.Wake(ctx, req)
	if err == nil {
		s.dispatchWakeAfter(ctx, req, decision)
	}
	return decision, err
}

func (s hookedHeartbeatWakeService) dispatchWakeBefore(ctx context.Context, req heartbeat.WakeRequest) {
	if s.hooks == nil {
		return
	}
	payload := hookspkg.AgentHeartbeatWakeBeforePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentHeartbeatWakeBefore,
			Timestamp: s.hooks.timestamp(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:   strings.TrimSpace(req.SessionID),
			AgentName:   strings.TrimSpace(req.AgentName),
			WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		},
		Source: string(req.Source),
		DryRun: req.DryRun,
	}
	if _, err := s.hooks.DispatchAgentHeartbeatWakeBefore(ctx, payload); err != nil {
		logAuthoredContextDependencyError(s.hooks.logger, "daemon: dispatch heartbeat wake before hook", err)
	}
}

func (s hookedHeartbeatWakeService) dispatchWakeAfter(
	ctx context.Context,
	req heartbeat.WakeRequest,
	decision heartbeat.WakeDecision,
) {
	if s.hooks == nil {
		return
	}
	payload := hookspkg.AgentHeartbeatWakeAfterPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentHeartbeatWakeAfter,
			Timestamp: s.hooks.timestamp(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:   strings.TrimSpace(req.SessionID),
			AgentName:   strings.TrimSpace(req.AgentName),
			WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		},
		WakeEventID:       strings.TrimSpace(decision.WakeEventID),
		Result:            string(decision.Result),
		Reason:            string(decision.Reason),
		PolicySnapshotID:  strings.TrimSpace(decision.PolicySnapshotID),
		PolicyDigest:      strings.TrimSpace(decision.PolicyDigest),
		ConfigDigest:      strings.TrimSpace(decision.ConfigDigest),
		SyntheticPromptID: strings.TrimSpace(decision.SyntheticPromptID),
		Source:            string(req.Source),
	}
	if _, err := s.hooks.DispatchAgentHeartbeatWakeAfter(ctx, payload); err != nil {
		logAuthoredContextDependencyError(s.hooks.logger, "daemon: dispatch heartbeat wake after hook", err)
	}
}

func dispatchHeartbeatPolicyResolved(
	ctx context.Context,
	hooks *hooksNotifier,
	workspaceID string,
	agentName string,
	policy *heartbeat.ResolvedPolicy,
	snapshotID string,
) {
	if hooks == nil || policy == nil {
		return
	}
	payload := hookspkg.AgentHeartbeatPolicyResolvedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentHeartbeatPolicyResolved,
			Timestamp: hooks.timestamp(),
		},
		AuthoredContextProvenance: hookspkg.AuthoredContextProvenance{
			WorkspaceID:      strings.TrimSpace(workspaceID),
			AgentName:        strings.TrimSpace(agentName),
			SourcePath:       strings.TrimSpace(policy.SourcePath),
			SnapshotID:       strings.TrimSpace(snapshotID),
			Digest:           strings.TrimSpace(policy.Digest),
			ConfigDigest:     strings.TrimSpace(policy.ConfigDigest),
			ValidationStatus: authoredValidationStatus(policy.Present, policy.Active, policy.Valid),
			Valid:            policy.Valid,
			Active:           policy.Active,
			Reason:           firstHeartbeatDiagnosticCode(policy.Diagnostics),
		},
		Summary: strings.TrimSpace(policy.Summary),
	}
	if _, err := hooks.DispatchAgentHeartbeatPolicyResolved(ctx, payload); err != nil {
		logAuthoredContextDependencyError(hooks.logger, "daemon: dispatch heartbeat policy hook", err)
	}
}

func authoredValidationStatus(present bool, active bool, valid bool) string {
	switch {
	case !present:
		return taskRecoverySessionMissing
	case !valid:
		return "invalid"
	case !active:
		return "inactive"
	default:
		return authoredContextRuntimeValidKey
	}
}

func firstSoulDiagnosticCode(diagnostics []soul.Diagnostic) string {
	for _, diagnostic := range diagnostics {
		if code := strings.TrimSpace(diagnostic.Code); code != "" {
			return code
		}
	}
	return ""
}

func firstHeartbeatDiagnosticCode(diagnostics []heartbeat.Diagnostic) string {
	for _, diagnostic := range diagnostics {
		if code := strings.TrimSpace(diagnostic.Code); code != "" {
			return code
		}
	}
	return ""
}
