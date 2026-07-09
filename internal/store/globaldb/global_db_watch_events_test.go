package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/automation"
	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

const watchEventsGenerationOutputEnqueuedForTest = "enqueued"

func TestGlobalDBWatchEventsReadMatches(t *testing.T) {
	t.Parallel()

	t.Run("Should read task events strictly after cursor with join-scoped workspace projection", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		parent := workspaceTaskRecordForTest("watch-parent", "ws-a")
		if err := globalDB.CreateTask(ctx, parent); err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}
		child := workspaceTaskRecordForTest("watch-child", "ws-a")
		child.ParentTaskID = parent.ID
		if err := globalDB.CreateTask(ctx, child); err != nil {
			t.Fatalf("CreateTask(child) error = %v", err)
		}
		foreign := workspaceTaskRecordForTest("watch-foreign", "ws-b")
		if err := globalDB.CreateTask(ctx, foreign); err != nil {
			t.Fatalf("CreateTask(foreign) error = %v", err)
		}
		base := time.Date(2026, 7, 8, 14, 0, 0, 0, time.UTC)
		appendTaskWatchEventForTest(ctx, t, globalDB, child.ID, base, "ready")
		appendTaskWatchEventForTest(ctx, t, globalDB, child.ID, base.Add(time.Minute), "blocked")
		appendTaskWatchEventForTest(ctx, t, globalDB, foreign.ID, base.Add(2*time.Minute), "blocked")

		cursors, err := globalDB.ReadCursors(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsTaskStream: 0},
			Kinds:       []string{string(hookspkg.HookTaskStatusChanged)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadCursors() error = %v", err)
		}
		if got, want := cursors[looppkg.WatchEventsTaskStream], int64(2); got != want {
			t.Fatalf("task cursor = %d, want %d", got, want)
		}

		events, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsTaskStream: 1},
			Kinds:       []string{string(hookspkg.HookTaskStatusChanged)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches() error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("events len = %d, want %d: %#v", got, want, events)
		}
		event := events[0]
		if event.Seq != 2 || event.TaskID != child.ID || event.WorkspaceID != "ws-a" {
			t.Fatalf("task event projection = %#v", event)
		}
		if got, want := event.Payload["parent_task_id"], parent.ID; got != want {
			t.Fatalf("parent_task_id = %v, want %q", got, want)
		}
		assertWatchEventRFC3339UTC(t, event.At)

		appendTaskWatchEventForTest(ctx, t, globalDB, parent.ID, base.Add(3*time.Minute), "completed")
		rootEvents, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsTaskStream: 2},
			Kinds:       []string{string(hookspkg.HookTaskStatusChanged)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(root) error = %v", err)
		}
		if got, want := len(rootEvents), 1; got != want {
			t.Fatalf("root events len = %d, want %d: %#v", got, want, rootEvents)
		}
		if got, want := rootEvents[0].Payload["parent_task_id"], ""; got != want {
			t.Fatalf("root parent_task_id = %v, want empty string", got)
		}
	})

	t.Run("Should apply limit per stream without starving loop events", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		taskRecord := workspaceTaskRecordForTest("watch-hot-task", "ws-a")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		base := time.Date(2026, 7, 8, 15, 0, 0, 0, time.UTC)
		appendTaskWatchEventForTest(ctx, t, globalDB, taskRecord.ID, base, "ready")
		appendTaskWatchEventForTest(ctx, t, globalDB, taskRecord.ID, base.Add(time.Minute), "blocked")
		loopRun := testLoopRun("watch-loop-run", base, looppkg.StatusRunning)
		loopRun.WorkspaceID = "ws-a"
		if _, err := globalDB.CreateLoopRunForStart(ctx, loopRun, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart() error = %v", err)
		}
		if err := globalDB.CompareAndSwapLoopRunStatus(
			ctx,
			loopRun.ID,
			looppkg.StatusRunning,
			looppkg.StatusDone,
			looppkg.TransitionCauseContract,
			base.Add(2*time.Minute),
		); err != nil {
			t.Fatalf("CompareAndSwapLoopRunStatus() error = %v", err)
		}
		cursors, err := globalDB.ReadCursors(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams: map[string]int64{
				looppkg.WatchEventsTaskStream: 0,
				looppkg.WatchEventsLoopStream: 0,
			},
			Kinds: []string{
				string(hookspkg.HookTaskStatusChanged),
				"status_changed",
			},
			Limit: 1,
		})
		if err != nil {
			t.Fatalf("ReadCursors(loop stream) error = %v", err)
		}
		if cursors[looppkg.WatchEventsLoopStream] == 0 {
			t.Fatalf("loop cursor = %d, want non-zero", cursors[looppkg.WatchEventsLoopStream])
		}

		events, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams: map[string]int64{
				looppkg.WatchEventsTaskStream: 0,
				looppkg.WatchEventsLoopStream: 0,
			},
			Kinds: []string{
				string(hookspkg.HookTaskStatusChanged),
				"status_changed",
			},
			Limit: 1,
		})
		if err != nil {
			t.Fatalf("ReadMatches() error = %v", err)
		}
		counts := watchEventCountsByStream(events)
		if got, want := counts[looppkg.WatchEventsTaskStream], 1; got != want {
			t.Fatalf("task stream count = %d, want %d", got, want)
		}
		if got, want := counts[looppkg.WatchEventsLoopStream], 1; got != want {
			t.Fatalf("loop stream count = %d, want %d; events=%#v", got, want, events)
		}
		for _, event := range events {
			assertWatchEventRFC3339UTC(t, event.At)
		}
	})

	t.Run("Should read automation terminal rows with join-scoped workspace projection", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		jobA, err := globalDB.CreateJob(
			ctx,
			automationJobForTest(
				automation.AutomationScopeWorkspace,
				"watch-auto-a",
				"ws-a",
				automation.JobSourceDynamic,
			),
		)
		if err != nil {
			t.Fatalf("CreateJob(ws-a) error = %v", err)
		}
		jobB, err := globalDB.CreateJob(
			ctx,
			automationJobForTest(
				automation.AutomationScopeWorkspace,
				"watch-auto-b",
				"ws-b",
				automation.JobSourceDynamic,
			),
		)
		if err != nil {
			t.Fatalf("CreateJob(ws-b) error = %v", err)
		}
		base := time.Date(2026, 7, 8, 20, 0, 0, 0, time.UTC)
		runningSeed := automationRunForJob(jobA.ID, automation.RunRunning, 1, base)
		runningSeed.EndedAt = nil
		running, err := globalDB.CreateRun(ctx, runningSeed)
		if err != nil {
			t.Fatalf("CreateRun(running) error = %v", err)
		}
		if _, err := globalDB.CreateRun(
			ctx,
			automationRunForJob(jobA.ID, automation.RunCompleted, 1, base.Add(time.Minute)),
		); err != nil {
			t.Fatalf("CreateRun(seed completed) error = %v", err)
		}
		cursorBefore, err := globalDB.ReadCursors(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsAutomationStream: 0},
			Kinds:       []string{string(hookspkg.HookAutomationRunCompleted)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadCursors(before terminal) error = %v", err)
		}
		running.Status = automation.RunCompleted
		running.SessionID = "sess-auto-watch"
		running.EndedAt = timePointer(base.Add(3 * time.Minute))
		updated, err := globalDB.UpdateRun(ctx, running)
		if err != nil {
			t.Fatalf("UpdateRun(terminal) error = %v", err)
		}
		if _, err := globalDB.CreateRun(
			ctx,
			automationRunForJob(jobB.ID, automation.RunCompleted, 1, base.Add(4*time.Minute)),
		); err != nil {
			t.Fatalf("CreateRun(foreign completed) error = %v", err)
		}

		events, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams: map[string]int64{
				looppkg.WatchEventsAutomationStream: cursorBefore[looppkg.WatchEventsAutomationStream],
			},
			Kinds: []string{string(hookspkg.HookAutomationRunCompleted)},
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(automation) error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("automation events len = %d, want %d: %#v", got, want, events)
		}
		event := events[0]
		if event.RunID != updated.ID || event.WorkspaceID != "ws-a" ||
			event.Seq <= cursorBefore[looppkg.WatchEventsAutomationStream] {
			t.Fatalf(
				"automation event projection = %#v, cursorBefore=%d",
				event,
				cursorBefore[looppkg.WatchEventsAutomationStream],
			)
		}
		if got, want := event.Payload[watchEventsPayloadJobIDKey], jobA.ID; got != want {
			t.Fatalf("automation job_id = %v, want %q", got, want)
		}
		if got, want := event.Payload[watchEventsPayloadDurationMSKey], (3 * time.Minute).Milliseconds(); got != want {
			t.Fatalf("automation duration_ms = %v, want %d", got, want)
		}
		assertWatchEventRFC3339UTC(t, event.At)

		failedJob := automationJobForTest(
			automation.AutomationScopeWorkspace,
			"watch-auto-failed",
			"ws-a",
			automation.JobSourceDynamic,
		)
		failedJob.Retry = automation.DefaultBackoffRetryConfig()
		createdFailedJob, err := globalDB.CreateJob(ctx, failedJob)
		if err != nil {
			t.Fatalf("CreateJob(failed automation) error = %v", err)
		}
		failedRun := automationRunForJob(createdFailedJob.ID, automation.RunFailed, 1, base.Add(5*time.Minute))
		failedRun.SessionID = "sess-auto-failed"
		failedRun.Error = "agent crashed"
		failed, err := globalDB.CreateRun(ctx, failedRun)
		if err != nil {
			t.Fatalf("CreateRun(failed automation) error = %v", err)
		}

		failedEvents, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsAutomationStream: 0},
			Kinds:       []string{string(hookspkg.HookAutomationRunFailed)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(automation failed) error = %v", err)
		}
		if got, want := len(failedEvents), 1; got != want {
			t.Fatalf("automation failed events len = %d, want %d: %#v", got, want, failedEvents)
		}
		failedEvent := failedEvents[0]
		if failedEvent.Kind != string(hookspkg.HookAutomationRunFailed) || failedEvent.RunID != failed.ID {
			t.Fatalf("automation failed event = %#v", failedEvent)
		}
		if got, want := failedEvent.Payload[watchEventsPayloadWillRetryKey], true; got != want {
			t.Fatalf("automation will_retry = %v, want %t", got, want)
		}
		if got, want := failedEvent.Payload[watchEventsPayloadErrorKey], "agent crashed"; got != want {
			t.Fatalf("automation error = %v, want %q", got, want)
		}
	})

	t.Run("Should read network work transitions with workspace-column scoping", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		base := time.Date(2026, 7, 8, 20, 30, 0, 0, time.UTC)
		opening := networkWatchThreadMessageForTest(
			"ws-a",
			"msg-work-open",
			"thread_watch",
			"coder.sess-a",
			"please review",
			base,
		)
		opening.PeerTo = "reviewer.sess-a"
		opening.WorkID = "work-watch"
		if _, err := globalDB.WriteConversationMessage(ctx, opening); err != nil {
			t.Fatalf("WriteConversationMessage(opening) error = %v", err)
		}
		transition := networkWatchTraceMessageForTest(
			"ws-a",
			"msg-work-transition",
			"thread_watch",
			"reviewer.sess-a",
			"work-watch",
			store.NetworkWorkStateWorking,
			base.Add(time.Minute),
		)
		if _, err := globalDB.WriteConversationMessage(ctx, transition); err != nil {
			t.Fatalf("WriteConversationMessage(transition) error = %v", err)
		}
		foreign := networkWatchThreadMessageForTest(
			"ws-b",
			"msg-work-open-foreign",
			"thread_watch_foreign",
			"coder.sess-b",
			"please review",
			base,
		)
		foreign.PeerTo = "reviewer.sess-b"
		foreign.WorkID = "work-foreign"
		if _, err := globalDB.WriteConversationMessage(ctx, foreign); err != nil {
			t.Fatalf("WriteConversationMessage(foreign opening) error = %v", err)
		}

		threadEvents, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsNetworkStream: 0},
			Kinds:       []string{string(hookspkg.HookNetworkThreadOpened)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(thread opened) error = %v", err)
		}
		if got, want := len(threadEvents), 1; got != want {
			t.Fatalf("thread opened events len = %d, want %d: %#v", got, want, threadEvents)
		}
		if got, want := threadEvents[0].Payload[watchEventsPayloadThreadIDKey], "thread_watch"; got != want {
			t.Fatalf("thread_id = %v, want %q", got, want)
		}

		directID, err := networkWatchDirectIDForTest("ws-a", "builders", "coder.sess-a", "reviewer.sess-a")
		if err != nil {
			t.Fatalf("networkWatchDirectIDForTest() error = %v", err)
		}
		directMessage := networkWatchDirectMessageForTest(
			"ws-a",
			"msg-direct-open",
			directID,
			"coder.sess-a",
			"reviewer.sess-a",
			"open direct",
			base.Add(2*time.Minute),
		)
		if _, err := globalDB.WriteConversationMessage(ctx, directMessage); err != nil {
			t.Fatalf("WriteConversationMessage(direct) error = %v", err)
		}
		directEvents, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsNetworkStream: 0},
			Kinds:       []string{string(hookspkg.HookNetworkDirectRoomOpened)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(direct opened) error = %v", err)
		}
		if got, want := len(directEvents), 1; got != want {
			t.Fatalf("direct opened events len = %d, want %d: %#v", got, want, directEvents)
		}
		if got, want := directEvents[0].Payload[watchEventsPayloadDirectIDKey], directID; got != want {
			t.Fatalf("direct_id = %v, want %q", got, want)
		}

		events, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsNetworkStream: 0},
			Kinds:       []string{string(hookspkg.HookNetworkWorkTransitioned)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadMatches(network) error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("network events len = %d, want %d: %#v", got, want, events)
		}
		event := events[0]
		if event.WorkspaceID != "ws-a" || event.Channel != "builders" || event.WorkID != "work-watch" {
			t.Fatalf("network event projection = %#v", event)
		}
		if got, want := event.Payload[watchEventsPayloadWorkStateKey], store.NetworkWorkStateWorking; got != want {
			t.Fatalf("network work_state = %v, want %q", got, want)
		}
		if got, want := event.Payload[taskRunResultKindKey], store.NetworkKindTrace; got != want {
			t.Fatalf("network payload kind = %v, want %q", got, want)
		}
		cursors, err := globalDB.ReadCursors(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{looppkg.WatchEventsNetworkStream: 0},
			Kinds:       []string{string(hookspkg.HookNetworkWorkTransitioned)},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ReadCursors(network) error = %v", err)
		}
		if got, want := cursors[looppkg.WatchEventsNetworkStream], event.Seq; got != want {
			t.Fatalf("network cursor = %d, want event seq %d", got, want)
		}
	})
}

