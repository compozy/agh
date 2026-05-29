package task

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DefaultTaskStarvationAge is the queued age past which a claimable run is treated as
// starved by the scheduler convergence backstop and the scheduler status surface.
const DefaultTaskStarvationAge = 2 * time.Minute

// WithStarvationAge overrides the queued-age threshold used by scheduler-status starvation
// counts so the status surface agrees with the scheduler's own threshold.
func WithStarvationAge(age time.Duration) Option {
	return func(opts *managerOptions) {
		opts.starvationAge = age
	}
}

type runStarvedPayload struct {
	QueuedAt            time.Time `json:"queued_at,omitzero"`
	QueuedAgeMS         int64     `json:"queued_age_ms"`
	CoordinationChannel string    `json:"coordination_channel,omitempty"`
}

type runNeedsAttentionPayload struct {
	PreviousStatus      RunStatus `json:"previous_status"`
	Status              RunStatus `json:"status"`
	Diagnostic          string    `json:"diagnostic,omitempty"`
	QueuedAt            time.Time `json:"queued_at,omitzero"`
	CoordinationChannel string    `json:"coordination_channel,omitempty"`
}

// RecordRunStarved emits the canonical task.run_starved event for one starved queued run.
// The scheduler stays observational; run-event authority remains in the task service.
func (m *Service) RecordRunStarved(
	ctx context.Context,
	runID string,
	queuedAt time.Time,
	age time.Duration,
	actor ActorContext,
) error {
	run, err := m.store.GetTaskRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	return m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunStarved, actor, runStarvedPayload{
		QueuedAt:            queuedAt,
		QueuedAgeMS:         age.Milliseconds(),
		CoordinationChannel: run.CoordinationChannelID,
	})
}

// MarkRunNeedsAttention transitions a queued run to needs_attention via a CAS store mutation
// and records the canonical event. It is idempotent: a run already in needs_attention is
// returned unchanged. The diagnostic must never embed a raw claim token.
func (m *Service) MarkRunNeedsAttention(
	ctx context.Context,
	runID string,
	diagnostic string,
	actor ActorContext,
) (Run, error) {
	run, err := m.store.GetTaskRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return Run{}, err
	}
	if run.Status.Normalize() == TaskRunStatusNeedsAttention {
		return run, nil
	}
	if run.Status.Normalize() != TaskRunStatusQueued {
		return Run{}, fmt.Errorf(
			"%w: run %q is %s; only queued runs can be marked needs_attention",
			ErrInvalidStatusTransition,
			run.ID,
			run.Status.Normalize(),
		)
	}
	diagnostic = strings.TrimSpace(diagnostic)
	if strings.Contains(diagnostic, "agh_claim_") {
		return Run{}, fmt.Errorf("task: needs_attention diagnostic must not embed a claim token")
	}
	updated, err := m.store.MarkTaskRunNeedsAttention(ctx, run.ID, diagnostic)
	if err != nil {
		return Run{}, err
	}
	if err := m.recordTaskEvent(
		ctx,
		updated.TaskID,
		updated.ID,
		taskEventRunNeedsAttention,
		actor,
		runNeedsAttentionPayload{
			PreviousStatus:      TaskRunStatusQueued,
			Status:              updated.Status.Normalize(),
			Diagnostic:          diagnostic,
			QueuedAt:            updated.QueuedAt,
			CoordinationChannel: updated.CoordinationChannelID,
		},
	); err != nil {
		return Run{}, err
	}
	return updated, nil
}
