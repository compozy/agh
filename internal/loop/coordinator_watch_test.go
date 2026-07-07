package loop

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/compozy/agh/internal/loop/dsl"
	watchpkg "github.com/compozy/agh/internal/loop/watch"
	"github.com/compozy/agh/internal/task"
)

func TestCoordinatorRunnerWatchSource(t *testing.T) {
	t.Run("Should yield to watching when source is not ready", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusRunning, 0, now.Add(-time.Minute))
		coordinatorRun := watchCoordinatorRun(loopRun)
		poller := watchPollerFunc(func(_ context.Context, req watchpkg.PollRequest) (watchpkg.PollResponse, error) {
			if string(req.Spec) != `{"kind":"reviews","query":"open"}` {
				t.Fatalf("PollRequest.Spec = %s, want watch spec", string(req.Spec))
			}
			if req.ExpectedStateDigest != "" {
				t.Fatalf("ExpectedStateDigest = %q, want empty", req.ExpectedStateDigest)
			}
			return watchpkg.PollResponse{Ready: false, StateDigest: "sha256:current"}, nil
		})
		runner := newWatchCoordinatorRunnerForTest(t, loopRun, coordinatorRun, nil, coordinatorRunnerOutputs{}, poller)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if plan.Terminal == nil {
			t.Fatal("Terminal = nil, want watching terminal")
		}
		if got, want := plan.Terminal.Status, string(StatusWatching); got != want {
			t.Fatalf("Terminal.Status = %q, want %q", got, want)
		}
		if got, want := plan.Terminal.Cause, string(TransitionCauseWatchPoll); got != want {
			t.Fatalf("Terminal.Cause = %q, want %q", got, want)
		}
		outputs := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		digest, err := watchpkg.ExpectedStateDigestFromOutputRef(outputs["watch_reviews"].OutputRef)
		if err != nil {
			t.Fatalf("ExpectedStateDigestFromOutputRef() error = %v", err)
		}
		if digest != "sha256:current" {
			t.Fatalf("pending digest = %q, want sha256:current", digest)
		}
		if len(plan.NodeRuns) != 0 {
			t.Fatalf("NodeRuns = %#v, want none while watching", plan.NodeRuns)
		}
	})

	t.Run("Should re-claim watching source and enqueue downstream when ready", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusWatching, 1, now.Add(-time.Minute))
		coordinatorRun := watchCoordinatorRun(loopRun)
		pendingRef, err := watchpkg.PendingOutputRef(watchpkg.PollResponse{StateDigest: "sha256:previous"})
		if err != nil {
			t.Fatalf("PendingOutputRef() error = %v", err)
		}
		poller := watchPollerFunc(func(_ context.Context, req watchpkg.PollRequest) (watchpkg.PollResponse, error) {
			if req.ExpectedStateDigest != "sha256:previous" {
				t.Fatalf("ExpectedStateDigest = %q, want sha256:previous", req.ExpectedStateDigest)
			}
			return watchpkg.PollResponse{
				Ready:       true,
				StateDigest: "sha256:next",
				Payload:     json.RawMessage(`{"review":"r1"}`),
			}, nil
		})
		runner := newWatchCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			nil,
			coordinatorRunnerOutputs{outputs: map[int][]GenerationOutput{1: {
				{Generation: 1, NodeID: "watch_reviews", Status: generationOutputPending, OutputRef: pendingRef},
				{Generation: 1, NodeID: "fix_review", Status: generationOutputPending},
			}}},
			poller,
		)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if plan.Terminal != nil {
			t.Fatalf("Terminal = %#v, want nil", plan.Terminal)
		}
		if plan.Yield {
			t.Fatal("Yield = true, want downstream enqueue")
		}
		if got, want := len(plan.NodeRuns), 1; got != want {
			t.Fatalf("NodeRuns = %d, want %d", got, want)
		}
		if got, want := plan.NodeRuns[0].TaskID, coordinatorNodeTaskID(loopRun.ID, 1, "fix_review", 0); got != want {
			t.Fatalf("NodeRuns[0].TaskID = %q, want %q", got, want)
		}
		snapshot := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		if got, want := snapshot["watch_reviews"].Status, generationOutputSucceeded; got != want {
			t.Fatalf("watch output status = %q, want %q", got, want)
		}
		post := outputsByNodeForTest(coordinatorPostReservePayloadForTest(t, plan).Outputs)
		if got, want := post["fix_review"].Status, generationOutputEnqueued; got != want {
			t.Fatalf("fix output status = %q, want %q", got, want)
		}
	})

	t.Run("Should stall watching source after silence window", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusWatching, 1, now.Add(-3*time.Minute))
		coordinatorRun := watchCoordinatorRun(loopRun)
		poller := watchPollerFunc(func(context.Context, watchpkg.PollRequest) (watchpkg.PollResponse, error) {
			return watchpkg.PollResponse{Ready: false, StateDigest: "sha256:old"}, nil
		})
		runner := newWatchCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			nil,
			coordinatorRunnerOutputs{outputs: map[int][]GenerationOutput{1: {
				{Generation: 1, NodeID: "watch_reviews", Status: generationOutputPending},
				{Generation: 1, NodeID: "fix_review", Status: generationOutputPending},
			}}},
			poller,
		)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if plan.Terminal == nil {
			t.Fatal("Terminal = nil, want stalled")
		}
		if got, want := plan.Terminal.Status, string(StatusStalled); got != want {
			t.Fatalf("Terminal.Status = %q, want %q", got, want)
		}
		if got, want := plan.Terminal.ReasonCode, watchSourceSilenceReason; got != want {
			t.Fatalf("Terminal.ReasonCode = %q, want %q", got, want)
		}
	})
}

