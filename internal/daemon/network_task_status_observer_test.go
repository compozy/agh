package daemon

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestNetworkTaskStatusObserver(t *testing.T) {
	t.Parallel()

	t.Run("Should post status back and persist completed designation rollup", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		seedNetworkStatusObserverThread(ctx, t, db, now)
		actor := taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "daemon.test"}
		origin := taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "network-status-test"}
		taskRecord := taskpkg.Task{
			ID:          "task-status",
			Scope:       taskpkg.ScopeGlobal,
			Title:       "Investigate latency",
			MaxAttempts: 3,
			Status:      taskpkg.TaskStatusInProgress,
			CreatedBy:   actor,
			Origin:      origin,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := db.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		first := networkStatusObserverRun(
			t,
			"run-status-a",
			taskRecord.ID,
			taskpkg.TaskRunStatusCompleted,
			0,
			"Inspect API latency",
			now,
			origin,
		)
		second := networkStatusObserverRun(
			t,
			"run-status-b",
			taskRecord.ID,
			taskpkg.TaskRunStatusFailed,
			1,
			"Inspect worker latency",
			now,
			origin,
		)
		second.Error = "worker timeout"
		for _, run := range []taskpkg.Run{first, second} {
			if err := db.CreateTaskRun(ctx, run); err != nil {
				t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
			}
		}
		if err := db.PutNetworkTaskThreadOrigin(ctx, store.NetworkTaskThreadOrigin{
			TaskID:           taskRecord.ID,
			WorkspaceID:      "wks_status",
			Channel:          "builders",
			ThreadID:         "thread_status",
			OriginMessageID:  "msg-origin",
			Digest:           "Investigate latency",
			SourceMessageIDs: []string{"msg-origin"},
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutNetworkTaskThreadOrigin() error = %v", err)
		}

		runtime := &fakeNetworkRuntime{}
		observer := &networkTaskStatusObserver{
			network: runtime,
			tasks:   db,
			prefs:   db,
			logger:  discardLogger(),
			now: func() time.Time {
				return now.Add(time.Minute)
			},
		}
		err := observer.processWithContext(ctx, taskpkg.EventRecord{Event: taskpkg.Event{
			ID:        "evt-run-failed",
			TaskID:    taskRecord.ID,
			RunID:     second.ID,
			EventType: taskEventRunFailed,
			Actor:     actor,
			Origin:    origin,
			Timestamp: now.Add(time.Second),
		}})
		if err != nil {
			t.Fatalf("processWithContext() error = %v", err)
		}

		runtime.mu.Lock()
		calls := append([]network.RuntimeSendRequest(nil), runtime.runtimeSendCalls...)
		runtime.mu.Unlock()
		if got, want := len(calls), 1; got != want {
			t.Fatalf("len(runtime sends) = %d, want %d: %#v", got, want, calls)
		}
		assertNetworkTaskStatusSend(t, calls[0], "task_fanout_rollup", "Task fan-out complete")

		rollups, err := db.ListTaskDesignationRollups(
			ctx,
			store.TaskDesignationRollupQuery{DesignationGroupID: "tdg-status", Limit: 1},
		)
		if err != nil {
			t.Fatalf("ListTaskDesignationRollups() error = %v", err)
		}
		if got, want := len(rollups), 1; got != want {
			t.Fatalf("len(rollups) = %d, want %d", got, want)
		}
		var summary taskDesignationRollupStatus
		if err := json.Unmarshal(rollups[0].SummaryJSON, &summary); err != nil {
			t.Fatalf("Unmarshal(rollup) error = %v", err)
		}
		if !summary.Complete || summary.Total != 2 || summary.Completed != 1 || summary.Failed != 1 {
			t.Fatalf("rollup summary = %#v, want complete 1/1 split across two runs", summary)
		}
	})

	t.Run("Should construct with configured queue size and timeout", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		observer := newNetworkTaskStatusObserver(
			&fakeNetworkRuntime{},
			db,
			withNetworkTaskStatusObserverLogger(discardLogger()),
			withNetworkTaskStatusObserverClock(func() time.Time { return now }),
			withNetworkTaskStatusObserverQueueSize(7),
			withNetworkTaskStatusObserverTimeout(2*time.Second),
		)
		if observer == nil {
			t.Fatal("newNetworkTaskStatusObserver() = nil, want observer")
		}
		t.Cleanup(observer.shutdown)
		if got, want := cap(observer.queue), 7; got != want {
			t.Fatalf("cap(observer.queue) = %d, want %d", got, want)
		}
		if got, want := observer.timeout, 2*time.Second; got != want {
			t.Fatalf("observer.timeout = %s, want %s", got, want)
		}
	})

	t.Run("Should not include raw run errors in network status messages", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openDaemonTestGlobalDB(t)
		now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		seedNetworkStatusObserverThread(ctx, t, db, now)
		actor := taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "daemon.test"}
		origin := taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "network-status-test"}
		taskRecord := taskpkg.Task{
			ID:          "task-secret-status",
			Scope:       taskpkg.ScopeGlobal,
			Title:       "Investigate secret leak",
			MaxAttempts: 3,
			Status:      taskpkg.TaskStatusInProgress,
			CreatedBy:   actor,
			Origin:      origin,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := db.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := networkStatusObserverRun(
			t,
			"run-secret-status",
			taskRecord.ID,
			taskpkg.TaskRunStatusFailed,
			0,
			"Inspect API latency",
			now,
			origin,
		)
		run.DesignationGroupID = ""
		run.Error = "agh_claim_secret-123 leaked from provider stderr"
		if err := db.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}
		if err := db.PutNetworkTaskThreadOrigin(ctx, store.NetworkTaskThreadOrigin{
			TaskID:           taskRecord.ID,
			WorkspaceID:      "wks_status",
			Channel:          "builders",
			ThreadID:         "thread_status",
			OriginMessageID:  "msg-origin",
			Digest:           "Investigate secret leak",
			SourceMessageIDs: []string{"msg-origin"},
			CreatedAt:        now,
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("PutNetworkTaskThreadOrigin() error = %v", err)
		}

		runtime := &fakeNetworkRuntime{}
		observer := &networkTaskStatusObserver{
			network: runtime,
			tasks:   db,
			prefs:   db,
			logger:  discardLogger(),
			now: func() time.Time {
				return now.Add(time.Minute)
			},
		}
		err := observer.processWithContext(ctx, taskpkg.EventRecord{Event: taskpkg.Event{
			ID:        "evt-run-secret-failed",
			TaskID:    taskRecord.ID,
			RunID:     run.ID,
			EventType: taskEventRunFailed,
			Actor:     actor,
			Origin:    origin,
			Timestamp: now.Add(time.Second),
		}})
		if err != nil {
			t.Fatalf("processWithContext() error = %v", err)
		}

		runtime.mu.Lock()
		calls := append([]network.RuntimeSendRequest(nil), runtime.runtimeSendCalls...)
		runtime.mu.Unlock()
		if got, want := len(calls), 1; got != want {
			t.Fatalf("len(runtime sends) = %d, want %d: %#v", got, want, calls)
		}
		assertNetworkTaskStatusSend(t, calls[0], "task_status", "failed.")
		var body network.SayBody
		if err := json.Unmarshal(calls[0].Body, &body); err != nil {
			t.Fatalf("Unmarshal(RuntimeSendRequest.Body) error = %v", err)
		}
		if strings.Contains(body.Text, "agh_claim_secret-123") ||
			strings.Contains(body.Text, "provider stderr") {
			t.Fatalf("SayBody.Text = %q, want no raw run error", body.Text)
		}
	})
}