func TestGlobalDBWatchEventsCoordinatorIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should park then wake and enqueue downstream with confirmed batch", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 16, 0, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		targetTask := workspaceTaskRecordForTest("watch-integration-target", "ws-a")
		if err := globalDB.CreateTask(ctx, targetTask); err != nil {
			t.Fatalf("CreateTask(target) error = %v", err)
		}
		loopRun := testLoopRun("watch-events-integration", now, looppkg.StatusRunning)
		loopRun.WorkspaceID = "ws-a"
		loopRun.Inputs = map[string]any{"target_task_id": targetTask.ID}
		created, err := globalDB.CreateLoopRunForStart(ctx, loopRun, dsl.ConcurrencyAllow)
		if err != nil {
			t.Fatalf("CreateLoopRunForStart() error = %v", err)
		}
		resolved := compileWatchEventsIntegrationDefinitionForTest(t)
		runner := newGlobalDBWatchEventsCoordinatorForTest(t, globalDB, resolved)
		actor := coordinatorActorContextForTest()
		firstClaim := claimCoordinatorRunForTest(
			ctx,
			t,
			globalDB,
			created.ID,
			"run-watch-events-integration-first",
			now,
		)

		firstPlan, err := runner.Run(ctx, taskpkg.RunID(firstClaim.Run.ID))
		if err != nil {
			t.Fatalf("Run(first) error = %v", err)
		}
		if firstPlan.Terminal == nil || firstPlan.Terminal.Status != string(looppkg.StatusWatching) {
			t.Fatalf("first terminal = %#v, want watching", firstPlan.Terminal)
		}
		firstResult, err := globalDB.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
			RunID:      firstClaim.Run.ID,
			ClaimToken: firstClaim.ClaimToken,
			Actor:      actor,
			Plan:       firstPlan,
			Now:        now.Add(time.Second),
		}, looppkg.NewStoreFinalizer())
		if err != nil {
			t.Fatalf("CompleteCoordinatorAndEnqueueNext(first) error = %v", err)
		}
		if got, want := coordinatorResultStatus(t, &firstResult), string(looppkg.StatusWatching); got != want {
			t.Fatalf("first loop status = %q, want %q", got, want)
		}

		appendTaskWatchEventForTest(ctx, t, globalDB, targetTask.ID, now.Add(2*time.Second), "blocked")
		wakeRun, added, err := globalDB.EnqueueLoopCoordinatorWake(
			ctx,
			string(created.ID),
			"watch-events-integration-wake",
			actor.Origin,
			now.Add(3*time.Second),
		)
		if err != nil {
			t.Fatalf("EnqueueLoopCoordinatorWake() error = %v", err)
		}
		if !added {
			t.Fatal("EnqueueLoopCoordinatorWake() added = false, want true")
		}
		secondClaim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			RunID:            wakeRun.ID,
			Scope:            taskpkg.ScopeWorkspace,
			WorkspaceID:      "ws-a",
			RunKind:          taskpkg.RunKindCoordinator,
			ClaimerSessionID: "daemon-loop-watch-events-integration",
			ClaimedBy:        &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "loop"},
			LeaseDuration:    time.Minute,
			Now:              now.Add(4 * time.Second),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(wake) error = %v", err)
		}

		secondPlan, err := runner.Run(ctx, taskpkg.RunID(secondClaim.Run.ID))
		if err != nil {
			t.Fatalf("Run(second) error = %v", err)
		}
		if secondPlan.Terminal != nil || secondPlan.Yield {
			t.Fatalf(
				"second plan Terminal=%#v Yield=%v, want downstream enqueue",
				secondPlan.Terminal,
				secondPlan.Yield,
			)
		}
		if got, want := len(secondPlan.NodeRuns), 1; got != want {
			t.Fatalf("second NodeRuns = %d, want %d", got, want)
		}
		secondResult, err := globalDB.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
			RunID:      secondClaim.Run.ID,
			ClaimToken: secondClaim.ClaimToken,
			Actor:      actor,
			Plan:       secondPlan,
			Now:        now.Add(5 * time.Second),
		}, looppkg.NewStoreFinalizer())
		if err != nil {
			t.Fatalf("CompleteCoordinatorAndEnqueueNext(second) error = %v", err)
		}
		if got, want := len(secondResult.EnqueuedRuns), 1; got != want {
			t.Fatalf("second EnqueuedRuns = %d, want %d", got, want)
		}
		outputs, err := globalDB.ListGenerationOutputs(ctx, created.ID, 1)
		if err != nil {
			t.Fatalf("ListGenerationOutputs() error = %v", err)
		}
		byNode := watchEventsGenerationOutputsByNode(outputs)
		watchOutput := byNode["watch_tasks"]
		if watchOutput.Status != "succeeded" {
			t.Fatalf("watch output status = %q, want succeeded", watchOutput.Status)
		}
		confirmed := decodeWatchEventsConfirmedRefForTest(t, watchOutput.OutputRef)
		if got, want := confirmed.Cursors[looppkg.WatchEventsTaskStream], int64(1); got != want {
			t.Fatalf("confirmed cursor = %d, want %d", got, want)
		}
		var events []looppkg.WatchEvent
		if err := json.Unmarshal(confirmed.Events, &events); err != nil {
			t.Fatalf("Unmarshal confirmed events error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("confirmed events len = %d, want %d", got, want)
		}
		if events[0].TaskID != targetTask.ID || events[0].Payload["to_status"] != "blocked" {
			t.Fatalf("confirmed event = %#v, want target blocked event", events[0])
		}
		downstream := byNode["summarize"]
		if downstream.Status != watchEventsGenerationOutputEnqueuedForTest || downstream.TaskRunID == "" {
			t.Fatalf("downstream output = %#v, want enqueued with task run", downstream)
		}
		storedRun, err := globalDB.GetTaskRun(ctx, downstream.TaskRunID)
		if err != nil {
			t.Fatalf("GetTaskRun(downstream) error = %v", err)
		}
		if storedRun.Status != taskpkg.TaskRunStatusQueued || storedRun.LoopRunID != string(created.ID) {
			t.Fatalf("downstream run = %#v, want queued loop worker", storedRun)
		}
	})

	t.Run("Should wake automation subscriptions from terminal run rows", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 21, 0, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		job, err := globalDB.CreateJob(
			ctx,
			automationJobForTest(
				automation.AutomationScopeWorkspace,
				"watch-auto-integration",
				"ws-a",
				automation.JobSourceDynamic,
			),
		)
		if err != nil {
			t.Fatalf("CreateJob(automation) error = %v", err)
		}
		created := parkWatchEventsLoopWithDefinitionForTest(
			ctx,
			t,
			globalDB,
			now,
			"watch-events-automation-integration",
			map[string]any{watchEventsPayloadJobIDKey: job.ID},
			compileAutomationWatchEventsIntegrationDefinitionForTest(t),
		)
		runningSeed := automationRunForJob(job.ID, automation.RunRunning, 1, now.Add(2*time.Second))
		runningSeed.EndedAt = nil
		running, err := globalDB.CreateRun(ctx, runningSeed)
		if err != nil {
			t.Fatalf("CreateRun(running automation) error = %v", err)
		}
		running.Status = automation.RunCompleted
		running.SessionID = "sess-auto-integration"
		running.EndedAt = timePointer(now.Add(42 * time.Second))
		updated, err := globalDB.UpdateRun(ctx, running)
		if err != nil {
			t.Fatalf("UpdateRun(completed automation) error = %v", err)
		}
		actor := coordinatorActorContextForTest()
		wakeRun, added, err := globalDB.EnqueueLoopCoordinatorWake(
			ctx,
			string(created.ID),
			"watch-events-automation-integration-wake",
			actor.Origin,
			now.Add(43*time.Second),
		)
		if err != nil {
			t.Fatalf("EnqueueLoopCoordinatorWake(automation) error = %v", err)
		}
		if !added {
			t.Fatal("EnqueueLoopCoordinatorWake(automation) added = false, want true")
		}

		events, byNode := claimAndRunWatchEventsWakeForTest(
			ctx,
			t,
			globalDB,
			created,
			compileAutomationWatchEventsIntegrationDefinitionForTest(t),
			wakeRun,
			now.Add(44*time.Second),
			"watch_automation",
		)
		if got, want := len(events), 1; got != want {
			t.Fatalf("automation confirmed events len = %d, want %d: %#v", got, want, events)
		}
		event := events[0]
		if event.Kind != string(hookspkg.HookAutomationRunCompleted) ||
			event.RunID != updated.ID ||
			event.WorkspaceID != "ws-a" ||
			event.Stream != looppkg.WatchEventsAutomationStream {
			t.Fatalf("automation confirmed event = %#v", event)
		}
		if got, want := event.Payload[watchEventsPayloadJobIDKey], job.ID; got != want {
			t.Fatalf("automation job_id = %v, want %q", got, want)
		}
		if got, want := event.Payload[watchEventsPayloadSessionIDKey], "sess-auto-integration"; got != want {
			t.Fatalf("automation session_id = %v, want %q", got, want)
		}
		downstream := byNode["summarize"]
		if downstream.Status != watchEventsGenerationOutputEnqueuedForTest || downstream.TaskRunID == "" {
			t.Fatalf("automation downstream output = %#v, want enqueued", downstream)
		}
	})
}

