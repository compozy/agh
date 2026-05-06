package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	reviewRouterActorRef        = "review-router"
	reviewRouterOriginRef       = "daemon.review_router"
	reviewRouterNoRouteGuidance = "Update the task execution profile review selectors " +
		"or start an eligible reviewer session."
	reviewRouterNoRouteDeliveryPrefix = "review-router:no-route:"
)

type runReviewRequestedForwarder struct {
	mu     sync.RWMutex
	target taskpkg.RunReviewRequestedObserver
}

var _ taskpkg.RunReviewRequestedObserver = (*runReviewRequestedForwarder)(nil)

func newRunReviewRequestedForwarder() *runReviewRequestedForwarder {
	return &runReviewRequestedForwarder{}
}

func (f *runReviewRequestedForwarder) Set(target taskpkg.RunReviewRequestedObserver) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.target = target
}

func (f *runReviewRequestedForwarder) OnRunReviewRequested(
	ctx context.Context,
	notification *taskpkg.RunReviewRequestedNotification,
) {
	if f == nil {
		return
	}
	f.mu.RLock()
	target := f.target
	f.mu.RUnlock()
	if target == nil {
		return
	}
	target.OnRunReviewRequested(ctx, notification)
}

type reviewRouterTasks interface {
	GetExecutionProfile(
		ctx context.Context,
		taskID string,
		actor taskpkg.ActorContext,
	) (taskpkg.ExecutionProfile, error)
	BindRunReviewSession(
		ctx context.Context,
		req taskpkg.BindRunReviewSessionRequest,
		actor taskpkg.ActorContext,
	) (taskpkg.RunReviewBinding, error)
	RecordRunReview(
		ctx context.Context,
		req taskpkg.RecordRunReviewRequest,
		actor taskpkg.ActorContext,
	) (taskpkg.RunReviewResult, error)
}

type reviewRouterTaskStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
}

type reviewRouterSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type reviewRouterAgentResolver interface {
	ResolveAgent(name string, resolved *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)
}

type reviewRouterWorkspaceResolver interface {
	Resolve(ctx context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error)
}

type reviewRouter struct {
	tasks          reviewRouterTasks
	store          reviewRouterTaskStore
	sessions       reviewRouterSessionManager
	workspaces     reviewRouterWorkspaceResolver
	agents         reviewRouterAgentResolver
	contextOverlay taskSessionContextOverlay
	logger         *slog.Logger
	now            func() time.Time
}

var _ taskpkg.RunReviewRequestedObserver = (*reviewRouter)(nil)

type reviewRouterOption func(*reviewRouter)

func withReviewRouterTaskContextOverlay(overlay taskSessionContextOverlay) reviewRouterOption {
	return func(router *reviewRouter) {
		if router != nil {
			router.contextOverlay = overlay
		}
	}
}

