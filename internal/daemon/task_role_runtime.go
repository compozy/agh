package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	// defaultStarvationWorkerTTL bounds a capability-matched starvation worker's lifetime. The
	// spawn reaper releases its leases and stops it past this deadline so a worker that claims
	// nothing cannot pile up; released work re-queues and re-escalates from the durable budget.
	defaultStarvationWorkerTTL = 15 * time.Minute
	// defaultStarvationMaxActivePerWorkspace is advisory metadata on the spawn budget; the real
	// per-(agent, channel, scope) cap is the role-session dedup in activeRoleSession.
	defaultStarvationMaxActivePerWorkspace = 3
)

const (
	taskRoleRuntimeTaskIDKey      = "task_id"
	taskRoleRuntimeWorkspaceIDKey = "workspace_id"
)

const taskRoleActivationReasonRunEnqueued = "task_run_enqueued"
const taskRoleActivationReasonRecovery = "recovery"

type taskRoleStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	GetExecutionProfile(ctx context.Context, taskID string) (taskpkg.ExecutionProfile, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []taskpkg.RunStatus) ([]taskpkg.Run, error)
}

type taskRoleSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
}

type taskRoleRuntime struct {
	mu                  sync.Mutex
	store               taskRoleStore
	sessions            taskRoleSessionManager
	globalWorkspacePath string
	logger              *slog.Logger
	now                 func() time.Time
}

type taskRoleActivation struct {
	TaskID        string
	RunID         string
	Scope         taskpkg.Scope
	WorkspaceID   string
	WorkspacePath string
	AgentName     string
	Provider      string
	Model         string
	Channel       string
	Title         string
	Profile       *taskpkg.ExecutionProfile
	Capabilities  []string
}

var _ taskRunEnqueuedObserver = (*taskRoleRuntime)(nil)

func newTaskRoleRuntime(
	store taskRoleStore,
	sessions taskRoleSessionManager,
	globalWorkspacePath string,
	logger *slog.Logger,
	now func() time.Time,
) (*taskRoleRuntime, error) {
	if store == nil {
		return nil, errors.New("daemon: task role runtime requires task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: task role runtime requires session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &taskRoleRuntime{
		store:               store,
		sessions:            sessions,
		globalWorkspacePath: strings.TrimSpace(globalWorkspacePath),
		logger:              logger,
		now:                 now,
	}, nil
}

func (r *taskRoleRuntime) OnTaskRunEnqueued(ctx context.Context, payload hookspkg.TaskRunEnqueuedPayload) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = context.WithoutCancel(ctx)
	}
	runID := strings.TrimSpace(payload.RunID)
	if runID == "" {
		r.logTaskRoleError("daemon: task role enqueue payload missing run id", nil, payload)
		return
	}
	run, err := r.store.GetTaskRun(ctx, runID)
	if err != nil {
		r.logTaskRoleError("daemon: load task run for role activation", err, payload)
		return
	}
	taskRecord, err := r.store.GetTask(ctx, run.TaskID)
	if err != nil {
		r.logTaskRoleError("daemon: load task for role activation", err, payload)
		return
	}
	if err := r.activateRun(ctx, taskRecord, run, taskRoleActivationReasonRunEnqueued); err != nil {
		r.logTaskRoleError("daemon: activate task role session from enqueue", err, payload)
	}
}