func TestGlobalDBWatchEventsParkedIndexAndRecovery(t *testing.T) {
	t.Parallel()

	t.Run("Should list parked subscriptions from durable pending output refs", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 18, 30, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		created, targetTask := parkWatchEventsLoopForRecoveryTest(ctx, t, globalDB, now)

		parked, err := globalDB.ListParkedWatchEventSubscriptions(ctx)
		if err != nil {
			t.Fatalf("ListParkedWatchEventSubscriptions() error = %v", err)
		}
		if got, want := len(parked), 1; got != want {
			t.Fatalf("parked len = %d, want %d", got, want)
		}
		subscription := parked[0]
		if got, want := subscription.LoopRunID, string(created.ID); got != want {
			t.Fatalf("LoopRunID = %q, want %q", got, want)
		}
		if got, want := subscription.NodeID, "watch_tasks"; got != want {
			t.Fatalf("NodeID = %q, want %q", got, want)
		}
		if got, want := subscription.Inputs["target_task_id"], targetTask.ID; got != want {
			t.Fatalf("target_task_id = %v, want %q", got, want)
		}
		if got, want := subscription.Cursors[looppkg.WatchEventsTaskStream], int64(0); got != want {
			t.Fatalf("task cursor = %d, want %d", got, want)
		}
	})

	t.Run("Should enqueue idempotent gap wakes without claiming coordinator runs", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 18, 45, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		created, targetTask := parkWatchEventsLoopForRecoveryTest(ctx, t, globalDB, now)
		appendTaskWatchEventForTest(ctx, t, globalDB, targetTask.ID, now.Add(time.Second), "blocked")
		actor := coordinatorActorContextForTest()

		runs, err := globalDB.EnqueueWatchEventsGapWakes(ctx, actor.Origin, now.Add(2*time.Second))
		if err != nil {
			t.Fatalf("EnqueueWatchEventsGapWakes() error = %v", err)
		}
		if got, want := len(runs), 1; got != want {
			t.Fatalf("gap wake runs = %d, want %d", got, want)
		}
		if got, want := runs[0].LoopRunID, string(created.ID); got != want {
			t.Fatalf("gap wake loop_run_id = %q, want %q", got, want)
		}
		again, err := globalDB.EnqueueWatchEventsGapWakes(ctx, actor.Origin, now.Add(3*time.Second))
		if err != nil {
			t.Fatalf("EnqueueWatchEventsGapWakes(second) error = %v", err)
		}
		if got := len(again); got != 0 {
			t.Fatalf("second gap wake runs = %d, want coalesced 0", got)
		}
		queued, err := globalDB.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
		if err != nil {
			t.Fatalf("ListTaskRunsByStatus(queued) error = %v", err)
		}
		if got, want := len(queued), 1; got != want {
			t.Fatalf("queued coordinator runs = %d, want %d", got, want)
		}
		summaries, err := globalDB.ListEventSummaries(ctx, EventSummaryQuery{Type: watchEventsWakeEnqueuedEvent})
		if err != nil {
			t.Fatalf("ListEventSummaries(wake_enqueued) error = %v", err)
		}
		if got := len(summaries); got == 0 {
			t.Fatal("wake_enqueued summaries = 0, want at least one")
		}
	})

	t.Run("Should include watch-events gap wakes in boot reconcile", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 19, 0, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		created, targetTask := parkWatchEventsLoopForRecoveryTest(ctx, t, globalDB, now)
		appendTaskWatchEventForTest(ctx, t, globalDB, targetTask.ID, now.Add(time.Second), "blocked")
		actor := coordinatorActorContextForTest()

		runs, err := globalDB.ReconcileLoopCoordinatorsOnBoot(ctx, actor.Origin, now.Add(2*time.Second))
		if err != nil {
			t.Fatalf("ReconcileLoopCoordinatorsOnBoot() error = %v", err)
		}
		if got, want := len(runs), 1; got != want {
			t.Fatalf("boot reconcile runs = %d, want %d", got, want)
		}
		if got, want := runs[0].LoopRunID, string(created.ID); got != want {
			t.Fatalf("boot reconcile loop_run_id = %q, want %q", got, want)
		}
	})

	t.Run("Should reconcile network message subscriptions from rows written while down", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 8, 21, 30, 0, 0, time.UTC)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		created := parkWatchEventsLoopWithDefinitionForTest(
			ctx,
			t,
			globalDB,
			now,
			"watch-events-network-recovery",
			map[string]any{
				watchEventsPayloadChannelKey: "builders",
				watchEventsPayloadWorkIDKey:  "work-watch",
			},
			compileNetworkMessageWatchEventsIntegrationDefinitionForTest(t),
		)
		message := networkWatchThreadMessageForTest(
			"ws-a",
			"msg-network-recovery",
			"thread_network_recovery",
			"coder.sess-a",
			"please inspect this work",
			now.Add(time.Second),
		)
		message.PeerTo = "reviewer.sess-a"
		message.WorkID = "work-watch"
		if _, err := globalDB.WriteConversationMessage(ctx, message); err != nil {
			t.Fatalf("WriteConversationMessage(network) error = %v", err)
		}
		foreign := networkWatchThreadMessageForTest(
			"ws-b",
			"msg-network-foreign",
			"thread_network_foreign",
			"coder.sess-b",
			"cross workspace work",
			now.Add(2*time.Second),
		)
		foreign.PeerTo = "reviewer.sess-b"
		foreign.WorkID = "work-watch"
		if _, err := globalDB.WriteConversationMessage(ctx, foreign); err != nil {
			t.Fatalf("WriteConversationMessage(foreign network) error = %v", err)
		}
		actor := coordinatorActorContextForTest()
		runs, err := globalDB.ReconcileLoopCoordinatorsOnBoot(ctx, actor.Origin, now.Add(3*time.Second))
		if err != nil {
			t.Fatalf("ReconcileLoopCoordinatorsOnBoot(network) error = %v", err)
		}
		if got, want := len(runs), 1; got != want {
			t.Fatalf("network boot reconcile runs = %d, want %d", got, want)
		}
		if got, want := runs[0].LoopRunID, string(created.ID); got != want {
			t.Fatalf("network boot reconcile loop_run_id = %q, want %q", got, want)
		}

		events, byNode := claimAndRunWatchEventsWakeForTest(
			ctx,
			t,
			globalDB,
			created,
			compileNetworkMessageWatchEventsIntegrationDefinitionForTest(t),
			runs[0],
			now.Add(4*time.Second),
			"watch_network_messages",
		)
		if got, want := len(events), 1; got != want {
			t.Fatalf("network confirmed events len = %d, want %d: %#v", got, want, events)
		}
		event := events[0]
		if event.Kind != string(hookspkg.HookNetworkMessagePersisted) ||
			event.WorkspaceID != "ws-a" ||
			event.Channel != "builders" ||
			event.WorkID != "work-watch" ||
			event.Stream != looppkg.WatchEventsNetworkStream {
			t.Fatalf("network confirmed event = %#v", event)
		}
		if got, want := event.Payload[watchEventsPayloadMessageIDKey], "msg-network-recovery"; got != want {
			t.Fatalf("network message_id = %v, want %q", got, want)
		}
		if got, want := event.Payload[watchEventsPayloadWorkIDKey], "work-watch"; got != want {
			t.Fatalf("network work_id = %v, want %q", got, want)
		}
		downstream := byNode["summarize"]
		if downstream.Status != watchEventsGenerationOutputEnqueuedForTest || downstream.TaskRunID == "" {
			t.Fatalf("network downstream output = %#v, want enqueued", downstream)
		}
	})
}

