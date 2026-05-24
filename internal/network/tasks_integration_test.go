//go:build integration

package network

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

func TestNetworkTaskIngressCreateAndEnqueueRun(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 14, 18, 30, 0, 0, time.UTC)
	db := openNetworkTaskIngressDB(t)
	taskManager := newNetworkTaskIntegrationManager(t, db, taskpkg.WithNetworkChannelValidator(ValidateChannel))
	manager := newNetworkTaskIngressManager(t, ctx, now, db, taskManager)

	peerID := "reviewer.sess-ops"
	registerRemoteTaskPeer(t, manager, now, "ops", peerID, []string{networkTaskWriteCapability, "task.read"})

	created, err := manager.CreateTaskFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-create-1",
	}, taskpkg.CreateTask{
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Peer-created task",
		NetworkChannel: "ops",
	})
	if err != nil {
		t.Fatalf("CreateTaskFromPeer() error = %v", err)
	}
	if got, want := created.CreatedBy.Kind, taskpkg.ActorKindNetworkPeer; got != want {
		t.Fatalf("created.CreatedBy.Kind = %q, want %q", got, want)
	}
	if got, want := created.CreatedBy.Ref, peerID; got != want {
		t.Fatalf("created.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := created.Origin.Ref, "workspace:wks_test/channel:ops/peer:"+peerID; got != want {
		t.Fatalf("created.Origin.Ref = %q, want %q", got, want)
	}

	run, err := manager.EnqueueRunFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-enqueue-1",
		Surface:     SurfaceThread,
		ThreadID:    "thread_task_ingress",
		WorkID:      "work_task_ingress",
		TraceID:     "trace-task-ingress",
	}, taskpkg.EnqueueRun{
		TaskID:         created.ID,
		IdempotencyKey: "idem-peer-enqueue-1",
		NetworkChannel: "ops",
		Metadata:       json.RawMessage(`{"client":"kept"}`),
	})
	if err != nil {
		t.Fatalf("EnqueueRunFromPeer() error = %v", err)
	}
	if got, want := run.TaskID, created.ID; got != want {
		t.Fatalf("run.TaskID = %q, want %q", got, want)
	}
	if got, want := run.NetworkChannel, "ops"; got != want {
		t.Fatalf("run.NetworkChannel = %q, want %q", got, want)
	}
	if got, want := run.Origin.Ref, "workspace:wks_test/channel:ops/peer:"+peerID; got != want {
		t.Fatalf("run.Origin.Ref = %q, want %q", got, want)
	}

	storedTask, err := db.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := storedTask.NetworkChannel, "ops"; got != want {
		t.Fatalf("storedTask.NetworkChannel = %q, want %q", got, want)
	}

	storedRun, err := db.GetTaskRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := storedRun.IdempotencyKey, "idem-peer-enqueue-1"; got != want {
		t.Fatalf("storedRun.IdempotencyKey = %q, want %q", got, want)
	}
	var metadata map[string]string
	if err := json.Unmarshal(storedRun.Metadata, &metadata); err != nil {
		t.Fatalf("json.Unmarshal(storedRun.Metadata) error = %v", err)
	}
	for key, want := range map[string]string{
		"client":             "kept",
		"network_work_id":    "work_task_ingress",
		"network_message_id": "req-enqueue-1",
		"network_channel":    "ops",
		"network_surface":    string(SurfaceThread),
		"network_thread_id":  "thread_task_ingress",
		"network_trace_id":   "trace-task-ingress",
	} {
		if got := metadata[key]; got != want {
			t.Fatalf("storedRun.Metadata[%q] = %q, want %q in %s", key, got, want, storedRun.Metadata)
		}
	}
	claimActor, err := taskpkg.DeriveHumanActorContext("operator", taskpkg.OriginKindCLI, "network task ingress test")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	claimed, err := taskManager.ClaimRun(ctx, run.ID, taskpkg.ClaimRun{}, claimActor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	if got, want := claimed.Status, taskpkg.TaskRunStatusClaimed; got != want {
		t.Fatalf("claimed.Status = %q, want %q", got, want)
	}

	createAudit := findNetworkAuditByMessageID(t, db, "req-create-1")
	if got, want := createAudit.Direction, AuditDirectionReceived; got != want {
		t.Fatalf("create audit direction = %q, want %q", got, want)
	}
	if got, want := createAudit.Kind, networkTaskActionCreate; got != want {
		t.Fatalf("create audit kind = %q, want %q", got, want)
	}

	enqueueAudit := findNetworkAuditByMessageID(t, db, "req-enqueue-1")
	if got, want := enqueueAudit.Direction, AuditDirectionReceived; got != want {
		t.Fatalf("enqueue audit direction = %q, want %q", got, want)
	}
	if got, want := enqueueAudit.Kind, networkTaskActionEnqueue; got != want {
		t.Fatalf("enqueue audit kind = %q, want %q", got, want)
	}
}

