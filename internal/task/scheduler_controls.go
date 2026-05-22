package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	diagnosticitems "github.com/pedronauck/agh/internal/diagnostics"
	eventspkg "github.com/pedronauck/agh/internal/events"
	"github.com/pedronauck/agh/internal/store"
)

const (
	defaultSchedulerDrainTimeout = 60 * time.Second
	schedulerDrainPollInterval   = 500 * time.Millisecond
)

// DefaultSchedulerDrainTimeout keeps API transports from duplicating scheduler policy.
func DefaultSchedulerDrainTimeout() time.Duration {
	return defaultSchedulerDrainTimeout
}

type taskPauseStore interface {
	PauseTask(context.Context, PauseMutation) (Task, error)
	ResumeTask(context.Context, ResumeMutation) (Task, error)
}

type schedulerControlStore interface {
	GetSchedulerPause(context.Context) (SchedulerPauseState, error)
	SetSchedulerPaused(context.Context, string, string) (SchedulerPauseState, error)
	SetSchedulerResumed(context.Context) (SchedulerPauseState, error)
	CountActiveTaskRunClaims(context.Context) (int, error)
	CountQueuedTaskRuns(context.Context, bool) (int, error)
	CountPausedTasks(context.Context) (int, error)
	SchedulerBacklog(context.Context, SchedulerBacklogQuery) (SchedulerBacklog, error)
}

type eventSummaryStore interface {
	WriteEventSummary(context.Context, store.EventSummary) error
}