func TestGlobalDBWatchEventsHelpers(t *testing.T) {
	t.Parallel()

	t.Run("Should classify supported loop status events", func(t *testing.T) {
		t.Parallel()

		if !loopWatchEventSupported(looppkg.WatchEvent{
			LedgerKind: "status_changed",
			Payload:    map[string]any{"status": string(looppkg.StatusDone)},
		}) {
			t.Fatal("loopWatchEventSupported(done status) = false, want true")
		}
		if loopWatchEventSupported(looppkg.WatchEvent{
			LedgerKind: "status_changed",
			Payload:    map[string]any{"to": string(looppkg.StatusRunning)},
		}) {
			t.Fatal("loopWatchEventSupported(running status) = true, want false")
		}
		if loopWatchEventSupported(looppkg.WatchEvent{LedgerKind: "status_changed"}) {
			t.Fatal("loopWatchEventSupported(missing status) = true, want false")
		}
		if !loopWatchEventSupported(looppkg.WatchEvent{LedgerKind: "node_succeeded"}) {
			t.Fatal("loopWatchEventSupported(node_succeeded) = false, want true")
		}
	})

	t.Run("Should decode watch event payload edge cases", func(t *testing.T) {
		t.Parallel()

		if payload, err := decodeWatchEventPayload(""); err != nil || len(payload) != 0 {
			t.Fatalf("decodeWatchEventPayload(empty) = (%#v, %v), want empty/nil", payload, err)
		}
		if payload, err := decodeWatchEventPayload("null"); err != nil || len(payload) != 0 {
			t.Fatalf("decodeWatchEventPayload(null) = (%#v, %v), want empty/nil", payload, err)
		}
		if _, err := decodeWatchEventPayload("{"); err == nil {
			t.Fatal("decodeWatchEventPayload(malformed) error = nil, want error")
		}
	})

	t.Run("Should build recovery coordinator wake keys", func(t *testing.T) {
		t.Parallel()

		got := watchEventsCoordinatorWakeKey(" loop-run-1 ", " watch_tasks ")
		want := "loop.coordinator.watch_events.loop-run-1.watch_tasks"
		if got != want {
			t.Fatalf("watchEventsCoordinatorWakeKey() = %q, want %q", got, want)
		}
	})

	t.Run("Should normalize benign recovery wake errors", func(t *testing.T) {
		t.Parallel()

		unexpected := errors.New("wake failed")
		cases := []struct {
			name    string
			err     error
			wantNil bool
		}{
			{name: "Should accept nil", err: nil, wantNil: true},
			{name: "Should swallow conflict", err: taskpkg.ErrConflict, wantNil: true},
			{name: "Should swallow invalid transition", err: taskpkg.ErrInvalidStatusTransition, wantNil: true},
			{name: "Should return unexpected errors", err: unexpected, wantNil: false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				err := normalizeWatchEventsGapWakeError(tc.err)
				if (err == nil) != tc.wantNil {
					t.Fatalf("normalizeWatchEventsGapWakeError() = %v, wantNil %v", err, tc.wantNil)
				}
				if !tc.wantNil && !errors.Is(err, unexpected) {
					t.Fatalf("normalizeWatchEventsGapWakeError() = %v, want %v", err, unexpected)
				}
			})
		}
	})

	t.Run("Should summarize recovery event outcomes", func(t *testing.T) {
		t.Parallel()

		subscription := looppkg.ParkedWatchEventSubscription{NodeID: " watch_tasks "}
		cases := []struct {
			name      string
			eventType string
			want      string
		}{
			{
				name:      "Should summarize a matched gap",
				eventType: watchEventsMatchedEvent,
				want:      "watch-events gap matched for watch_tasks",
			},
			{
				name:      "Should summarize a wake error",
				eventType: watchEventsWakeErrorEvent,
				want:      "watch-events gap wake failed for watch_tasks",
			},
			{
				name:      "Should summarize an enqueued wake",
				eventType: watchEventsWakeEnqueuedEvent,
				want:      "watch-events gap wake enqueued for watch_tasks",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if got := watchEventsGapSummary(tc.eventType, subscription); got != tc.want {
					t.Fatalf("watchEventsGapSummary() = %q, want %q", got, tc.want)
				}
			})
		}
	})

	t.Run("Should stringify optional recovery errors", func(t *testing.T) {
		t.Parallel()

		if got := watchEventsErrorString(nil); got != "" {
			t.Fatalf("watchEventsErrorString(nil) = %q, want empty", got)
		}
		if got, want := watchEventsErrorString(errors.New("boom")), "boom"; got != want {
			t.Fatalf("watchEventsErrorString(error) = %q, want %q", got, want)
		}
	})
}