func (r *taskRoleRuntime) Recover(ctx context.Context) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runs, err := r.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		r.logTaskRoleError("daemon: list queued task runs for role recovery", err, hookspkg.TaskRunEnqueuedPayload{})
		return
	}
	for _, run := range runs {
		taskRecord, err := r.store.GetTask(ctx, run.TaskID)
		if err != nil {
			r.logTaskRoleError("daemon: load task for role recovery", err, hookspkg.TaskRunEnqueuedPayload{
				TaskRunContext: hookspkg.TaskRunContext{RunID: run.ID, TaskID: run.TaskID},
			})
			continue
		}
		if err := r.activateRun(ctx, taskRecord, run, taskRoleActivationReasonRecovery); err != nil {
			r.logTaskRoleError(
				"daemon: recover task role session for queued run",
				err,
				hookspkg.TaskRunEnqueuedPayload{
					TaskRunContext: hookspkg.TaskRunContext{
						RunID:                 run.ID,
						TaskID:                run.TaskID,
						WorkspaceID:           taskRecord.WorkspaceID,
						CoordinationChannelID: run.CoordinationChannelID,
					},
				},
			)
		}
	}
}

func (r *taskRoleRuntime) activateRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
) error {
	if ctx == nil {
		return errors.New("daemon: task role activation context is required")
	}
	activation, ok, err := r.activationForRun(ctx, taskRecord, run)
	if err != nil || !ok {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.activeRoleSession(ctx, activation)
	if err != nil {
		return err
	}
	if existing != nil {
		r.logger.Info(
			"daemon: task role session already active",
			taskRoleRuntimeTaskIDKey, activation.TaskID,
			"run_id", activation.RunID,
			"agent_name", activation.AgentName,
			"channel", activation.Channel,
			"reason", reason,
		)
		return nil
	}

	info, err := r.startRoleSession(ctx, activation)
	if err != nil {
		return err
	}
	r.logger.Info(
		"daemon: task role session started",
		"session_id", info.ID,
		taskRoleRuntimeTaskIDKey, activation.TaskID,
		"run_id", activation.RunID,
		"agent_name", activation.AgentName,
		"channel", activation.Channel,
		"reason", reason,
	)
	return nil
}

func (r *taskRoleRuntime) activationForRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
) (taskRoleActivation, bool, error) {
	if run.Status.Normalize() != taskpkg.TaskRunStatusQueued {
		return taskRoleActivation{}, false, nil
	}
	switch taskRecord.Status.Normalize() {
	case taskpkg.TaskStatusDraft, taskpkg.TaskStatusBlocked, taskpkg.TaskStatusCanceled:
		return taskRoleActivation{}, false, nil
	default:
	}
	agentName, provider, model, profile, ok, err := r.workerTargetForRun(ctx, taskRecord)
	if err != nil || !ok {
		return taskRoleActivation{}, false, err
	}
	if agentName == "" {
		return taskRoleActivation{}, false, nil
	}

	activation := taskRoleActivation{
		TaskID:      strings.TrimSpace(taskRecord.ID),
		RunID:       strings.TrimSpace(run.ID),
		Scope:       taskRecord.Scope.Normalize(),
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
		AgentName:   agentName,
		Provider:    provider,
		Model:       model,
		Channel:     taskRunSessionChannel(run),
		Title:       strings.TrimSpace(taskRecord.Title),
		Profile:     profile,
		Capabilities: append(
			append([]string(nil), run.RequiredCapabilities...),
			profileRequiredWorkerCapabilities(profile)...,
		),
	}
	if err := r.applyActivationScope(&activation, taskRecord.ID); err != nil {
		return taskRoleActivation{}, false, err
	}
	return activation, true, nil
}