type taskPausePayload struct {
	Manual         bool            `json:"manual"`
	ActorKind      ActorKind       `json:"actor_kind"`
	ActorID        string          `json:"actor_id,omitempty"`
	PreviousPaused bool            `json:"previous_paused"`
	Paused         bool            `json:"paused"`
	Reason         string          `json:"reason,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type taskResumePayload struct {
	Manual         bool            `json:"manual"`
	ActorKind      ActorKind       `json:"actor_kind"`
	ActorID        string          `json:"actor_id,omitempty"`
	PreviousPaused bool            `json:"previous_paused"`
	Paused         bool            `json:"paused"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type schedulerEventPayload struct {
	Manual          bool      `json:"manual"`
	ActorKind       ActorKind `json:"actor_kind"`
	ActorID         string    `json:"actor_id,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	PreviousPaused  bool      `json:"previous_paused"`
	Paused          bool      `json:"paused"`
	ActiveClaims    int       `json:"active_claims,omitempty"`
	RemainingClaims int       `json:"remaining_claims,omitempty"`
	TimedOut        bool      `json:"timed_out,omitempty"`
	StartedAt       time.Time `json:"started_at,omitzero"`
	CompletedAt     time.Time `json:"completed_at,omitzero"`
}

// PauseTask marks one task as paused for future scheduler and claim eligibility.
func (m *Service) PauseTask(
	ctx context.Context,
	id string,
	req PauseTaskRequest,
	actor ActorContext,
) (*Task, error) {
	if err := m.requireForceRunAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := normalizePauseTaskRequest(req)
	if err != nil {
		return nil, err
	}
	pauseStore, err := m.requireTaskPauseStore()
	if err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(id)
	previous, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if isTerminalTaskStatus(previous.Status) {
		return nil, terminalTaskPauseError(previous)
	}
	if err := m.requireForceRunRate(actor, previous.ID); err != nil {
		return nil, err
	}
	updated, err := pauseStore.PauseTask(ctx, PauseMutation{
		TaskID:   previous.ID,
		Actor:    actorLabel(actor),
		Reason:   normalized.Reason,
		PausedAt: m.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, updated.ID, "", taskEventPaused, actor, taskPausePayload{
		Manual:         true,
		ActorKind:      actor.Actor.Kind.Normalize(),
		ActorID:        actor.Actor.Ref,
		PreviousPaused: previous.Paused,
		Paused:         updated.Paused,
		Reason:         normalized.Reason,
		Metadata:       cloneRawJSON(normalized.Metadata),
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ResumeTask clears one task pause for future scheduler and claim eligibility.
func (m *Service) ResumeTask(
	ctx context.Context,
	id string,
	req ResumeTaskRequest,
	actor ActorContext,
) (*Task, error) {
	if err := m.requireForceRunAuthority(actor); err != nil {
		return nil, err
	}
	normalized, err := normalizeResumeTaskRequest(req)
	if err != nil {
		return nil, err
	}
	pauseStore, err := m.requireTaskPauseStore()
	if err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(id)
	previous, err := m.store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := m.requireForceRunRate(actor, previous.ID); err != nil {
		return nil, err
	}
	updated, err := pauseStore.ResumeTask(ctx, ResumeMutation{TaskID: previous.ID, ResumedAt: m.now().UTC()})
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, updated.ID, "", taskEventResumed, actor, taskResumePayload{
		Manual:         true,
		ActorKind:      actor.Actor.Kind.Normalize(),
		ActorID:        actor.Actor.Ref,
		PreviousPaused: previous.Paused,
		Paused:         updated.Paused,
		Metadata:       cloneRawJSON(normalized.Metadata),
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// SchedulerStatus returns scheduler-wide pause state and live queue pressure.
func (m *Service) SchedulerStatus(ctx context.Context, actor ActorContext) (SchedulerStatus, error) {
	if err := requireReadAuthority(actor); err != nil {
		return SchedulerStatus{}, err
	}
	controlStore, err := m.requireSchedulerControlStore()
	if err != nil {
		return SchedulerStatus{}, err
	}
	return m.schedulerStatus(ctx, controlStore, m.now().UTC())
}

// PauseScheduler marks the daemon scheduler as paused for new dispatch and claims.
func (m *Service) PauseScheduler(
	ctx context.Context,
	req SchedulerPauseRequest,
	actor ActorContext,
) (SchedulerStatus, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return SchedulerStatus{}, err
	}
	controlStore, err := m.requireSchedulerControlStore()
	if err != nil {
		return SchedulerStatus{}, err
	}
	previous, err := controlStore.GetSchedulerPause(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	reason := strings.TrimSpace(req.Reason)
	state, err := controlStore.SetSchedulerPaused(ctx, actorLabel(actor), reason)
	if err != nil {
		return SchedulerStatus{}, err
	}
	status, err := m.schedulerStatus(ctx, controlStore, m.now().UTC())
	if err != nil {
		return SchedulerStatus{}, err
	}
	m.recordSchedulerEventBestEffort(ctx, eventspkg.SchedulerPaused, actor, schedulerEventPayload{
		Manual:         true,
		ActorKind:      actor.Actor.Kind.Normalize(),
		ActorID:        actor.Actor.Ref,
		Reason:         reason,
		PreviousPaused: previous.Paused,
		Paused:         state.Paused,
		ActiveClaims:   status.ActiveClaimCount,
	})
	return status, nil
}

// ResumeScheduler clears the daemon scheduler pause flag.
func (m *Service) ResumeScheduler(
	ctx context.Context,
	_ SchedulerResumeRequest,
	actor ActorContext,
) (SchedulerStatus, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return SchedulerStatus{}, err
	}
	controlStore, err := m.requireSchedulerControlStore()
	if err != nil {
		return SchedulerStatus{}, err
	}
	previous, err := controlStore.GetSchedulerPause(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	state, err := controlStore.SetSchedulerResumed(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	status, err := m.schedulerStatus(ctx, controlStore, m.now().UTC())
	if err != nil {
		return SchedulerStatus{}, err
	}
	m.recordSchedulerEventBestEffort(ctx, eventspkg.SchedulerResumed, actor, schedulerEventPayload{
		Manual:         true,
		ActorKind:      actor.Actor.Kind.Normalize(),
		ActorID:        actor.Actor.Ref,
		PreviousPaused: previous.Paused,
		Paused:         state.Paused,
		ActiveClaims:   status.ActiveClaimCount,
	})
	return status, nil
}

// DrainScheduler pauses the scheduler and waits until active claims reach zero or the timeout expires.
func (m *Service) DrainScheduler(
	ctx context.Context,
	req SchedulerDrainRequest,
	actor ActorContext,
) (SchedulerDrainResult, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return SchedulerDrainResult{}, err
	}
	controlStore, err := m.requireSchedulerControlStore()
	if err != nil {
		return SchedulerDrainResult{}, err
	}
	startedAt := m.now().UTC()
	timeout := req.Timeout
	if timeout < 0 {
		return SchedulerDrainResult{}, fmt.Errorf("%w: scheduler drain timeout must be non-negative", ErrValidation)
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "scheduler drain"
	}
	if _, err := controlStore.SetSchedulerPaused(ctx, actorLabel(actor), reason); err != nil {
		return SchedulerDrainResult{}, err
	}
	m.recordSchedulerEventBestEffort(ctx, eventspkg.SchedulerDrainStarted, actor, schedulerEventPayload{
		Manual:    true,
		ActorKind: actor.Actor.Kind.Normalize(),
		ActorID:   actor.Actor.Ref,
		Reason:    reason,
		Paused:    true,
		StartedAt: startedAt,
	})

	drainCtx, cancelDrain := detachedSchedulerDrainContext(ctx, timeout)
	defer cancelDrain()
	result, err := m.waitForSchedulerDrain(drainCtx, controlStore, timeout, startedAt)
	if err != nil {
		return SchedulerDrainResult{}, err
	}
	m.recordSchedulerEventBestEffort(ctx, eventspkg.SchedulerDrainCompleted, actor, schedulerEventPayload{
		Manual:          true,
		ActorKind:       actor.Actor.Kind.Normalize(),
		ActorID:         actor.Actor.Ref,
		Reason:          reason,
		Paused:          result.Status.Paused,
		RemainingClaims: result.RemainingClaims,
		TimedOut:        result.TimedOut,
		StartedAt:       result.StartedAt,
		CompletedAt:     result.CompletedAt,
	})
	return result, nil
}

// SchedulerBacklog returns queued scheduler backlog rows.
func (m *Service) SchedulerBacklog(
	ctx context.Context,
	query SchedulerBacklogQuery,
	actor ActorContext,
) (SchedulerBacklog, error) {
	if err := requireReadAuthority(actor); err != nil {
		return SchedulerBacklog{}, err
	}
	controlStore, err := m.requireSchedulerControlStore()
	if err != nil {
		return SchedulerBacklog{}, err
	}
	return controlStore.SchedulerBacklog(ctx, query)
}

func terminalTaskPauseError(record Task) error {
	status := record.Status.Normalize()
	item := diagnosticitems.NewItem(
		"task.pause."+diagnosticcontract.CodeTaskRunAlreadyTerminal,
		diagnosticcontract.CodeTaskRunAlreadyTerminal,
		diagnosticcontract.CategoryTask,
		"Task is already terminal",
		fmt.Sprintf("Task %s is %s and cannot be paused.", record.ID, status),
		diagnosticcontract.SeverityInfo,
		diagnosticcontract.FreshnessLive,
		diagnosticitems.WithSuggestedCommand(fmt.Sprintf("agh task inspect %s", record.ID)),
		diagnosticitems.WithEvidence(map[string]any{
			taskEvidenceIDKey: record.ID,
			"task_status":     string(status),
		}),
	)
	return diagnosticitems.NewStructuredError(item, ErrInvalidStatusTransition)
}

func detachedSchedulerDrainContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	drainWindow := timeout
	if drainWindow <= 0 {
		drainWindow = schedulerDrainPollInterval
	}
	return context.WithTimeout(context.WithoutCancel(ctx), drainWindow+schedulerDrainPollInterval)
}

func (m *Service) waitForSchedulerDrain(
	ctx context.Context,
	controlStore schedulerControlStore,
	timeout time.Duration,
	startedAt time.Time,
) (SchedulerDrainResult, error) {
	deadline := startedAt.Add(timeout)
	for {
		status, err := m.schedulerStatus(ctx, controlStore, m.now().UTC())
		if err != nil {
			return SchedulerDrainResult{}, err
		}
		if status.ActiveClaimCount == 0 {
			completedAt := m.now().UTC()
			return SchedulerDrainResult{
				Status:          status,
				Completed:       true,
				RemainingClaims: 0,
				StartedAt:       startedAt,
				CompletedAt:     completedAt,
			}, nil
		}
		if timeout == 0 || !m.now().UTC().Before(deadline) {
			completedAt := m.now().UTC()
			return SchedulerDrainResult{
				Status:          status,
				Completed:       false,
				TimedOut:        status.ActiveClaimCount > 0,
				RemainingClaims: status.ActiveClaimCount,
				StartedAt:       startedAt,
				CompletedAt:     completedAt,
			}, nil
		}
		wait := schedulerDrainPollInterval
		if remaining := deadline.Sub(m.now().UTC()); remaining > 0 && remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return SchedulerDrainResult{}, fmt.Errorf("task: drain scheduler: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (m *Service) schedulerStatus(
	ctx context.Context,
	controlStore schedulerControlStore,
	asOf time.Time,
) (SchedulerStatus, error) {
	state, err := controlStore.GetSchedulerPause(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	active, err := controlStore.CountActiveTaskRunClaims(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	queued, err := controlStore.CountQueuedTaskRuns(ctx, true)
	if err != nil {
		return SchedulerStatus{}, err
	}
	pausedTasks, err := controlStore.CountPausedTasks(ctx)
	if err != nil {
		return SchedulerStatus{}, err
	}
	return SchedulerStatus{
		Paused:           state.Paused,
		PausedBy:         state.PausedBy,
		PausedAt:         state.PausedAt,
		PausedReason:     state.Reason,
		ActiveClaimCount: active,
		QueuedRunCount:   queued,
		PausedTaskCount:  pausedTasks,
		AsOf:             asOf,
	}, nil
}

func (m *Service) requireTaskPauseStore() (taskPauseStore, error) {
	pauseStore, ok := m.store.(taskPauseStore)
	if !ok {
		return nil, errors.New("task: store does not support task pause controls")
	}
	return pauseStore, nil
}

func (m *Service) requireSchedulerControlStore() (schedulerControlStore, error) {
	controlStore, ok := m.store.(schedulerControlStore)
	if !ok {
		return nil, errors.New("task: store does not support scheduler controls")
	}
	return controlStore, nil
}

func (m *Service) recordSchedulerEventBestEffort(
	ctx context.Context,
	eventType string,
	actor ActorContext,
	payload schedulerEventPayload,
) {
	writer, ok := m.store.(eventSummaryStore)
	if !ok {
		return
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return
	}
	summary := store.EventSummary{
		Type:      eventType,
		Outcome:   string(eventspkg.OutcomeFor(eventType)),
		Content:   content,
		Summary:   schedulerEventSummary(eventType, payload),
		Timestamp: m.now().UTC(),
		EventCorrelation: store.EventCorrelation{
			ActorKind:       string(actor.Actor.Kind.Normalize()),
			ActorID:         actor.Actor.Ref,
			SchedulerReason: payload.Reason,
		},
	}
	eventCtx := context.Background()
	if ctx != nil {
		eventCtx = context.WithoutCancel(ctx)
	}
	if err := writer.WriteEventSummary(eventCtx, summary); err != nil {
		return
	}
}

func schedulerEventSummary(eventType string, payload schedulerEventPayload) string {
	switch eventType {
	case eventspkg.SchedulerPaused:
		return "scheduler paused"
	case eventspkg.SchedulerResumed:
		return "scheduler resumed"
	case eventspkg.SchedulerDrainStarted:
		return "scheduler drain started"
	case eventspkg.SchedulerDrainCompleted:
		if payload.TimedOut {
			return fmt.Sprintf("scheduler drain timed out with %d active claims", payload.RemainingClaims)
		}
		return "scheduler drain completed"
	default:
		return strings.TrimSpace(eventType)
	}
}

func normalizePauseTaskRequest(req PauseTaskRequest) (PauseTaskRequest, error) {
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return PauseTaskRequest{}, forceRunDiagnosticError(
			diagnosticcontract.CodeForceOpRequiresReason,
			"Task pause requires a reason",
			"Provide --reason so the pause audit event explains why new claims were stopped.",
			diagnosticcontract.SeverityError,
			"agh task pause <task-id> --reason \"incident response\"",
			nil,
			ErrForceOpRequiresReason,
		)
	}
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "task_pause.metadata"); err != nil {
		return PauseTaskRequest{}, err
	}
	return req, nil
}

func normalizeResumeTaskRequest(req ResumeTaskRequest) (ResumeTaskRequest, error) {
	req.Metadata = normalizeRawJSON(req.Metadata)
	if err := ValidateMetadataSize(req.Metadata, "task_resume.metadata"); err != nil {
		return ResumeTaskRequest{}, err
	}
	return req, nil
}

func actorLabel(actor ActorContext) string {
	kind := strings.TrimSpace(string(actor.Actor.Kind.Normalize()))
	ref := strings.TrimSpace(actor.Actor.Ref)
	if kind == "" {
		return ref
	}
	if ref == "" {
		return kind
	}
	return kind + ":" + ref
}