func parkWatchEventsLoopForRecoveryTest(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	now time.Time,
) (looppkg.Run, taskpkg.Task) {
	t.Helper()
	targetTask := workspaceTaskRecordForTest("watch-recovery-target", "ws-a")
	if err := globalDB.CreateTask(ctx, targetTask); err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}
	loopRun := testLoopRun("watch-events-recovery", now, looppkg.StatusRunning)
	loopRun.WorkspaceID = "ws-a"
	loopRun.Inputs = map[string]any{"target_task_id": targetTask.ID}
	created, err := globalDB.CreateLoopRunForStart(ctx, loopRun, dsl.ConcurrencyAllow)
	if err != nil {
		t.Fatalf("CreateLoopRunForStart() error = %v", err)
	}
	runner := newGlobalDBWatchEventsCoordinatorForTest(
		t,
		globalDB,
		compileWatchEventsIntegrationDefinitionForTest(t),
	)
	claim := claimCoordinatorRunForTest(ctx, t, globalDB, created.ID, "run-watch-events-recovery-first", now)
	plan, err := runner.Run(ctx, taskpkg.RunID(claim.Run.ID))
	if err != nil {
		t.Fatalf("Run(first) error = %v", err)
	}
	if plan.Terminal == nil || plan.Terminal.Status != string(looppkg.StatusWatching) {
		t.Fatalf("first terminal = %#v, want watching", plan.Terminal)
	}
	result, err := globalDB.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Actor:      coordinatorActorContextForTest(),
		Plan:       plan,
		Now:        now.Add(time.Second),
	}, looppkg.NewStoreFinalizer())
	if err != nil {
		t.Fatalf("CompleteCoordinatorAndEnqueueNext(first) error = %v", err)
	}
	if got, want := coordinatorResultStatus(t, &result), string(looppkg.StatusWatching); got != want {
		t.Fatalf("loop status = %q, want %q", got, want)
	}
	return created, targetTask
}