func seedNetworkStatusObserverThread(
	ctx context.Context,
	t *testing.T,
	db interface {
		InsertWorkspace(context.Context, workspacepkg.Workspace) error
		WriteNetworkChannel(context.Context, store.NetworkChannelEntry) error
		WriteConversationMessage(
			context.Context,
			store.NetworkConversationMessage,
		) (store.NetworkConversationWriteResult, error)
	},
	now time.Time,
) {
	t.Helper()

	if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
		ID:        "wks_status",
		RootDir:   t.TempDir(),
		Name:      "status-observer",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
		WorkspaceID:  "wks_status",
		Channel:      "builders",
		Purpose:      "Coordination",
		FanoutPolicy: store.NetworkFanoutPolicyCapabilityMatch,
		CreatedBy:    "daemon.test",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}
	_, err := db.WriteConversationMessage(ctx, store.NetworkConversationMessage{
		MessageID:   "msg-origin",
		SessionID:   "sess-origin",
		WorkspaceID: "wks_status",
		Channel:     "builders",
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    "thread_status",
		Direction:   "received",
		PeerFrom:    "reviewer.sess-origin",
		Kind:        store.NetworkKindSay,
		Body:        json.RawMessage(`{"text":"please investigate latency"}`),
		Text:        "please investigate latency",
		SizeBytes:   int64(len(`{"text":"please investigate latency"}`)),
		Timestamp:   now,
	})
	if err != nil {
		t.Fatalf("WriteConversationMessage() error = %v", err)
	}
}

func networkStatusObserverRun(
	t *testing.T,
	id string,
	taskID string,
	status taskpkg.RunStatus,
	index int,
	brief string,
	now time.Time,
	origin taskpkg.Origin,
) taskpkg.Run {
	t.Helper()

	metadata, err := json.Marshal(map[string]any{
		"designation": map[string]any{
			"index": index,
			"brief": brief,
		},
	})
	if err != nil {
		t.Fatalf("Marshal(designation metadata) error = %v", err)
	}
	return taskpkg.Run{
		ID:                 id,
		TaskID:             taskID,
		Status:             status,
		Attempt:            index + 1,
		Origin:             origin,
		DesignationGroupID: "tdg-status",
		Metadata:           metadata,
		QueuedAt:           now,
		StartedAt:          now.Add(time.Second),
		EndedAt:            now.Add(2 * time.Second),
	}
}

func assertNetworkTaskStatusSend(
	t *testing.T,
	call network.RuntimeSendRequest,
	wantIntent string,
	wantText string,
) {
	t.Helper()

	if call.Surface == nil || *call.Surface != network.SurfaceThread {
		t.Fatalf("RuntimeSendRequest.Surface = %#v, want thread", call.Surface)
	}
	if call.ThreadID == nil || *call.ThreadID != "thread_status" {
		t.Fatalf("RuntimeSendRequest.ThreadID = %#v, want thread_status", call.ThreadID)
	}
	if call.ReplyTo == nil || *call.ReplyTo != "msg-origin" {
		t.Fatalf("RuntimeSendRequest.ReplyTo = %#v, want msg-origin", call.ReplyTo)
	}
	var body network.SayBody
	if err := json.Unmarshal(call.Body, &body); err != nil {
		t.Fatalf("Unmarshal(RuntimeSendRequest.Body) error = %v", err)
	}
	if body.Intent != wantIntent {
		t.Fatalf("SayBody.Intent = %q, want %q", body.Intent, wantIntent)
	}
	if !strings.Contains(body.Text, wantText) {
		t.Fatalf("SayBody.Text = %q, want substring %q", body.Text, wantText)
	}
}