func newReviewRouter(
	tasks reviewRouterTasks,
	store reviewRouterTaskStore,
	sessions reviewRouterSessionManager,
	workspaces reviewRouterWorkspaceResolver,
	agents reviewRouterAgentResolver,
	logger *slog.Logger,
	now func() time.Time,
	options ...reviewRouterOption,
) (*reviewRouter, error) {
	if tasks == nil {
		return nil, errors.New("daemon: review router requires task manager")
	}
	if store == nil {
		return nil, errors.New("daemon: review router requires task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: review router requires session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	router := &reviewRouter{
		tasks:      tasks,
		store:      store,
		sessions:   sessions,
		workspaces: workspaces,
		agents:     agents,
		logger:     logger,
		now:        now,
	}
	for _, option := range options {
		if option != nil {
			option(router)
		}
	}
	return router, nil
}

func (r *reviewRouter) OnRunReviewRequested(
	ctx context.Context,
	notification *taskpkg.RunReviewRequestedNotification,
) {
	if r == nil || notification == nil {
		return
	}
	ctx = detachDaemonOwnedContext(ctx)
	if strings.TrimSpace(notification.Review.ReviewID) == "" {
		return
	}
	routed, diagnostic, err := r.routeRunReview(ctx, notification)
	if err != nil {
		r.logger.Warn(
			"daemon: review router failed",
			"review_id", notification.Review.ReviewID,
			"task_id", notification.Review.TaskID,
			"run_id", notification.Review.RunID,
			"error", err,
		)
	}
	if routed || strings.TrimSpace(diagnostic) == "" {
		return
	}
	if err := r.recordNoRouteDiagnostic(ctx, notification.Review, diagnostic); err != nil {
		r.logger.Warn(
			"daemon: review router failed to record no-route diagnostic",
			"review_id", notification.Review.ReviewID,
			"task_id", notification.Review.TaskID,
			"run_id", notification.Review.RunID,
			"error", err,
		)
	}
}

func (r *reviewRouter) routeRunReview(
	ctx context.Context,
	notification *taskpkg.RunReviewRequestedNotification,
) (bool, string, error) {
	actor, err := taskpkg.DeriveDaemonActorContext(reviewRouterActorRef, reviewRouterOriginRef)
	if err != nil {
		return false, "", err
	}
	review := notification.Review
	taskRecord, err := r.store.GetTask(ctx, review.TaskID)
	if err != nil {
		return false, "", fmt.Errorf("daemon: review router load task %q: %w", review.TaskID, err)
	}
	run, err := r.store.GetTaskRun(ctx, review.RunID)
	if err != nil {
		return false, "", fmt.Errorf("daemon: review router load run %q: %w", review.RunID, err)
	}
	profile, err := r.tasks.GetExecutionProfile(ctx, taskRecord.ID, actor)
	if err != nil {
		return false, "", fmt.Errorf("daemon: review router load profile for task %q: %w", taskRecord.ID, err)
	}

	route, diagnostic, err := r.selectRoute(ctx, taskRecord, run, &profile)
	if err != nil || diagnostic != "" {
		return false, diagnostic, err
	}
	if route.info == nil && route.create == nil {
		return false, "no eligible reviewer route", nil
	}
	info := route.info
	if info == nil {
		created, err := r.sessions.Create(ctx, *route.create)
		if err != nil {
			return false, "reviewer session create failed: " + err.Error(), err
		}
		if created == nil || created.Info() == nil {
			return false, "reviewer session create returned no session info", nil
		}
		info = created.Info()
	}

	peerID := reviewRouterPeerID(info)
	if _, err := r.tasks.BindRunReviewSession(ctx, taskpkg.BindRunReviewSessionRequest{
		ReviewID:          review.ReviewID,
		SessionID:         info.ID,
		ReviewerAgentName: info.AgentName,
		ReviewerPeerID:    peerID,
		ReviewerChannelID: info.Channel,
	}, actor); err != nil {
		if route.create != nil {
			err = errors.Join(err, r.cleanupCreatedReviewerSession(ctx, info))
		}
		return false, "reviewer session binding failed: " + err.Error(), err
	}
	return true, "", nil
}

type reviewRoute struct {
	info   *session.Info
	create *session.CreateOpts
}

func (r *reviewRouter) selectRoute(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	profile *taskpkg.ExecutionProfile,
) (reviewRoute, string, error) {
	resolved, err := r.resolveWorkspace(ctx, taskRecord.WorkspaceID)
	if err != nil {
		return reviewRoute{}, "review workspace resolution failed: " + err.Error(), err
	}
	review := profile.Review
	original := r.originalWorkerIdentity(ctx, taskRecord.WorkspaceID, run)
	existing, err := r.selectExistingRoute(ctx, taskRecord, &review, original, resolved)
	if err != nil {
		return reviewRoute{}, "", err
	}
	if existing != nil {
		return reviewRoute{info: existing}, "", nil
	}

	create, diagnostic, err := r.createRoute(ctx, taskRecord, run, &review, original, resolved)
	if err != nil || diagnostic != "" {
		return reviewRoute{}, diagnostic, err
	}
	return reviewRoute{create: create}, "", nil
}

type originalWorkerIdentity struct {
	sessionID string
	agentName string
	peerID    string
}

func (r *reviewRouter) originalWorkerIdentity(
	ctx context.Context,
	workspaceID string,
	run taskpkg.Run,
) originalWorkerIdentity {
	identity := originalWorkerIdentity{sessionID: strings.TrimSpace(run.SessionID)}
	if identity.sessionID == "" && run.ClaimedBy != nil &&
		run.ClaimedBy.Kind.Normalize() == taskpkg.ActorKindAgentSession {
		identity.sessionID = strings.TrimSpace(run.ClaimedBy.Ref)
	}
	if identity.sessionID == "" {
		return identity
	}
	infos, err := r.sessions.ListAll(ctx)
	if err != nil {
		r.logger.Warn(
			"daemon: review router could not list sessions for original-worker exclusion",
			"session_id", identity.sessionID,
			"error", err,
		)
		return identity
	}
	for _, info := range infos {
		if info == nil || strings.TrimSpace(info.ID) != identity.sessionID {
			continue
		}
		if workspaceID != "" && strings.TrimSpace(info.WorkspaceID) != strings.TrimSpace(workspaceID) {
			continue
		}
		identity.agentName = strings.TrimSpace(info.AgentName)
		identity.peerID = reviewRouterPeerID(info)
		return identity
	}
	return identity
}

type existingReviewCandidate struct {
	info  *session.Info
	score int
}

func (r *reviewRouter) selectExistingRoute(
	ctx context.Context,
	taskRecord taskpkg.Task,
	review *taskpkg.ReviewProfile,
	original originalWorkerIdentity,
	resolved *workspacepkg.ResolvedWorkspace,
) (*session.Info, error) {
	infos, err := r.sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list reviewer sessions: %w", err)
	}
	candidates := make([]existingReviewCandidate, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		score, ok, err := r.existingCandidateScore(ctx, taskRecord, review, original, resolved, info)
		if err != nil {
			return nil, err
		}
		if ok {
			candidates = append(candidates, existingReviewCandidate{info: info, score: score})
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.SliceStable(candidates, func(i int, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return strings.TrimSpace(candidates[i].info.ID) < strings.TrimSpace(candidates[j].info.ID)
	})
	return candidates[0].info, nil
}

func (r *reviewRouter) existingCandidateScore(
	ctx context.Context,
	taskRecord taskpkg.Task,
	review *taskpkg.ReviewProfile,
	original originalWorkerIdentity,
	resolved *workspacepkg.ResolvedWorkspace,
	info *session.Info,
) (int, bool, error) {
	if info.State != session.StateActive {
		return 0, false, nil
	}
	if strings.TrimSpace(taskRecord.WorkspaceID) != "" &&
		strings.TrimSpace(info.WorkspaceID) != strings.TrimSpace(taskRecord.WorkspaceID) {
		return 0, false, nil
	}
	if r.isOriginalWorker(info, original) {
		return 0, false, nil
	}
	agentName := strings.TrimSpace(info.AgentName)
	channelID := strings.TrimSpace(info.Channel)
	peerID := reviewRouterPeerID(info)
	if !selectorAllows(review.AgentName, review.AllowedAgentNames, agentName) ||
		!selectorAllows("", review.AllowedChannelIDs, channelID) ||
		!selectorAllows("", review.AllowedPeerIDs, peerID) {
		return 0, false, nil
	}
	if ok, err := r.agentHasCapabilities(ctx, resolved, agentName, review.RequiredCapabilities); err != nil {
		if errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
			return 0, false, nil
		}
		return 0, false, err
	} else if !ok {
		return 0, false, nil
	}
	score := 0
	if slices.Contains(review.PreferredAgentNames, agentName) || strings.TrimSpace(review.AgentName) == agentName {
		score += 4
	}
	if slices.Contains(review.PreferredPeerIDs, peerID) {
		score += 3
	}
	if slices.Contains(review.PreferredChannelIDs, channelID) {
		score += 2
	}
	if ok, err := r.agentHasCapabilities(ctx, resolved, agentName, review.PreferredCapabilities); err != nil {
		if errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
			return score, true, nil
		}
		return 0, false, err
	} else if ok && len(review.PreferredCapabilities) > 0 {
		score++
	}
	return score, true, nil
}