func parkWatchEventsLoopWithDefinitionForTest(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	now time.Time,
	loopRunID string,
	inputs map[string]any,
	resolved *looppkg.ResolvedDefinition,
) looppkg.Run {
	t.Helper()
	loopRun := testLoopRun(loopRunID, now, looppkg.StatusRunning)
	loopRun.WorkspaceID = "ws-a"
	loopRun.Inputs = inputs
	created, err := globalDB.CreateLoopRunForStart(ctx, loopRun, dsl.ConcurrencyAllow)
	if err != nil {
		t.Fatalf("CreateLoopRunForStart(%s) error = %v", loopRunID, err)
	}
	runner := newGlobalDBWatchEventsCoordinatorForTest(t, globalDB, resolved)
	claim := claimCoordinatorRunForTest(ctx, t, globalDB, created.ID, "run-"+loopRunID+"-first", now)
	plan, err := runner.Run(ctx, taskpkg.RunID(claim.Run.ID))
	if err != nil {
		t.Fatalf("Run(%s first) error = %v", loopRunID, err)
	}
	if plan.Terminal == nil || plan.Terminal.Status != string(looppkg.StatusWatching) {
		t.Fatalf("%s first terminal = %#v, want watching", loopRunID, plan.Terminal)
	}
	result, err := globalDB.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Actor:      coordinatorActorContextForTest(),
		Plan:       plan,
		Now:        now.Add(time.Second),
	}, looppkg.NewStoreFinalizer())
	if err != nil {
		t.Fatalf("CompleteCoordinatorAndEnqueueNext(%s first) error = %v", loopRunID, err)
	}
	if got, want := coordinatorResultStatus(t, &result), string(looppkg.StatusWatching); got != want {
		t.Fatalf("%s loop status = %q, want %q", loopRunID, got, want)
	}
	return created
}

