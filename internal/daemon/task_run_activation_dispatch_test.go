package daemon

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"

	hookspkg "github.com/compozy/agh/internal/hooks"
	taskpkg "github.com/compozy/agh/internal/task"
)

func TestTaskRunActivationDispatcherShouldRouteWorkerRunsByKind(t *testing.T) {
	t.Parallel()

	t.Run("Should route loop-correlated workers to loop runtime and plain workers to task roles", func(t *testing.T) {
		t.Parallel()

		store := newActivationDispatchStore(
			activationDispatchRun("run-loop-worker", taskpkg.RunKindWorker, "loop-run-1"),
			activationDispatchRun("run-plain-worker", taskpkg.RunKindWorker, ""),
			activationDispatchRun("run-coordinator", taskpkg.RunKindCoordinator, "loop-run-1"),
		)
		roles := &activationDispatchObserver{}
		loops := &activationDispatchObserver{}
		dispatcher, err := newTaskRunActivationDispatcher(store, roles, loops, nil)
		if err != nil {
			t.Fatalf("newTaskRunActivationDispatcher() error = %v", err)
		}

		for _, runID := range []string{"run-loop-worker", "run-plain-worker", "run-coordinator"} {
			dispatcher.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
				TaskRunContext: hookspkg.TaskRunContext{RunID: runID},
			})
		}

		if got, want := loops.runIDs(), []string{"run-loop-worker"}; !slices.Equal(got, want) {
			t.Fatalf("loop observer run IDs = %#v, want %#v", got, want)
		}
		if got, want := roles.runIDs(), []string{"run-plain-worker"}; !slices.Equal(got, want) {
			t.Fatalf("role observer run IDs = %#v, want %#v", got, want)
		}
	})

	t.Run("Should recover queued runs through the same dispatch table", func(t *testing.T) {
		t.Parallel()

		store := newActivationDispatchStore(
			activationDispatchRun("run-loop-worker", taskpkg.RunKindWorker, "loop-run-1"),
			activationDispatchRun("run-plain-worker", taskpkg.RunKindWorker, ""),
			activationDispatchRun("run-coordinator", taskpkg.RunKindCoordinator, "loop-run-1"),
		)
		roles := &activationDispatchObserver{}
		loops := &activationDispatchObserver{}
		dispatcher, err := newTaskRunActivationDispatcher(store, roles, loops, nil)
		if err != nil {
			t.Fatalf("newTaskRunActivationDispatcher() error = %v", err)
		}

		dispatcher.Recover(context.Background())

		if got, want := loops.runIDs(), []string{"run-loop-worker"}; !slices.Equal(got, want) {
			t.Fatalf("loop observer run IDs = %#v, want %#v", got, want)
		}
		if got, want := roles.runIDs(), []string{"run-plain-worker"}; !slices.Equal(got, want) {
			t.Fatalf("role observer run IDs = %#v, want %#v", got, want)
		}
	})

	t.Run("Should detach post-commit dispatch with a bounded context", func(t *testing.T) {
		t.Parallel()

		store := newActivationDispatchStore(
			activationDispatchRun("run-plain-worker", taskpkg.RunKindWorker, ""),
		)
		roles := &activationDispatchObserver{}
		dispatcher, err := newTaskRunActivationDispatcher(store, roles, nil, nil)
		if err != nil {
			t.Fatalf("newTaskRunActivationDispatcher() error = %v", err)
		}
		parent, cancel := context.WithCancel(context.Background())
		cancel()

		dispatcher.OnTaskRunEnqueued(parent, hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{RunID: "run-plain-worker"},
		})

		hasDeadline, contextErr := store.getContextState()
		if !hasDeadline {
			t.Fatal("GetTaskRun() context has no deadline, want bounded post-commit work")
		}
		if contextErr != nil {
			t.Fatalf("GetTaskRun() context error = %v, want detached live context", contextErr)
		}
		if got, want := roles.runIDs(), []string{"run-plain-worker"}; !slices.Equal(got, want) {
			t.Fatalf("role observer run IDs = %#v, want %#v", got, want)
		}
	})

	t.Run("Should report a missing activation target for an eligible worker", func(t *testing.T) {
		t.Parallel()

		store := newActivationDispatchStore(
			activationDispatchRun("run-loop-worker", taskpkg.RunKindWorker, "loop-run-1"),
		)
		roles := &activationDispatchObserver{}
		var logs bytes.Buffer
		dispatcher, err := newTaskRunActivationDispatcher(
			store,
			roles,
			nil,
			slog.New(slog.NewTextHandler(&logs, nil)),
		)
		if err != nil {
			t.Fatalf("newTaskRunActivationDispatcher() error = %v", err)
		}

		dispatcher.OnTaskRunEnqueued(context.Background(), hookspkg.TaskRunEnqueuedPayload{
			TaskRunContext: hookspkg.TaskRunContext{RunID: "run-loop-worker"},
		})

		output := logs.String()
		for _, want := range []string{
			"daemon: task run activation target is unavailable",
			"run_id=run-loop-worker",
			"run_kind=worker",
			"loop_worker=true",
		} {
			if !strings.Contains(output, want) {
				t.Fatalf("activation log = %q, want substring %q", output, want)
			}
		}
	})
}

type activationDispatchStore struct {
	mu             sync.Mutex
	runs           map[string]taskpkg.Run
	getHasDeadline bool
	getContextErr  error
}

func newActivationDispatchStore(runs ...taskpkg.Run) *activationDispatchStore {
	store := &activationDispatchStore{runs: make(map[string]taskpkg.Run, len(runs))}
	for _, run := range runs {
		store.runs[run.ID] = run
	}
	return store
}

func (s *activationDispatchStore) GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, s.getHasDeadline = ctx.Deadline()
	s.getContextErr = ctx.Err()
	run, ok := s.runs[id]
	if !ok {
		return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
	}
	return run, nil
}

func (s *activationDispatchStore) ListTaskRunsByStatus(
	_ context.Context,
	statuses []taskpkg.RunStatus,
) ([]taskpkg.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	allowed := make(map[taskpkg.RunStatus]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status.Normalize()] = struct{}{}
	}
	runs := make([]taskpkg.Run, 0, len(s.runs))
	for _, run := range s.runs {
		if _, ok := allowed[run.Status.Normalize()]; ok {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func (s *activationDispatchStore) getContextState() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getHasDeadline, s.getContextErr
}

type activationDispatchObserver struct {
	mu       sync.Mutex
	payloads []hookspkg.TaskRunEnqueuedPayload
}

func (o *activationDispatchObserver) OnTaskRunEnqueued(
	_ context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.payloads = append(o.payloads, payload)
}

func (o *activationDispatchObserver) runIDs() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	ids := make([]string, 0, len(o.payloads))
	for _, payload := range o.payloads {
		ids = append(ids, payload.RunID)
	}
	return ids
}

func activationDispatchRun(id string, kind taskpkg.RunKind, loopRunID string) taskpkg.Run {
	return taskpkg.Run{
		ID:        id,
		TaskID:    "task-" + id,
		Status:    taskpkg.TaskRunStatusQueued,
		RunKind:   kind,
		LoopRunID: loopRunID,
	}
}
