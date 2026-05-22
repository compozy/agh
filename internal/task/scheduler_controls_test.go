package task

import (
	"context"
	"errors"
	"testing"
	"time"

	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	diagnosticitems "github.com/pedronauck/agh/internal/diagnostics"
	eventspkg "github.com/pedronauck/agh/internal/events"
	storepkg "github.com/pedronauck/agh/internal/store"
)

type schedulerControlTestStore struct {
	*inMemoryManagerStore
	pause            SchedulerPauseState
	activeClaimCount []int
	summaries        []storepkg.EventSummary
	cancelAfterPause context.CancelFunc
	requireDeadline  bool
}

func newSchedulerControlTestStore() *schedulerControlTestStore {
	return &schedulerControlTestStore{inMemoryManagerStore: newInMemoryManagerStore()}
}

func (s *schedulerControlTestStore) PauseTask(ctx context.Context, mutation PauseMutation) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	record, err := s.GetTask(ctx, mutation.TaskID)
	if err != nil {
		return Task{}, err
	}
	record.Paused = true
	record.PausedBy = mutation.Actor
	record.PausedAt = mutation.PausedAt
	record.PausedReason = mutation.Reason
	record.UpdatedAt = mutation.PausedAt
	s.tasks[record.ID] = record
	return record, nil
}

func (s *schedulerControlTestStore) ResumeTask(ctx context.Context, mutation ResumeMutation) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	record, err := s.GetTask(ctx, mutation.TaskID)
	if err != nil {
		return Task{}, err
	}
	record.Paused = false
	record.PausedBy = ""
	record.PausedAt = time.Time{}
	record.PausedReason = ""
	record.UpdatedAt = mutation.ResumedAt
	s.tasks[record.ID] = record
	return record, nil
}

func (s *schedulerControlTestStore) GetSchedulerPause(ctx context.Context) (SchedulerPauseState, error) {
	if err := ctx.Err(); err != nil {
		return SchedulerPauseState{}, err
	}
	if err := s.requireStatusDeadline(ctx); err != nil {
		return SchedulerPauseState{}, err
	}
	return s.pause, nil
}

func (s *schedulerControlTestStore) SetSchedulerPaused(
	ctx context.Context,
	actor string,
	reason string,
) (SchedulerPauseState, error) {
	if err := ctx.Err(); err != nil {
		return SchedulerPauseState{}, err
	}
	now := time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	s.pause = SchedulerPauseState{Paused: true, PausedBy: actor, Reason: reason, PausedAt: now, UpdatedAt: now}
	if s.cancelAfterPause != nil {
		s.cancelAfterPause()
		s.cancelAfterPause = nil
	}
	return s.pause, nil
}

func (s *schedulerControlTestStore) SetSchedulerResumed(ctx context.Context) (SchedulerPauseState, error) {
	if err := ctx.Err(); err != nil {
		return SchedulerPauseState{}, err
	}
	now := time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC)
	s.pause = SchedulerPauseState{UpdatedAt: now}
	return s.pause, nil
}

func (s *schedulerControlTestStore) CountActiveTaskRunClaims(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := s.requireStatusDeadline(ctx); err != nil {
		return 0, err
	}
	if len(s.activeClaimCount) == 0 {
		return 0, nil
	}
	count := s.activeClaimCount[0]
	s.activeClaimCount = s.activeClaimCount[1:]
	return count, nil
}

func (s *schedulerControlTestStore) CountQueuedTaskRuns(ctx context.Context, _ bool) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := s.requireStatusDeadline(ctx); err != nil {
		return 0, err
	}
	count := 0
	for _, run := range s.runs {
		if run.Status.Normalize() == TaskRunStatusQueued {
			count++
		}
	}
	return count, nil
}

func (s *schedulerControlTestStore) CountPausedTasks(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := s.requireStatusDeadline(ctx); err != nil {
		return 0, err
	}
	count := 0
	for _, record := range s.tasks {
		if record.Paused {
			count++
		}
	}
	return count, nil
}

func (s *schedulerControlTestStore) SchedulerBacklog(
	ctx context.Context,
	_ SchedulerBacklogQuery,
) (SchedulerBacklog, error) {
	if err := ctx.Err(); err != nil {
		return SchedulerBacklog{}, err
	}
	return SchedulerBacklog{}, nil
}

func (s *schedulerControlTestStore) WriteEventSummary(ctx context.Context, summary storepkg.EventSummary) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.summaries = append(s.summaries, summary)
	return nil
}

func (s *schedulerControlTestStore) requireStatusDeadline(ctx context.Context) error {
	if !s.requireDeadline {
		return nil
	}
	if _, ok := ctx.Deadline(); ok {
		return nil
	}
	return errors.New("scheduler status context missing deadline")
}

func TestSchedulerControls(t *testing.T) {
	t.Parallel()

	for _, status := range []Status{TaskStatusCompleted, TaskStatusFailed, TaskStatusCanceled} {
		t.Run("Should reject pausing "+string(status)+" task", func(t *testing.T) {
			t.Parallel()

			store := newSchedulerControlTestStore()
			store.tasks["task-terminal"] = Task{
				ID:     "task-terminal",
				Scope:  ScopeGlobal,
				Title:  "Terminal task",
				Status: status,
			}
			manager := newTaskManagerForTest(t, store)

			_, err := manager.PauseTask(
				context.Background(),
				"task-terminal",
				PauseTaskRequest{Reason: "hold"},
				validActorContext(),
			)
			if !errors.Is(err, ErrInvalidStatusTransition) {
				t.Fatalf("PauseTask(%s) error = %v, want %v", status, err, ErrInvalidStatusTransition)
			}
			item, ok := diagnosticitems.ItemFromError(err)
			if !ok {
				t.Fatalf("PauseTask(%s) error = %v, want structured diagnostic", status, err)
			}
			if item.Code != diagnosticcontract.CodeTaskRunAlreadyTerminal {
				t.Fatalf("diagnostic code = %q, want %q", item.Code, diagnosticcontract.CodeTaskRunAlreadyTerminal)
			}
			if store.tasks["task-terminal"].Paused {
				t.Fatal("PauseTask() mutated terminal task pause state")
			}
		})
	}

	t.Run("Should complete drain audit after request context is canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		store := newSchedulerControlTestStore()
		store.activeClaimCount = []int{1, 0}
		store.cancelAfterPause = cancel
		store.requireDeadline = true
		manager := newTaskManagerForTest(t, store)

		result, err := manager.DrainScheduler(
			ctx,
			SchedulerDrainRequest{Reason: "deploy handoff", Timeout: time.Millisecond},
			validActorContext(),
		)
		if err != nil {
			t.Fatalf("DrainScheduler() error = %v", err)
		}
		if !result.Completed || result.TimedOut || result.RemainingClaims != 0 {
			t.Fatalf("DrainScheduler() result = %#v, want completed without remaining claims", result)
		}
		if !schedulerEventSummariesContain(store.summaries, eventspkg.SchedulerDrainStarted) {
			t.Fatalf("event summaries = %#v, want %s", store.summaries, eventspkg.SchedulerDrainStarted)
		}
		if !schedulerEventSummariesContain(store.summaries, eventspkg.SchedulerDrainCompleted) {
			t.Fatalf("event summaries = %#v, want %s", store.summaries, eventspkg.SchedulerDrainCompleted)
		}
	})
}

func schedulerEventSummariesContain(summaries []storepkg.EventSummary, eventType string) bool {
	for _, summary := range summaries {
		if summary.Type == eventType {
			return true
		}
	}
	return false
}