func claimAndRunWatchEventsWakeForTest(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	loopRun looppkg.Run,
	resolved *looppkg.ResolvedDefinition,
	wakeRun taskpkg.Run,
	now time.Time,
	watchNodeID string,
) ([]looppkg.WatchEvent, map[string]looppkg.GenerationOutput) {
	t.Helper()
	runner := newGlobalDBWatchEventsCoordinatorForTest(t, globalDB, resolved)
	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		RunID:            wakeRun.ID,
		Scope:            taskpkg.ScopeWorkspace,
		WorkspaceID:      string(loopRun.WorkspaceID),
		RunKind:          taskpkg.RunKindCoordinator,
		ClaimerSessionID: "daemon-loop-" + wakeRun.ID,
		ClaimedBy:        &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "loop"},
		LeaseDuration:    time.Minute,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(%s) error = %v", wakeRun.ID, err)
	}
	plan, err := runner.Run(ctx, taskpkg.RunID(claim.Run.ID))
	if err != nil {
		t.Fatalf("Run(%s) error = %v", wakeRun.ID, err)
	}
	if plan.Terminal != nil || plan.Yield {
		t.Fatalf("wake plan Terminal=%#v Yield=%v, want downstream enqueue", plan.Terminal, plan.Yield)
	}
	if got, want := len(plan.NodeRuns), 1; got != want {
		t.Fatalf("wake NodeRuns = %d, want %d", got, want)
	}
	result, err := globalDB.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Actor:      coordinatorActorContextForTest(),
		Plan:       plan,
		Now:        now.Add(time.Second),
	}, looppkg.NewStoreFinalizer())
	if err != nil {
		t.Fatalf("CompleteCoordinatorAndEnqueueNext(%s) error = %v", wakeRun.ID, err)
	}
	if got, want := len(result.EnqueuedRuns), 1; got != want {
		t.Fatalf("wake EnqueuedRuns = %d, want %d", got, want)
	}
	outputs, err := globalDB.ListGenerationOutputs(ctx, loopRun.ID, 1)
	if err != nil {
		t.Fatalf("ListGenerationOutputs(%s) error = %v", loopRun.ID, err)
	}
	byNode := watchEventsGenerationOutputsByNode(outputs)
	watchOutput := byNode[watchNodeID]
	if watchOutput.Status != "succeeded" {
		t.Fatalf("%s output status = %q, want succeeded", watchNodeID, watchOutput.Status)
	}
	confirmed := decodeWatchEventsConfirmedRefForTest(t, watchOutput.OutputRef)
	var events []looppkg.WatchEvent
	if err := json.Unmarshal(confirmed.Events, &events); err != nil {
		t.Fatalf("Unmarshal confirmed events error = %v", err)
	}
	return events, byNode
}

func appendTaskWatchEventForTest(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	taskID string,
	at time.Time,
	toStatus string,
) {
	t.Helper()
	err := globalDB.withTaskImmediateTransaction(
		ctx,
		"append task watch event test",
		func(exec taskSQLExecutor) error {
			return appendTaskEventPayloadWithExecutor(
				ctx,
				exec,
				taskID,
				"",
				string(hookspkg.HookTaskStatusChanged),
				coordinatorActorContextForTest(),
				at,
				map[string]string{
					"from_status": string(taskpkg.TaskStatusPending),
					"to_status":   toStatus,
				},
			)
		},
	)
	if err != nil {
		t.Fatalf("appendTaskEventPayloadWithExecutor() error = %v", err)
	}
}

func networkWatchThreadMessageForTest(
	workspaceID string,
	messageID string,
	threadID string,
	peerFrom string,
	text string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: workspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
		Direction:   "sent",
		PeerFrom:    peerFrom,
		Kind:        store.NetworkKindSay,
		Text:        text,
		PreviewText: text,
		Body:        []byte(`{"text":"` + text + `"}`),
		Timestamp:   timestamp,
	}
}

func networkWatchDirectIDForTest(workspaceID, channel, peerA, peerB string) (string, error) {
	directID, _, _, err := store.NetworkDirectRoomIdentity(workspaceID, channel, peerA, peerB)
	if err != nil {
		return "", fmt.Errorf("derive network direct room identity: %w", err)
	}
	return directID, nil
}

func networkWatchDirectMessageForTest(
	workspaceID string,
	messageID string,
	directID string,
	peerFrom string,
	peerTo string,
	text string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: workspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceDirect,
		DirectID:    directID,
		Direction:   "sent",
		PeerFrom:    peerFrom,
		PeerTo:      peerTo,
		Kind:        store.NetworkKindSay,
		Text:        text,
		PreviewText: text,
		Body:        []byte(`{"text":"` + text + `"}`),
		Timestamp:   timestamp,
	}
}

func networkWatchTraceMessageForTest(
	workspaceID string,
	messageID string,
	threadID string,
	peerFrom string,
	workID string,
	state string,
	timestamp time.Time,
) store.NetworkConversationMessage {
	return store.NetworkConversationMessage{
		MessageID:   messageID,
		SessionID:   "sess-" + messageID,
		WorkspaceID: workspaceID,
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    threadID,
		Direction:   "received",
		PeerFrom:    peerFrom,
		Kind:        store.NetworkKindTrace,
		WorkID:      workID,
		PreviewText: state,
		Body:        []byte(`{"state":"` + state + `"}`),
		Timestamp:   timestamp,
	}
}

