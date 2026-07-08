package loop

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/hooks"
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

func TestCoordinatorRunnerWatchEvents(t *testing.T) {
	t.Run("Should park with rendered subscriptions and initialized cursors when no events match", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusRunning, 0, now.Add(-24*time.Hour))
		coordinatorRun := watchCoordinatorRun(loopRun)
		ledger := &watchEventsLedgerForTest{
			cursors: map[string]int64{WatchEventsTaskStream: 12},
		}
		runner := newWatchEventsCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			coordinatorRunnerOutputs{},
			compileCoordinatorControlDefinition(t, watchEventsDefinitionForTest("")),
			ledger,
		)
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
		if got, want := plan.Terminal.Cause, string(TransitionCauseWatchEvents); got != want {
			t.Fatalf("Terminal.Cause = %q, want %q", got, want)
		}
		outputs := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		pending := decodeWatchEventsOutputRefForTest(t, outputs["watch_tasks"].OutputRef)
		if got, want := pending.Kind, "watch_events_pending"; got != want {
			t.Fatalf("pending.Kind = %q, want %q", got, want)
		}
		if got, want := pending.Cursors[WatchEventsTaskStream], int64(12); got != want {
			t.Fatalf("pending cursor = %d, want %d", got, want)
		}
		if got, want := len(pending.Subscriptions), 1; got != want {
			t.Fatalf("subscriptions len = %d, want %d", got, want)
		}
		if got, want := pending.Subscriptions[0].Kind, string(hooks.HookTaskStatusChanged); got != want {
			t.Fatalf("subscription kind = %q, want %q", got, want)
		}
		if len(plan.NodeRuns) != 0 {
			t.Fatalf("NodeRuns = %#v, want none while watching", plan.NodeRuns)
		}
		if got, want := ledger.cursorQueries[0].Limit, LoopMaxFanoutWidth; got != want {
			t.Fatalf("cursor query Limit = %d, want %d", got, want)
		}
	})

	t.Run("Should yield already-watching run with unchanged cursors on spurious wake", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusWatching, 1, now.Add(-24*time.Hour))
		coordinatorRun := watchCoordinatorRun(loopRun)
		pendingRef := watchEventsPendingRefForTest(t, int64(7), "")
		ledger := &watchEventsLedgerForTest{}
		runner := newWatchEventsCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			coordinatorRunnerOutputs{outputs: map[int][]GenerationOutput{1: {
				{Generation: 1, NodeID: "watch_tasks", Status: generationOutputPending, OutputRef: pendingRef},
				{Generation: 1, NodeID: "summarize", Status: generationOutputPending},
			}}},
			compileCoordinatorControlDefinition(t, watchEventsDefinitionForTest("")),
			ledger,
		)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !plan.Yield {
			t.Fatal("Yield = false, want yield on spurious wake")
		}
		if plan.Terminal != nil {
			t.Fatalf("Terminal = %#v, want nil", plan.Terminal)
		}
		outputs := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		pending := decodeWatchEventsOutputRefForTest(t, outputs["watch_tasks"].OutputRef)
		if got, want := pending.Cursors[WatchEventsTaskStream], int64(7); got != want {
			t.Fatalf("pending cursor = %d, want %d", got, want)
		}
	})

	t.Run("Should apply exact CEL and confirm only matched rows with advanced cursors", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusWatching, 1, now.Add(-time.Minute))
		loopRun.Inputs = map[string]any{"parent_task_id": "task-1"}
		coordinatorRun := watchCoordinatorRun(loopRun)
		filter := `event.task_id == inputs.parent_task_id && event.payload.to_status == "blocked"`
		pendingRef := watchEventsPendingRefForTest(t, int64(7), filter)
		ledger := &watchEventsLedgerForTest{
			rows: []WatchEvent{
				watchTaskStatusEventForTest(8, "task-1", "ready"),
				watchTaskStatusEventForTest(9, "task-1", "blocked"),
			},
		}
		runner := newWatchEventsCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			coordinatorRunnerOutputs{outputs: map[int][]GenerationOutput{1: {
				{Generation: 1, NodeID: "watch_tasks", Status: generationOutputPending, OutputRef: pendingRef},
				{Generation: 1, NodeID: "summarize", Status: generationOutputPending},
			}}},
			compileCoordinatorControlDefinition(t, watchEventsDefinitionForTest(filter)),
			ledger,
		)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if plan.Yield || plan.Terminal != nil {
			t.Fatalf("plan Yield=%v Terminal=%#v, want downstream enqueue", plan.Yield, plan.Terminal)
		}
		if got, want := len(plan.NodeRuns), 1; got != want {
			t.Fatalf("NodeRuns = %d, want %d", got, want)
		}
		outputs := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		confirmed := decodeWatchEventsOutputRefForTest(t, outputs["watch_tasks"].OutputRef)
		if got, want := confirmed.Kind, "watch_events_confirmed"; got != want {
			t.Fatalf("confirmed.Kind = %q, want %q", got, want)
		}
		if got, want := confirmed.Cursors[WatchEventsTaskStream], int64(9); got != want {
			t.Fatalf("confirmed cursor = %d, want %d", got, want)
		}
		var events []WatchEvent
		if err := json.Unmarshal(confirmed.Events, &events); err != nil {
			t.Fatalf("Unmarshal confirmed events error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("events len = %d, want %d", got, want)
		}
		if got, want := events[0].Seq, int64(9); got != want {
			t.Fatalf("event seq = %d, want %d", got, want)
		}
		if got, want := events[0].Payload["to_status"], "blocked"; got != want {
			t.Fatalf("event payload to_status = %v, want %q", got, want)
		}
	})

	t.Run("Should externalize large confirmed batch and enqueue self wake when stream is capped", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		loopRun := watchLoopRun(StatusWatching, 1, now.Add(-time.Minute))
		coordinatorRun := watchCoordinatorRun(loopRun)
		pendingRef := watchEventsPendingRefForTest(t, int64(1), "")
		ledger := &watchEventsLedgerForTest{rows: watchLargeTaskStatusEventsForTest(2, LoopMaxFanoutWidth)}
		runner := newWatchEventsCoordinatorRunnerForTest(
			t,
			loopRun,
			coordinatorRun,
			coordinatorRunnerOutputs{outputs: map[int][]GenerationOutput{1: {
				{Generation: 1, NodeID: "watch_tasks", Status: generationOutputPending, OutputRef: pendingRef},
				{Generation: 1, NodeID: "summarize", Status: generationOutputPending},
			}}},
			compileCoordinatorControlDefinition(t, watchEventsDefinitionForTest("")),
			ledger,
		)
		runner.now = func() time.Time { return now }

		plan, err := runner.Run(context.Background(), task.RunID(coordinatorRun.ID))
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		outputs := outputsByNodeForTest(coordinatorSnapshotPayloadForTest(t, plan).Outputs)
		if !OutputRefLooksContentAddressed(outputs["watch_tasks"].OutputRef) {
			t.Fatalf("watch output_ref = %q, want content-addressed ref", outputs["watch_tasks"].OutputRef)
		}
		blobs := coordinatorSnapshotPayloadForTest(t, plan).OutputBlobs
		if got, want := len(blobs), 1; got != want {
			t.Fatalf("OutputBlobs len = %d, want %d", got, want)
		}
		if blobs[0].OutputRef != outputs["watch_tasks"].OutputRef {
			t.Fatalf("blob ref = %q, want %q", blobs[0].OutputRef, outputs["watch_tasks"].OutputRef)
		}
		confirmed := decodeWatchEventsOutputRefForTest(t, string(blobs[0].Payload))
		var events []WatchEvent
		if err := json.Unmarshal(confirmed.Events, &events); err != nil {
			t.Fatalf("Unmarshal blob events error = %v", err)
		}
		if got, want := len(events), LoopMaxFanoutWidth; got != want {
			t.Fatalf("blob events len = %d, want %d", got, want)
		}
		if got, want := len(plan.PostCommitWakes), 1; got != want {
			t.Fatalf("PostCommitWakes len = %d, want %d", got, want)
		}
		if got, want := plan.PostCommitWakes[0].IdempotencyKey,
			watchEventsWakeKey(loopRun.ID, "watch_tasks"); got != want {
			t.Fatalf("wake key = %q, want %q", got, want)
		}
	})
}

