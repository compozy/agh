package globaldb

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

type recordingTaskEventCommitObserver struct {
	db      *GlobalDB
	records []taskpkg.EventRecord
	tasks   []taskpkg.Task
	err     error
}

func (o *recordingTaskEventCommitObserver) OnTaskEvent(ctx context.Context, record taskpkg.EventRecord) {
	o.records = append(o.records, record)
	if o.db == nil || o.err != nil {
		return
	}
	taskRecord, err := o.db.GetTask(ctx, record.Event.TaskID)
	if err != nil {
		o.err = err
		return
	}
	o.tasks = append(o.tasks, taskRecord)
}

func TestGlobalDBTaskEventCommitObserverShouldPublishRecoveredAfterCommit(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-recovered-commit-observer")
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	markedAt := taskRecord.UpdatedAt.Add(time.Minute)
	if _, err := globalDB.MarkTaskNeedsAttention(ctx, taskpkg.NeedsAttentionMutation{
		Origin:   coordinatorActorContextForTest().Origin,
		TaskID:   taskRecord.ID,
		Reason:   "operator input required",
		Actor:    taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "scheduler"},
		MarkedAt: markedAt,
	}); err != nil {
		t.Fatalf("MarkTaskNeedsAttention() error = %v", err)
	}

	observer := &recordingTaskEventCommitObserver{db: globalDB}
	globalDB.SetTaskEventCommitObserver(observer)
	if _, err := globalDB.ClearTaskNeedsAttention(ctx, taskpkg.NeedsAttentionClearMutation{
		Origin:    operatorActorContextForTest("operator").Origin,
		TaskID:    taskRecord.ID,
		ClearedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "operator"},
		ClearedAt: markedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("ClearTaskNeedsAttention() error = %v", err)
	}
	if observer.err != nil {
		t.Fatalf("observer GetTask() error = %v", observer.err)
	}
	if got, want := len(observer.records), 1; got != want {
		t.Fatalf("len(observer.records) = %d, want %d", got, want)
	}
	if got, want := observer.records[0].Event.EventType, string(hookspkg.HookTaskRecovered); got != want {
		t.Fatalf("observer event type = %q, want %q", got, want)
	}
	if got, want := len(observer.tasks), 1; got != want {
		t.Fatalf("len(observer.tasks) = %d, want %d", got, want)
	}
	if observer.tasks[0].NeedsAttention != nil {
		t.Fatalf("observer task NeedsAttention = %#v, want committed nil", observer.tasks[0].NeedsAttention)
	}
}

func TestGlobalDBUpdateTaskStatusShouldAppendStatusChangedEvent(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-status-changed")
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	updated := taskRecord
	updated.Status = taskpkg.TaskStatusReady
	updated.UpdatedAt = taskRecord.UpdatedAt.Add(time.Minute)
	if err := globalDB.UpdateTask(ctx, updated, operatorActorContextForTest("user:alice")); err != nil {
		t.Fatalf("UpdateTask(status) error = %v", err)
	}

	event := requireTaskEventRecordForTest(t, globalDB, taskRecord.ID, string(hookspkg.HookTaskStatusChanged))
	if got, want := event.Event.Timestamp, updated.UpdatedAt; !got.Equal(want) {
		t.Fatalf("status_changed timestamp = %s, want task UpdatedAt %s", got, want)
	}
	var payload struct {
		FromStatus string `json:"from_status"`
		ToStatus   string `json:"to_status"`
	}
	if err := json.Unmarshal(event.Event.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal(status_changed payload) error = %v", err)
	}
	if got, want := payload.FromStatus, string(taskpkg.TaskStatusPending); got != want {
		t.Fatalf("from_status = %q, want %q", got, want)
	}
	if got, want := payload.ToStatus, string(taskpkg.TaskStatusReady); got != want {
		t.Fatalf("to_status = %q, want %q", got, want)
	}
}