type watchPollerFunc func(context.Context, watchpkg.PollRequest) (watchpkg.PollResponse, error)

func (f watchPollerFunc) Poll(ctx context.Context, req watchpkg.PollRequest) (watchpkg.PollResponse, error) {
	return f(ctx, req)
}

func watchLoopRun(status Status, generation int, lastProgressAt time.Time) Run {
	return Run{
		ID:             "looprun-watch-source",
		WorkspaceID:    "ws-1",
		LoopName:       "watch-loop",
		Status:         status,
		Generation:     generation,
		LastProgressAt: lastProgressAt,
	}
}

func watchCoordinatorRun(run Run) task.Run {
	return task.Run{
		ID:        "run-coordinator-watch",
		TaskID:    "task-coordinator-watch",
		RunKind:   task.RunKindCoordinator,
		LoopRunID: string(run.ID),
		Status:    task.TaskRunStatusClaimed,
	}
}

func newWatchCoordinatorRunnerForTest(
	t *testing.T,
	loopRun Run,
	coordinatorRun task.Run,
	runs map[string]task.Run,
	outputs GenerationOutputReader,
	poller WatchPoller,
) *CoordinatorRunner {
	t.Helper()
	if runs == nil {
		runs = map[string]task.Run{coordinatorRun.ID: coordinatorRun}
	}
	runner, err := NewCoordinatorRunner(
		&coordinatorRunnerTaskRunReader{runs: runs},
		coordinatorRunnerLoopStore{run: loopRun},
		outputs,
		DefinitionResolverFunc(
			func(context.Context, WorkspaceID, string) (*ResolvedDefinition, error) {
				return &ResolvedDefinition{
					Definition: dsl.Definition{Graph: watchSourceGraphForTest()},
				}, nil
			},
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithCoordinatorWatchPoller(poller),
		WithCoordinatorWatchSilenceWindow(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("NewCoordinatorRunner() error = %v", err)
	}
	return runner
}

func watchSourceGraphForTest() dsl.Graph {
	return dsl.Graph{
		Nodes: []dsl.Node{
			{
				ID:        "watch_reviews",
				Class:     dsl.NodeClassSource,
				Kind:      string(dsl.SourceWatchSource),
				WatchSpec: map[string]any{"kind": "reviews", "query": "open"},
			},
			{
				ID:    "fix_review",
				Class: dsl.NodeClassAction,
				Kind:  string(dsl.ActionTransform),
			},
		},
		Edges: []dsl.Edge{{From: "watch_reviews", To: "fix_review"}},
	}
}