func TestWatchEventsEvaluatorHelpers(t *testing.T) {
	t.Run("Should reject invalid pending state", func(t *testing.T) {
		t.Parallel()

		if err := validateWatchEventsPendingState(watchpkg.EventsPendingState{}); err == nil {
			t.Fatal("validateWatchEventsPendingState(empty) error = nil, want error")
		}
		if err := validateWatchEventsPendingState(watchpkg.EventsPendingState{
			Subscriptions: []watchpkg.EventSubscriptionRef{{Kind: string(hooks.HookTaskStatusChanged)}},
		}); err == nil {
			t.Fatal("validateWatchEventsPendingState(no cursors) error = nil, want error")
		}
	})

	t.Run("Should reject invalid watch-events query state", func(t *testing.T) {
		t.Parallel()

		_, err := watchEventsQuery(Run{WorkspaceID: "ws-1"}, watchpkg.EventsPendingState{
			Subscriptions: []watchpkg.EventSubscriptionRef{{Kind: "unknown.event"}},
			Cursors:       map[string]int64{WatchEventsTaskStream: 1},
		})
		if err == nil {
			t.Fatal("watchEventsQuery(unknown kind) error = nil, want error")
		}
	})

	t.Run("Should build blocked terminal for invalid watch-events state", func(t *testing.T) {
		t.Parallel()

		terminal := watchEventsBlockedTerminal("bad_state")
		if terminal == nil {
			t.Fatal("watchEventsBlockedTerminal() = nil, want terminal")
		}
		if got, want := terminal.Status, string(StatusBlocked); got != want {
			t.Fatalf("terminal status = %q, want %q", got, want)
		}
		if got, want := terminal.ReasonCode, "bad_state"; got != want {
			t.Fatalf("terminal reason = %q, want %q", got, want)
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

type watchEventsLedgerForTest struct {
	cursors       map[string]int64
	rows          []WatchEvent
	cursorQueries []WatchEventsQuery
	matchQueries  []WatchEventsQuery
}

func (l *watchEventsLedgerForTest) ReadCursors(
	_ context.Context,
	query WatchEventsQuery,
) (map[string]int64, error) {
	l.cursorQueries = append(l.cursorQueries, cloneWatchEventsQueryForTest(query))
	return cloneInt64Map(l.cursors), nil
}

func (l *watchEventsLedgerForTest) ReadMatches(
	_ context.Context,
	query WatchEventsQuery,
) ([]WatchEvent, error) {
	l.matchQueries = append(l.matchQueries, cloneWatchEventsQueryForTest(query))
	return append([]WatchEvent(nil), l.rows...), nil
}

func cloneWatchEventsQueryForTest(query WatchEventsQuery) WatchEventsQuery {
	return WatchEventsQuery{
		WorkspaceID: query.WorkspaceID,
		Streams:     cloneInt64Map(query.Streams),
		Kinds:       append([]string(nil), query.Kinds...),
		Limit:       query.Limit,
	}
}

func newWatchEventsCoordinatorRunnerForTest(
	t *testing.T,
	loopRun Run,
	coordinatorRun task.Run,
	outputs GenerationOutputReader,
	resolved *ResolvedDefinition,
	ledger WatchEventsLedger,
) *CoordinatorRunner {
	t.Helper()
	runner, err := NewCoordinatorRunner(
		&coordinatorRunnerTaskRunReader{runs: map[string]task.Run{coordinatorRun.ID: coordinatorRun}},
		coordinatorRunnerLoopStore{run: loopRun},
		outputs,
		DefinitionResolverFunc(
			func(context.Context, WorkspaceID, string) (*ResolvedDefinition, error) {
				return resolved, nil
			},
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		WithCoordinatorWatchEventsLedger(ledger),
		WithCoordinatorWatchSilenceWindow(time.Minute),
	)
	if err != nil {
		t.Fatalf("NewCoordinatorRunner() error = %v", err)
	}
	return runner
}

func watchEventsDefinitionForTest(filter string) dsl.Definition {
	definition := dsl.Definition{
		APIVersion: dsl.APIVersion,
		Kind:       dsl.KindLoop,
		Graph: dsl.Graph{
			Nodes: []dsl.Node{
				{
					ID:    "watch_tasks",
					Class: dsl.NodeClassSource,
					Kind:  string(dsl.SourceWatchEvents),
					Events: []dsl.EventSubscription{{
						Kind:   string(hooks.HookTaskStatusChanged),
						Filter: filter,
					}},
				},
				{
					ID:    "summarize",
					Class: dsl.NodeClassAction,
					Kind:  string(dsl.ActionTransform),
					Params: dsl.NodeParams{
						"map": map[string]any{"ok": map[string]any{"value": true}},
					},
				},
			},
			Edges: []dsl.Edge{{From: "watch_tasks", To: "summarize"}},
		},
	}
	if strings.Contains(filter, "inputs.parent_task_id") {
		definition.Inputs = map[string]dsl.Input{
			"parent_task_id": {Type: dsl.InputTypeString},
		}
	}
	return definition
}

func watchEventsPendingRefForTest(t *testing.T, cursor int64, filter string) string {
	t.Helper()
	ref, err := watchpkg.EventsPendingOutputRef(watchpkg.EventsPendingState{
		Subscriptions: []watchpkg.EventSubscriptionRef{{
			Kind:   string(hooks.HookTaskStatusChanged),
			Filter: strings.TrimSpace(filter),
		}},
		Cursors: map[string]int64{WatchEventsTaskStream: cursor},
	})
	if err != nil {
		t.Fatalf("EventsPendingOutputRef() error = %v", err)
	}
	return ref
}

func decodeWatchEventsOutputRefForTest(t *testing.T, ref string) watchpkg.OutputRef {
	t.Helper()
	var output watchpkg.OutputRef
	if err := json.Unmarshal([]byte(ref), &output); err != nil {
		t.Fatalf("Unmarshal watch-events output ref error = %v", err)
	}
	return output
}

func watchTaskStatusEventForTest(seq int64, taskID string, toStatus string) WatchEvent {
	return WatchEvent{
		Kind:        string(hooks.HookTaskStatusChanged),
		LedgerKind:  string(hooks.HookTaskStatusChanged),
		Seq:         seq,
		Stream:      WatchEventsTaskStream,
		At:          time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		WorkspaceID: "ws-1",
		TaskID:      taskID,
		Payload: map[string]any{
			"from_status": "ready",
			"to_status":   toStatus,
		},
	}
}

func watchLargeTaskStatusEventsForTest(firstSeq int64, count int) []WatchEvent {
	events := make([]WatchEvent, 0, count)
	for idx := range count {
		event := watchTaskStatusEventForTest(firstSeq+int64(idx), "task-large", "blocked")
		event.Payload["details"] = strings.Repeat("x", 512)
		events = append(events, event)
	}
	return events
}