func TestGlobalDBCompleteRunLeaseShouldAppendRunCompletedWatchEvent(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	globalDB := openTestGlobalDB(t)
	taskRecord := taskRecordForTest("task-run-watch-completed")
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	rawToken := "claim-token-watch-completed"
	leased := storeLeasedTaskRunForBlockTest(
		ctx,
		t,
		globalDB,
		taskRecord.ID,
		"run-watch-completed",
		"sess-watch-completed",
		rawToken,
		time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC),
	)

	if _, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
		Actor:      coordinatorActorContextForTest(),
		RunID:      leased.ID,
		ClaimToken: rawToken,
		Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        leased.LeaseUntil.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("CompleteRunLease() error = %v", err)
	}

	event := requireTaskEventRecordForTest(t, globalDB, taskRecord.ID, string(hookspkg.HookTaskRunCompleted))
	if got, want := event.Event.RunID, leased.ID; got != want {
		t.Fatalf("run_id = %q, want %q", got, want)
	}
}

func TestGlobalDBTaskEventAppendFailureShouldRollbackOwningState(t *testing.T) {
	t.Parallel()

	t.Run("Should roll back status update when status_changed append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-status-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskStatusChanged))

		updated := taskRecord
		updated.Status = taskpkg.TaskStatusReady
		updated.UpdatedAt = taskRecord.UpdatedAt.Add(time.Minute)
		err := globalDB.UpdateTask(ctx, updated, operatorActorContextForTest("user:alice"))
		assertForcedTaskEventInsertError(t, err, "UpdateTask(status)")
		stored, err := globalDB.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if got, want := stored.Status, taskpkg.TaskStatusPending; got != want {
			t.Fatalf("stored.Status = %q, want %q", got, want)
		}
	})

	t.Run("Should roll back child completion when parent rollup event append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		parent := taskRecordForTest("task-parent-rollup-rollback")
		if err := globalDB.CreateTask(ctx, parent); err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}
		parentRun := taskRunForTest("run-parent-rollup-rollback", parent.ID)
		if err := globalDB.CreateTaskRun(ctx, parentRun); err != nil {
			t.Fatalf("CreateTaskRun(parent) error = %v", err)
		}
		if _, err := globalDB.MarkTaskRunNeedsAttention(ctx, parentRun.ID, "starved"); err != nil {
			t.Fatalf("MarkTaskRunNeedsAttention(parent) error = %v", err)
		}

		completedChild := taskRecordForTest("task-child-completed-rollup-rollback")
		completedChild.ParentTaskID = parent.ID
		completedChild.Status = taskpkg.TaskStatusCompleted
		if err := globalDB.CreateTask(ctx, completedChild); err != nil {
			t.Fatalf("CreateTask(completed child) error = %v", err)
		}
		settlingChild := taskRecordForTest("task-child-settling-rollup-rollback")
		settlingChild.ParentTaskID = parent.ID
		if err := globalDB.CreateTask(ctx, settlingChild); err != nil {
			t.Fatalf("CreateTask(settling child) error = %v", err)
		}
		rawToken := "claim-token-parent-rollup-rollback"
		leased := storeLeasedTaskRunForBlockTest(
			ctx,
			t,
			globalDB,
			settlingChild.ID,
			"run-child-settling-rollup-rollback",
			"sess-child-settling-rollup-rollback",
			rawToken,
			time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC),
		)

		observer := &recordingTaskEventCommitObserver{db: globalDB}
		globalDB.SetTaskEventCommitObserver(observer)
		installTaskEventInsertFailureTriggerForTaskAndType(
			t,
			globalDB,
			parent.ID,
			string(hookspkg.HookTaskStatusChanged),
		)
		_, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			Actor:      coordinatorActorContextForTest(),
			RunID:      leased.ID,
			ClaimToken: rawToken,
			Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
			Now:        leased.LeaseUntil.Add(-time.Minute),
		})
		assertForcedTaskEventInsertError(t, err, "CompleteRunLease(parent rollup)")

		storedChildRun, err := globalDB.GetTaskRun(ctx, leased.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(child) error = %v", err)
		}
		if got, want := storedChildRun.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("child run status = %q, want rollback to %q", got, want)
		}
		storedChild, err := globalDB.GetTask(ctx, settlingChild.ID)
		if err != nil {
			t.Fatalf("GetTask(child) error = %v", err)
		}
		if got, want := storedChild.Status, taskpkg.TaskStatusPending; got != want {
			t.Fatalf("child task status = %q, want rollback to %q", got, want)
		}
		storedParent, err := globalDB.GetTask(ctx, parent.ID)
		if err != nil {
			t.Fatalf("GetTask(parent) error = %v", err)
		}
		if got, want := storedParent.Status, taskpkg.TaskStatusPending; got != want {
			t.Fatalf("parent task status = %q, want rollback to %q", got, want)
		}
		storedParentRun, err := globalDB.GetTaskRun(ctx, parentRun.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(parent) error = %v", err)
		}
		if got, want := storedParentRun.Status, taskpkg.TaskRunStatusNeedsAttention; got != want {
			t.Fatalf("parent run status = %q, want rollback to %q", got, want)
		}
		if got := len(observer.records); got != 0 {
			t.Fatalf("len(observer.records) after rollback = %d, want 0", got)
		}
	})

	t.Run("Should roll back block creation when task.blocked append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-block-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskBlocked))

		_, err := globalDB.CreateTaskBlock(ctx, taskpkg.CreateTaskBlockMutation{
			Actor: coordinatorActorContextForTest(),
			Block: taskBlockRecordForTest(
				"block-rollback",
				taskRecord.ID,
				taskpkg.BlockKindNeedsInput,
				taskRecord.UpdatedAt,
			),
			RecurrenceLimit: 2,
		})
		assertForcedTaskEventInsertError(t, err, "CreateTaskBlock()")
		blocks, err := globalDB.ListTaskBlocks(ctx, taskRecord.ID, true)
		if err != nil {
			t.Fatalf("ListTaskBlocks() error = %v", err)
		}
		if len(blocks) != 0 {
			t.Fatalf("blocks = %#v, want rollback to remove block row", blocks)
		}
	})

	t.Run("Should roll back block clear when task.unblocked append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-unblocked-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		created, err := globalDB.CreateTaskBlock(ctx, taskpkg.CreateTaskBlockMutation{
			Actor: coordinatorActorContextForTest(),
			Block: taskBlockRecordForTest(
				"block-unblocked-rollback",
				taskRecord.ID,
				taskpkg.BlockKindTransient,
				taskRecord.UpdatedAt,
			),
			RecurrenceLimit: 2,
		})
		if err != nil {
			t.Fatalf("CreateTaskBlock(setup) error = %v", err)
		}
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskUnblocked))

		err = clearTaskBlockForRollbackTest(
			ctx,
			globalDB,
			taskRecord.ID,
			created.Block.ID,
			taskRecord.UpdatedAt.Add(time.Minute),
		)
		assertForcedTaskEventInsertError(t, err, "ClearTaskBlock()")
		blocks, err := globalDB.ListTaskBlocks(ctx, taskRecord.ID, false)
		if err != nil {
			t.Fatalf("ListTaskBlocks(open) error = %v", err)
		}
		assertTaskBlockIDs(t, blocks, []string{created.Block.ID})
	})

	t.Run("Should roll back breaker escalation when task.needs_attention append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-attention-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		first, err := globalDB.CreateTaskBlock(ctx, taskpkg.CreateTaskBlockMutation{
			Actor: coordinatorActorContextForTest(),
			Block: taskBlockRecordForTest(
				"block-attention-first",
				taskRecord.ID,
				taskpkg.BlockKindNeedsInput,
				taskRecord.UpdatedAt,
			),
			RecurrenceLimit: 1,
		})
		if err != nil {
			t.Fatalf("CreateTaskBlock(first) error = %v", err)
		}
		if err := clearTaskBlockForRollbackTest(
			ctx,
			globalDB,
			taskRecord.ID,
			first.Block.ID,
			taskRecord.UpdatedAt.Add(time.Minute),
		); err != nil {
			t.Fatalf("ClearTaskBlock(first) error = %v", err)
		}
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskNeedsAttention))

		_, err = globalDB.CreateTaskBlock(ctx, taskpkg.CreateTaskBlockMutation{
			Actor: coordinatorActorContextForTest(),
			Block: taskBlockRecordForTest(
				"block-attention-second",
				taskRecord.ID,
				taskpkg.BlockKindNeedsInput,
				taskRecord.UpdatedAt.Add(2*time.Minute),
			),
			RecurrenceLimit: 1,
		})
		assertForcedTaskEventInsertError(t, err, "CreateTaskBlock(escalating)")
		stored, err := globalDB.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if stored.NeedsAttention != nil {
			t.Fatalf("NeedsAttention = %#v, want rollback to keep task clear", stored.NeedsAttention)
		}
		blocks, err := globalDB.ListTaskBlocks(ctx, taskRecord.ID, true)
		if err != nil {
			t.Fatalf("ListTaskBlocks(all) error = %v", err)
		}
		assertTaskBlockIDs(t, blocks, []string{first.Block.ID})
	})

	t.Run("Should roll back attention clear when task.recovered append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-recovered-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		markedAt := taskRecord.UpdatedAt.Add(time.Minute)
		if _, err := globalDB.MarkTaskNeedsAttention(ctx, taskpkg.NeedsAttentionMutation{
			Origin:   coordinatorActorContextForTest().Origin,
			TaskID:   taskRecord.ID,
			Reason:   "operator input required",
			Actor:    taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "scheduler"},
			MarkedAt: markedAt,
		}); err != nil {
			t.Fatalf("MarkTaskNeedsAttention() error = %v", err)
		}
		observer := &recordingTaskEventCommitObserver{db: globalDB}
		globalDB.SetTaskEventCommitObserver(observer)
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskRecovered))

		_, err := globalDB.ClearTaskNeedsAttention(ctx, taskpkg.NeedsAttentionClearMutation{
			Origin:    operatorActorContextForTest("operator").Origin,
			TaskID:    taskRecord.ID,
			ClearedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "operator"},
			ClearedAt: markedAt.Add(time.Minute),
		})
		assertForcedTaskEventInsertError(t, err, "ClearTaskNeedsAttention()")
		stored, err := globalDB.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if stored.NeedsAttention == nil {
			t.Fatal("NeedsAttention = nil, want rollback to keep escalation metadata")
		}
		if got := len(observer.records); got != 0 {
			t.Fatalf("len(observer.records) after rollback = %d, want 0", got)
		}
	})

	t.Run("Should roll back lease completion when task.run.completed append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-run-completed-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		rawToken := "claim-token-completed-rollback"
		leased := storeLeasedTaskRunForBlockTest(
			ctx,
			t,
			globalDB,
			taskRecord.ID,
			"run-completed-rollback",
			"sess-completed-rollback",
			rawToken,
			time.Date(2026, 4, 14, 15, 0, 0, 0, time.UTC),
		)
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskRunCompleted))

		_, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			Actor:      coordinatorActorContextForTest(),
			RunID:      leased.ID,
			ClaimToken: rawToken,
			Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
			Now:        leased.LeaseUntil.Add(-time.Minute),
		})
		assertForcedTaskEventInsertError(t, err, "CompleteRunLease()")
		stored, err := globalDB.GetTaskRun(ctx, leased.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := stored.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("stored.Status = %q, want rollback to %q", got, want)
		}
		if stored.Result != nil || !stored.EndedAt.IsZero() {
			t.Fatalf("stored terminal fields = result %s ended_at %v, want rollback", stored.Result, stored.EndedAt)
		}
	})

	t.Run("Should roll back lease failure when task.run.failed append fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		taskRecord := taskRecordForTest("task-run-failed-rollback")
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		rawToken := "claim-token-failed-rollback"
		leased := storeLeasedTaskRunForBlockTest(
			ctx,
			t,
			globalDB,
			taskRecord.ID,
			"run-failed-rollback",
			"sess-failed-rollback",
			rawToken,
			time.Date(2026, 4, 14, 16, 0, 0, 0, time.UTC),
		)
		installTaskEventInsertFailureTriggerForType(t, globalDB, string(hookspkg.HookTaskRunFailed))

		_, err := globalDB.FailRunLease(ctx, taskpkg.LeaseFailure{
			Actor:      coordinatorActorContextForTest(),
			RunID:      leased.ID,
			ClaimToken: rawToken,
			Failure:    taskpkg.RunFailure{Error: "worker failed"},
			Now:        leased.LeaseUntil.Add(-time.Minute),
		})
		assertForcedTaskEventInsertError(t, err, "FailRunLease()")
		stored, err := globalDB.GetTaskRun(ctx, leased.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := stored.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("stored.Status = %q, want rollback to %q", got, want)
		}
		if stored.Error != "" || !stored.EndedAt.IsZero() {
			t.Fatalf("stored terminal fields = error %q ended_at %v, want rollback", stored.Error, stored.EndedAt)
		}
	})
}