func (r *reviewRouter) createRoute(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	review *taskpkg.ReviewProfile,
	original originalWorkerIdentity,
	resolved *workspacepkg.ResolvedWorkspace,
) (*session.CreateOpts, string, error) {
	if len(review.AllowedPeerIDs) > 0 {
		return nil, "review profile allows only explicit peers and no active eligible peer is available", nil
	}
	agentName, diagnostic, err := r.selectCreateAgent(ctx, review, original, resolved)
	if err != nil || diagnostic != "" {
		return nil, diagnostic, err
	}
	channelID := selectReviewChannel(taskRecord, run, review)
	promptOverlay := reviewRouterPromptOverlay(taskRecord.ID, run.ID)
	if r.contextOverlay != nil {
		taskOverlay, err := r.contextOverlay.TaskRunPromptOverlay(ctx, taskRecord, run, nil)
		if err != nil {
			return nil, "reviewer task context render failed: " + err.Error(), err
		}
		promptOverlay = joinPromptOverlays(taskOverlay, promptOverlay)
	}
	return &session.CreateOpts{
		Name:          reviewSessionName(taskRecord.ID),
		AgentName:     agentName,
		Provider:      review.Provider,
		Model:         review.Model,
		Workspace:     taskRecord.WorkspaceID,
		Channel:       channelID,
		Type:          session.SessionTypeSystem,
		PromptOverlay: promptOverlay,
	}, "", nil
}