func (r *taskRoleRuntime) workerTargetForRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
) (agentName string, provider string, model string, profile *taskpkg.ExecutionProfile, ok bool, err error) {
	if taskRecord.Owner != nil && !taskRecord.Owner.IsZero() {
		owner := *taskRecord.Owner
		if owner.Kind.Normalize() != taskpkg.OwnerKindPool {
			return "", "", "", nil, false, nil
		}
		return strings.TrimSpace(owner.Ref), "", "", nil, true, nil
	}

	loaded, err := r.store.GetExecutionProfile(ctx, taskRecord.ID)
	if err != nil {
		if errors.Is(err, taskpkg.ErrExecutionProfileNotFound) {
			return "", "", "", nil, false, nil
		}
		return "", "", "", nil, false, err
	}
	worker := loaded.Worker
	if worker.Mode.Normalize() != taskpkg.WorkerModeSelect {
		return "", "", "", nil, false, nil
	}
	// The exact worker name wins; selector lists only provide fallback candidates.
	agentName = strings.TrimSpace(worker.AgentName)
	if agentName == "" && len(worker.PreferredAgentNames) > 0 {
		agentName = strings.TrimSpace(worker.PreferredAgentNames[0])
	}
	if agentName == "" && len(worker.AllowedAgentNames) == 1 {
		agentName = strings.TrimSpace(worker.AllowedAgentNames[0])
	}
	if agentName == "" {
		return "", "", "", nil, false, nil
	}
	return agentName, strings.TrimSpace(worker.Provider), strings.TrimSpace(worker.Model), &loaded, true, nil
}

func (r *taskRoleRuntime) applyActivationScope(activation *taskRoleActivation, taskID string) error {
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		if activation.WorkspaceID == "" {
			return fmt.Errorf("%w: workspace-scoped task %q has no workspace id", taskpkg.ErrValidation, taskID)
		}
	case taskpkg.ScopeGlobal:
		if r.globalWorkspacePath == "" {
			return errors.New("daemon: task role global workspace path is required")
		}
		activation.WorkspacePath = r.globalWorkspacePath
	default:
		return fmt.Errorf("%w: unsupported task scope %q for role activation", taskpkg.ErrValidation, activation.Scope)
	}
	return nil
}

func (r *taskRoleRuntime) activeRoleSession(
	ctx context.Context,
	activation taskRoleActivation,
) (*session.Info, error) {
	infos, err := r.sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list sessions for task role activation: %w", err)
	}
	for _, info := range infos {
		if taskRoleSessionMatches(info, activation) {
			return info, nil
		}
	}
	return nil, nil
}

func (r *taskRoleRuntime) startRoleSession(
	ctx context.Context,
	activation taskRoleActivation,
) (*session.Info, error) {
	opts, err := taskRoleCreateOpts(activation)
	if err != nil {
		return nil, err
	}
	return r.createRoleSession(ctx, opts)
}

func taskRoleCreateOpts(activation taskRoleActivation) (session.CreateOpts, error) {
	opts := session.CreateOpts{
		AgentName:     activation.AgentName,
		Provider:      activation.Provider,
		Model:         activation.Model,
		Name:          taskRoleSessionName(activation),
		Channel:       activation.Channel,
		PromptOverlay: taskRolePromptOverlay(activation),
		Type:          session.SessionTypeSystem,
	}
	applyTaskSessionSandboxProfile(&opts, activation.Profile)
	applyTaskSessionRuntimeProfile(&opts, activation.Profile)
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		opts.Workspace = activation.WorkspaceID
	case taskpkg.ScopeGlobal:
		opts.WorkspacePath = activation.WorkspacePath
	default:
		return session.CreateOpts{}, fmt.Errorf(
			"%w: unsupported task scope %q for role session start",
			taskpkg.ErrValidation,
			activation.Scope,
		)
	}
	return opts, nil
}

func (r *taskRoleRuntime) createRoleSession(
	ctx context.Context,
	opts session.CreateOpts,
) (*session.Info, error) {
	created, err := r.sessions.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("daemon: create task role session: %w", err)
	}
	if created == nil {
		return nil, errors.New("daemon: task role session create returned nil")
	}
	info := created.Info()
	if info == nil {
		return nil, errors.New("daemon: task role session create returned nil info")
	}
	return info, nil
}