func assertForcedTaskEventInsertError(t *testing.T, err error, operation string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), "forced task event insert failure") {
		t.Fatalf("%s error = %v, want forced task event insert failure", operation, err)
	}
}

func clearTaskBlockForRollbackTest(
	ctx context.Context,
	globalDB *GlobalDB,
	taskID string,
	blockID string,
	clearedAt time.Time,
) error {
	_, err := globalDB.ClearTaskBlock(ctx, taskpkg.ClearTaskBlockMutation{
		TaskID:    taskID,
		BlockID:   blockID,
		ClearedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:resolver"},
		ClearedAt: clearedAt,
		ClearNote: "resolved by operator",
		Actor:     operatorActorContextForTest("user:resolver"),
	})
	return err
}

func installTaskEventInsertFailureTriggerForType(t *testing.T, globalDB *GlobalDB, eventType string) {
	t.Helper()

	_, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`CREATE TRIGGER fail_task_event_insert
		 BEFORE INSERT ON task_events
		 WHEN NEW.event_type = '`+strings.ReplaceAll(eventType, "'", "''")+`'
		 BEGIN
		 	SELECT RAISE(ABORT, 'forced task event insert failure');
		 END;`,
	)
	if err != nil {
		t.Fatalf("install task_event insert failure trigger error = %v", err)
	}
}