func (r *reviewRouter) selectCreateAgent(
	ctx context.Context,
	review *taskpkg.ReviewProfile,
	_ originalWorkerIdentity,
	resolved *workspacepkg.ResolvedWorkspace,
) (string, string, error) {
	candidates := reviewCreateAgentCandidates(review, resolved)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		ok, err := r.agentHasCapabilities(ctx, resolved, candidate, review.RequiredCapabilities)
		if err != nil {
			if errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
				continue
			}
			return "", "", err
		}
		if ok {
			return strings.TrimSpace(candidate), "", nil
		}
	}
	if len(review.RequiredCapabilities) > 0 {
		return "", "no reviewer agent satisfies required capabilities", nil
	}
	if strings.TrimSpace(review.AgentName) != "" || len(review.AllowedAgentNames) > 0 {
		return "", "review profile agent selectors exclude all eligible reviewer agents", nil
	}
	if resolved != nil {
		for _, agent := range sortedResolvedAgents(resolved) {
			if strings.TrimSpace(agent.Name) == "" {
				continue
			}
			return strings.TrimSpace(agent.Name), "", nil
		}
	}
	return "", "", nil
}

func reviewCreateAgentCandidates(
	review *taskpkg.ReviewProfile,
	resolved *workspacepkg.ResolvedWorkspace,
) []string {
	values := make([]string, 0)
	if strings.TrimSpace(review.AgentName) != "" {
		values = append(values, strings.TrimSpace(review.AgentName))
	}
	values = append(values, review.PreferredAgentNames...)
	values = append(values, review.AllowedAgentNames...)
	if len(values) == 0 && len(review.RequiredCapabilities) > 0 && resolved != nil {
		for _, agent := range sortedResolvedAgents(resolved) {
			values = append(values, agent.Name)
		}
	}
	return uniqueTrimmedStrings(values)
}

func sortedResolvedAgents(resolved *workspacepkg.ResolvedWorkspace) []aghconfig.AgentDef {
	if resolved == nil {
		return nil
	}
	agents := append([]aghconfig.AgentDef(nil), resolved.Agents...)
	sort.SliceStable(agents, func(i int, j int) bool {
		return strings.TrimSpace(agents[i].Name) < strings.TrimSpace(agents[j].Name)
	})
	return agents
}