// activateForStarvation spawns a capability-matched worker for a starved run that no agent has
// claimed. The worker self-claims via `agh task next`; the scheduler never claims. It carries a
// TTL + spawn budget so the reaper bounds its lifetime. Dedup on (agent, channel, scope) keeps the
// effective per-workspace cap at one active worker per role.
func (r *taskRoleRuntime) activateForStarvation(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	spawner starvationSpawner,
) error {
	if ctx == nil {
		return errors.New("daemon: starvation activation context is required")
	}
	agentName, err := r.resolveStarvationAgent(ctx, taskRecord, run, spawner)
	if err != nil {
		return err
	}
	activation, ok, err := r.starvationActivation(taskRecord, run, agentName)
	if err != nil || !ok {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.activeRoleSession(ctx, activation)
	if err != nil {
		return err
	}
	if existing != nil {
		r.logger.Info(
			"daemon: starvation worker already active",
			taskRoleRuntimeTaskIDKey, activation.TaskID,
			"run_id", activation.RunID,
			"agent_name", activation.AgentName,
			"channel", activation.Channel,
		)
		return nil
	}
	info, err := r.startStarvationSession(ctx, activation)
	if err != nil {
		return err
	}
	r.logger.Info(
		"daemon: starvation worker spawned",
		"session_id", info.ID,
		taskRoleRuntimeTaskIDKey, activation.TaskID,
		"run_id", activation.RunID,
		"agent_name", activation.AgentName,
		"channel", activation.Channel,
	)
	return nil
}

func (r *taskRoleRuntime) resolveStarvationAgent(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	spawner starvationSpawner,
) (string, error) {
	required := trimmedNonEmptyStrings(run.RequiredCapabilities)
	if len(required) == 0 &&
		taskRecord.Owner != nil && !taskRecord.Owner.IsZero() &&
		taskRecord.Owner.Kind.Normalize() == taskpkg.OwnerKindPool {
		if name := strings.TrimSpace(taskRecord.Owner.Ref); name != "" {
			return name, nil
		}
	}
	name, ok, err := spawner.resolveAgent(ctx, strings.TrimSpace(taskRecord.WorkspaceID), required)
	if err != nil {
		return "", fmt.Errorf("daemon: resolve starvation agent: %w", err)
	}
	if ok {
		return name, nil
	}
	return "", errStarvationSpawnUnresolvable
}

func (r *taskRoleRuntime) starvationActivation(
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	agentName string,
) (taskRoleActivation, bool, error) {
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return taskRoleActivation{}, false, nil
	}
	if run.Status.Normalize() != taskpkg.TaskRunStatusQueued {
		return taskRoleActivation{}, false, nil
	}
	switch taskRecord.Status.Normalize() {
	case taskpkg.TaskStatusDraft, taskpkg.TaskStatusBlocked, taskpkg.TaskStatusCanceled:
		return taskRoleActivation{}, false, nil
	default:
	}
	activation := taskRoleActivation{
		TaskID:      strings.TrimSpace(taskRecord.ID),
		RunID:       strings.TrimSpace(run.ID),
		Scope:       taskRecord.Scope.Normalize(),
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
		AgentName:   agentName,
		Channel:     taskRunSessionChannel(run),
		Title:       strings.TrimSpace(taskRecord.Title),
	}
	if err := r.applyActivationScope(&activation, taskRecord.ID); err != nil {
		return taskRoleActivation{}, false, err
	}
	return activation, true, nil
}

func (r *taskRoleRuntime) startStarvationSession(
	ctx context.Context,
	activation taskRoleActivation,
) (*session.Info, error) {
	opts, err := taskRoleCreateOpts(activation)
	if err != nil {
		return nil, err
	}
	ttlExpiresAt := r.now().UTC().Add(defaultStarvationWorkerTTL)
	opts.Lineage = &store.SessionLineage{
		SpawnRole:    session.DefaultSpawnRole,
		TTLExpiresAt: &ttlExpiresAt,
		SpawnBudget: store.SessionSpawnBudget{
			MaxChildren:           session.DefaultSpawnMaxChildren,
			MaxDepth:              session.DefaultSpawnMaxDepth,
			TTLSeconds:            int64(defaultStarvationWorkerTTL / time.Second),
			MaxActivePerWorkspace: defaultStarvationMaxActivePerWorkspace,
		},
	}
	return r.createRoleSession(ctx, opts)
}