func installTaskEventInsertFailureTriggerForTaskAndType(
	t *testing.T,
	globalDB *GlobalDB,
	taskID string,
	eventType string,
) {
	t.Helper()

	_, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`CREATE TRIGGER fail_task_event_insert_for_task
		 BEFORE INSERT ON task_events
		 WHEN NEW.task_id = '`+strings.ReplaceAll(taskID, "'", "''")+`'
		  AND NEW.event_type = '`+strings.ReplaceAll(eventType, "'", "''")+`'
		 BEGIN
		 SELECT RAISE(ABORT, 'forced task event insert failure');
		 END;`,
	)
	if err != nil {
		t.Fatalf("install task_event task/type failure trigger error = %v", err)
	}
}

func requireTaskEventRecordForTest(
	t *testing.T,
	globalDB *GlobalDB,
	taskID string,
	eventType string,
) taskpkg.EventRecord {
	t.Helper()

	records, err := globalDB.ListTaskEventRecords(testutil.Context(t), taskpkg.EventRecordQuery{TaskID: taskID})
	if err != nil {
		t.Fatalf("ListTaskEventRecords() error = %v", err)
	}
	for _, record := range records {
		if record.Event.EventType == eventType {
			return record
		}
	}
	t.Fatalf("task event %q not found in %#v", eventType, records)
	return taskpkg.EventRecord{}
}