func (r *reviewRouter) agentHasCapabilities(
	_ context.Context,
	resolved *workspacepkg.ResolvedWorkspace,
	agentName string,
	required []string,
) (bool, error) {
	if len(required) == 0 {
		return true, nil
	}
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return false, nil
	}
	agent, err := r.resolveAgent(agentName, resolved)
	if err != nil {
		return false, err
	}
	available := make(map[string]struct{})
	if agent.Capabilities != nil {
		for _, capability := range agent.Capabilities.Capabilities {
			id := strings.TrimSpace(capability.ID)
			if id != "" {
				available[id] = struct{}{}
			}
		}
	}
	for _, capability := range required {
		if _, ok := available[strings.TrimSpace(capability)]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func (r *reviewRouter) resolveAgent(
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	if r.agents != nil {
		return r.agents.ResolveAgent(agentName, resolved)
	}
	if resolved != nil {
		for _, agent := range resolved.Agents {
			if strings.TrimSpace(agent.Name) == strings.TrimSpace(agentName) {
				return agent, nil
			}
		}
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, agentName)
}

func (r *reviewRouter) resolveWorkspace(
	ctx context.Context,
	workspaceID string,
) (*workspacepkg.ResolvedWorkspace, error) {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" || r.workspaces == nil {
		return nil, nil
	}
	resolved, err := r.workspaces.Resolve(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}

func (r *reviewRouter) isOriginalWorker(info *session.Info, original originalWorkerIdentity) bool {
	if info == nil {
		return false
	}
	if strings.TrimSpace(original.sessionID) != "" && strings.TrimSpace(info.ID) == original.sessionID {
		return true
	}
	if strings.TrimSpace(original.peerID) != "" && reviewRouterPeerID(info) == original.peerID {
		return true
	}
	return false
}

func (r *reviewRouter) cleanupCreatedReviewerSession(ctx context.Context, info *session.Info) error {
	if info == nil || strings.TrimSpace(info.ID) == "" {
		return nil
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ctx != nil {
		stopCtx = context.WithoutCancel(stopCtx)
	}
	return r.sessions.StopWithCause(
		stopCtx,
		strings.TrimSpace(info.ID),
		session.CauseFailed,
		"review router bind failed",
	)
}

func detachDaemonOwnedContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func (r *reviewRouter) recordNoRouteDiagnostic(
	ctx context.Context,
	review taskpkg.RunReview,
	diagnostic string,
) error {
	actor, err := taskpkg.DeriveDaemonActorContext(reviewRouterActorRef, reviewRouterOriginRef)
	if err != nil {
		return err
	}
	confidence := 1.0
	_, err = r.tasks.RecordRunReview(ctx, taskpkg.RecordRunReviewRequest{
		ReviewID: review.ReviewID,
		RunID:    review.RunID,
		Verdict: taskpkg.RunReviewVerdict{
			Outcome:           taskpkg.RunReviewOutcomeBlocked,
			Confidence:        &confidence,
			Reason:            normalizeReviewRouterDiagnostic(diagnostic),
			DeliveryID:        reviewRouterNoRouteDeliveryPrefix + strings.TrimSpace(review.ReviewID),
			NextRoundGuidance: reviewRouterNoRouteGuidance,
		},
	}, actor)
	return err
}

func selectorAllows(exact string, allowed []string, value string) bool {
	value = strings.TrimSpace(value)
	if strings.TrimSpace(exact) != "" && value != strings.TrimSpace(exact) {
		return false
	}
	return len(allowed) == 0 || slices.Contains(allowed, value)
}

func selectReviewChannel(
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	review *taskpkg.ReviewProfile,
) string {
	for _, value := range review.PreferredChannelIDs {
		if len(review.AllowedChannelIDs) == 0 || slices.Contains(review.AllowedChannelIDs, value) {
			return value
		}
	}
	if len(review.AllowedChannelIDs) > 0 {
		return review.AllowedChannelIDs[0]
	}
	for _, value := range []string{run.CoordinationChannelID, run.NetworkChannel, taskRecord.NetworkChannel} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func reviewRouterPeerID(info *session.Info) string {
	if info == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(info.AgentName)) + "." + strings.TrimSpace(info.ID)
}

func reviewSessionName(taskID string) string {
	trimmed := strings.TrimSpace(taskID)
	if trimmed == "" {
		return "AGH Task Reviewer"
	}
	return "AGH Task Reviewer " + trimmed
}

func reviewRouterPromptOverlay(taskID string, runID string) string {
	return fmt.Sprintf(
		"Load the agh-task-reviewer skill. Review task %s run %s and submit the verdict with submit_run_review.",
		strings.TrimSpace(taskID),
		strings.TrimSpace(runID),
	)
}

func normalizeReviewRouterDiagnostic(diagnostic string) string {
	trimmed := strings.TrimSpace(diagnostic)
	if trimmed == "" {
		return "review router found no eligible reviewer"
	}
	return "review router found no eligible reviewer: " + trimmed
}

func uniqueTrimmedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