func compileWatchEventsIntegrationDefinitionForTest(t *testing.T) *looppkg.ResolvedDefinition {
	t.Helper()
	resolved, err := looppkg.NewCompiler().Compile(dsl.Definition{
		APIVersion: dsl.APIVersion,
		Kind:       dsl.KindLoop,
		Inputs: map[string]dsl.Input{
			"target_task_id": {Type: dsl.InputTypeString},
		},
		Graph: dsl.Graph{
			Nodes: []dsl.Node{
				{
					ID:    "watch_tasks",
					Class: dsl.NodeClassSource,
					Kind:  string(dsl.SourceWatchEvents),
					Events: []dsl.EventSubscription{{
						Kind: string(hookspkg.HookTaskStatusChanged),
						Filter: "event.task_id == inputs.target_task_id" +
							" && " + `event.payload.to_status == "blocked"`,
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
	})
	if err != nil {
		t.Fatalf("Compile(watch-events integration) error = %v", err)
	}
	return resolved
}

func compileAutomationWatchEventsIntegrationDefinitionForTest(t *testing.T) *looppkg.ResolvedDefinition {
	t.Helper()
	resolved, err := looppkg.NewCompiler().Compile(dsl.Definition{
		APIVersion: dsl.APIVersion,
		Kind:       dsl.KindLoop,
		Inputs: map[string]dsl.Input{
			watchEventsPayloadJobIDKey: {Type: dsl.InputTypeString},
		},
		Graph: dsl.Graph{
			Nodes: []dsl.Node{
				{
					ID:    "watch_automation",
					Class: dsl.NodeClassSource,
					Kind:  string(dsl.SourceWatchEvents),
					Events: []dsl.EventSubscription{{
						Kind:   string(hookspkg.HookAutomationRunCompleted),
						Filter: "event.payload.job_id == inputs.job_id",
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
			Edges: []dsl.Edge{{From: "watch_automation", To: "summarize"}},
		},
	})
	if err != nil {
		t.Fatalf("Compile(automation watch-events integration) error = %v", err)
	}
	return resolved
}

func compileNetworkMessageWatchEventsIntegrationDefinitionForTest(t *testing.T) *looppkg.ResolvedDefinition {
	t.Helper()
	resolved, err := looppkg.NewCompiler().Compile(dsl.Definition{
		APIVersion: dsl.APIVersion,
		Kind:       dsl.KindLoop,
		Inputs: map[string]dsl.Input{
			watchEventsPayloadChannelKey: {Type: dsl.InputTypeString},
			watchEventsPayloadWorkIDKey:  {Type: dsl.InputTypeString},
		},
		Graph: dsl.Graph{
			Nodes: []dsl.Node{
				{
					ID:    "watch_network_messages",
					Class: dsl.NodeClassSource,
					Kind:  string(dsl.SourceWatchEvents),
					Events: []dsl.EventSubscription{{
						Kind: string(hookspkg.HookNetworkMessagePersisted),
						Filter: "event.channel == inputs.channel" +
							" && " + "event.payload.work_id == inputs.work_id",
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
			Edges: []dsl.Edge{{From: "watch_network_messages", To: "summarize"}},
		},
	})
	if err != nil {
		t.Fatalf("Compile(network watch-events integration) error = %v", err)
	}
	return resolved
}

func newGlobalDBWatchEventsCoordinatorForTest(
	t *testing.T,
	globalDB *GlobalDB,
	resolved *looppkg.ResolvedDefinition,
) *looppkg.CoordinatorRunner {
	t.Helper()
	runner, err := looppkg.NewCoordinatorRunner(
		globalDB,
		globalDB,
		globalDB,
		looppkg.DefinitionResolverFunc(
			func(context.Context, looppkg.WorkspaceID, string) (*looppkg.ResolvedDefinition, error) {
				return resolved, nil
			},
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		looppkg.WithCoordinatorWatchEventsLedger(globalDB),
	)
	if err != nil {
		t.Fatalf("NewCoordinatorRunner() error = %v", err)
	}
	return runner
}

func watchEventsGenerationOutputsByNode(
	outputs []looppkg.GenerationOutput,
) map[string]looppkg.GenerationOutput {
	byNode := make(map[string]looppkg.GenerationOutput, len(outputs))
	for _, output := range outputs {
		byNode[output.NodeID] = output
	}
	return byNode
}

type watchEventsConfirmedRefForTest struct {
	Kind    string           `json:"kind"`
	Events  json.RawMessage  `json:"events"`
	Cursors map[string]int64 `json:"cursors"`
}

func decodeWatchEventsConfirmedRefForTest(
	t *testing.T,
	ref string,
) watchEventsConfirmedRefForTest {
	t.Helper()
	var confirmed watchEventsConfirmedRefForTest
	if err := json.Unmarshal([]byte(ref), &confirmed); err != nil {
		t.Fatalf("Unmarshal watch-events confirmed ref error = %v", err)
	}
	if confirmed.Kind != "watch_events_confirmed" {
		t.Fatalf("watch-events ref kind = %q, want watch_events_confirmed", confirmed.Kind)
	}
	return confirmed
}

func watchEventCountsByStream(events []looppkg.WatchEvent) map[string]int {
	counts := make(map[string]int)
	for _, event := range events {
		counts[event.Stream]++
	}
	return counts
}

func assertWatchEventRFC3339UTC(t *testing.T, value string) {
	t.Helper()
	if !strings.HasSuffix(value, "Z") {
		t.Fatalf("watch event at = %q, want UTC RFC3339 suffix", value)
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}
}

func TestGlobalDBWatchEventsReadMatchesShouldRejectInvalidQuery(t *testing.T) {
	t.Parallel()

	t.Run("Should reject unsupported watch-events streams", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openFreshLoopTestGlobalDB(t, "ws-a")
		_, err := globalDB.ReadMatches(ctx, looppkg.WatchEventsQuery{
			WorkspaceID: "ws-a",
			Streams:     map[string]int64{"unsupported": 0},
			Kinds:       []string{string(hookspkg.HookTaskStatusChanged)},
			Limit:       1,
		})
		if err == nil {
			t.Fatal("ReadMatches() error = nil, want validation error")
		}
	})
}