func taskRoleSessionMatches(info *session.Info, activation taskRoleActivation) bool {
	if info == nil {
		return false
	}
	if !taskRoleSessionStateReusable(info.State) {
		return false
	}
	if strings.TrimSpace(info.AgentName) != activation.AgentName {
		return false
	}
	if strings.TrimSpace(info.Channel) != activation.Channel {
		return false
	}
	if strings.TrimSpace(info.Name) != taskRoleSessionName(activation) {
		return false
	}
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		return strings.TrimSpace(info.WorkspaceID) == activation.WorkspaceID
	case taskpkg.ScopeGlobal:
		return strings.TrimSpace(info.WorkspaceID) == "" &&
			strings.TrimSpace(info.Workspace) == activation.WorkspacePath
	default:
		return false
	}
}

func taskRoleSessionStateReusable(state session.State) bool {
	switch state {
	case session.StateStarting, session.StateActive:
		return true
	default:
		return false
	}
}

func taskRoleSessionName(activation taskRoleActivation) string {
	base := fmt.Sprintf("task-role:%s:%s", activation.AgentName, firstNonEmpty(activation.Channel, "default"))
	fingerprint := taskRoleProfileFingerprint(activation)
	if fingerprint == "" {
		return base
	}
	return base + ":" + fingerprint
}

func taskRoleProfileFingerprint(activation taskRoleActivation) string {
	if activation.Profile == nil {
		return ""
	}
	profile := activation.Profile
	parts := []string{
		strings.TrimSpace(activation.Provider),
		strings.TrimSpace(activation.Model),
		string(profile.Sandbox.Mode.Normalize()),
		strings.TrimSpace(profile.Sandbox.SandboxRef),
		string(profile.Runtime.Mode.Normalize()),
		strings.Join(uniqueNonEmptyStrings(profile.Worker.RequiredCapabilities), "\x1f"),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func taskRolePromptOverlay(activation taskRoleActivation) string {
	title := firstNonEmpty(activation.Title, activation.TaskID)
	channel := firstNonEmpty(activation.Channel, "default")
	claimCommand := taskRoleClaimCommand(activation.Capabilities)
	return fmt.Sprintf(`A queued AGH task run is assigned to this agent.

Task: %s
Run: %s
Coordination channel: %s

Use `+"`%s`"+` once to claim work for this session before changing files. Complete or fail the claimed run through the AGH task lease commands from this same session. Do not use `+"`agh task run claim`"+` for autonomous work.`,
		title,
		activation.RunID,
		channel,
		claimCommand,
	)
}

func taskRoleClaimCommand(capabilities []string) string {
	args := []string{"agh task next --wait -o json"}
	for _, capability := range uniqueNonEmptyStrings(capabilities) {
		args = append(args, "--capability "+shellQuoteSimple(capability))
	}
	return strings.Join(args, " ")
}

func profileRequiredWorkerCapabilities(profile *taskpkg.ExecutionProfile) []string {
	if profile == nil {
		return nil
	}
	return append([]string(nil), profile.Worker.RequiredCapabilities...)
}

func uniqueNonEmptyStrings(values []string) []string {
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

func shellQuoteSimple(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (r *taskRoleRuntime) logTaskRoleError(
	message string,
	err error,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil {
		return
	}
	args := []any{
		taskRoleRuntimeTaskIDKey, strings.TrimSpace(payload.TaskID),
		"run_id", strings.TrimSpace(payload.RunID),
		taskRoleRuntimeWorkspaceIDKey, strings.TrimSpace(payload.WorkspaceID),
		"coordination_channel_id", strings.TrimSpace(payload.CoordinationChannelID),
	}
	if err != nil {
		args = append(args, "error", err)
	}
	r.logger.Warn(message, args...)
}