func TestNetworkTaskIngressMismatchRecordsAuditWithoutMutation(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 14, 18, 35, 0, 0, time.UTC)
	db := openNetworkTaskIngressDB(t)
	taskManager := newNetworkTaskIntegrationManager(t, db, taskpkg.WithNetworkChannelValidator(ValidateChannel))
	manager := newNetworkTaskIngressManager(t, ctx, now, db, taskManager)

	peerID := "reviewer.sess-ops"
	registerRemoteTaskPeer(t, manager, now, "ops", peerID, []string{networkTaskWriteCapability})

	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task create")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	taskRecord, err := taskManager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Finance task",
		NetworkChannel: "finance",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	originalUpdatedAt := taskRecord.UpdatedAt

	_, err = manager.EnqueueRunFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-enqueue-mismatch",
		Surface:     SurfaceThread,
		ThreadID:    "thread_task_mismatch",
		WorkID:      "work_task_mismatch",
	}, taskpkg.EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "idem-mismatch",
	})
	if !errors.Is(err, ErrTaskChannelMismatch) {
		t.Fatalf("EnqueueRunFromPeer() error = %v, want %v", err, ErrTaskChannelMismatch)
	}

	storedTask, err := db.GetTask(ctx, taskRecord.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := storedTask.NetworkChannel, "finance"; got != want {
		t.Fatalf("storedTask.NetworkChannel = %q, want %q", got, want)
	}
	if !storedTask.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("storedTask.UpdatedAt = %s, want unchanged %s", storedTask.UpdatedAt, originalUpdatedAt)
	}

	runs, err := db.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if got := len(runs); got != 0 {
		t.Fatalf("len(runs) = %d, want 0", got)
	}

	audit := findNetworkAuditByMessageID(t, db, "req-enqueue-mismatch")
	if got, want := audit.Direction, AuditDirectionRejected; got != want {
		t.Fatalf("audit.Direction = %q, want %q", got, want)
	}
	if got, want := audit.Kind, networkTaskActionEnqueue; got != want {
		t.Fatalf("audit.Kind = %q, want %q", got, want)
	}
	if got, want := audit.Reason, "channel_mismatch"; got != want {
		t.Fatalf("audit.Reason = %q, want %q", got, want)
	}
}

func TestNetworkTaskIngressDuplicateEnqueueUsesCanonicalRun(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 14, 18, 40, 0, 0, time.UTC)
	db := openNetworkTaskIngressDB(t)
	taskManager := newNetworkTaskIntegrationManager(t, db, taskpkg.WithNetworkChannelValidator(ValidateChannel))
	manager := newNetworkTaskIngressManager(t, ctx, now, db, taskManager)

	peerID := "reviewer.sess-ops"
	registerRemoteTaskPeer(t, manager, now, "ops", peerID, []string{networkTaskWriteCapability})

	taskRecord, err := manager.CreateTaskFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-create-dup",
	}, taskpkg.CreateTask{
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Idempotent peer task",
		NetworkChannel: "ops",
	})
	if err != nil {
		t.Fatalf("CreateTaskFromPeer() error = %v", err)
	}

	firstRun, err := manager.EnqueueRunFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-enqueue-dup-1",
		Surface:     SurfaceThread,
		ThreadID:    "thread_task_dup",
		WorkID:      "work_task_dup",
	}, taskpkg.EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "idem-dup-1",
	})
	if err != nil {
		t.Fatalf("EnqueueRunFromPeer(first) error = %v", err)
	}
	secondRun, err := manager.EnqueueRunFromPeer(ctx, TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-enqueue-dup-2",
		Surface:     SurfaceThread,
		ThreadID:    "thread_task_dup",
		WorkID:      "work_task_dup",
	}, taskpkg.EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: "idem-dup-1",
	})
	if err != nil {
		t.Fatalf("EnqueueRunFromPeer(duplicate) error = %v", err)
	}
	if got, want := secondRun.ID, firstRun.ID; got != want {
		t.Fatalf("duplicate run id = %q, want %q", got, want)
	}

	runs, err := db.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
	if err != nil {
		t.Fatalf("ListTaskRuns() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("len(runs) = %d, want %d", got, want)
	}
}

func openNetworkTaskIngressDB(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := globaldb.OpenGlobalDB(ctx, filepath.Join(t.TempDir(), "agh.db"))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(context.Background()); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	})
	return db
}

func newNetworkTaskIntegrationManager(t *testing.T, store taskpkg.Store, extraOpts ...taskpkg.Option) *taskpkg.Service {
	t.Helper()

	options := []taskpkg.Option{taskpkg.WithStore(store)}
	options = append(options, extraOpts...)
	manager, err := taskpkg.NewManager(options...)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}
	return manager
}

func newNetworkTaskIngressManager(
	t *testing.T,
	ctx context.Context,
	now time.Time,
	auditStore AuditStore,
	tasks TaskService,
) *Manager {
	t.Helper()

	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		newFakeDeliveryPrompter(),
		filepath.Join(t.TempDir(), "network.audit"),
		auditStore,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return now }),
		WithManagerTaskService(tasks),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	return manager
}

func registerRemoteTaskPeer(
	t *testing.T,
	manager *Manager,
	now time.Time,
	channel string,
	peerID string,
	capabilities []string,
) {
	t.Helper()

	card, err := DefaultPeerCard(peerID)
	if err != nil {
		t.Fatalf("DefaultPeerCard(%q) error = %v", peerID, err)
	}
	card.Capabilities = append([]string(nil), capabilities...)
	if _, stored, err := manager.peers.RefreshRemote(testWorkspaceID, channel, card, now); err != nil {
		t.Fatalf("RefreshRemote(%q, %q) error = %v", channel, peerID, err)
	} else if !stored {
		t.Fatalf("RefreshRemote(%q, %q) stored = false, want true", channel, peerID)
	}
}

func findNetworkAuditByMessageID(t *testing.T, db *globaldb.GlobalDB, messageID string) store.NetworkAuditEntry {
	t.Helper()

	entries, err := db.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		WorkspaceID: testWorkspaceID,
		MessageID:   messageID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit(%q) error = %v", messageID, err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListNetworkAudit(%q) returned %d entries, want exactly 1", messageID, len(entries))
	}
	return entries[0]
}
