package network

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	taskpkg "github.com/compozy/agh/internal/task"
)

type fakeNetworkTaskService struct {
	createTaskFn func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	updateTaskFn func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error)
	cancelTaskFn func(context.Context, string, taskpkg.CancelTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	enqueueRunFn func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.Run, error)
}

func (f fakeNetworkTaskService) CreateTask(
	ctx context.Context,
	spec taskpkg.CreateTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if f.createTaskFn == nil {
		return nil, errors.New("unexpected CreateTask call")
	}
	return f.createTaskFn(ctx, spec, actor)
}

func (f fakeNetworkTaskService) UpdateTask(
	ctx context.Context,
	id string,
	patch taskpkg.Patch,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if f.updateTaskFn == nil {
		return nil, errors.New("unexpected UpdateTask call")
	}
	return f.updateTaskFn(ctx, id, patch, actor)
}

func (f fakeNetworkTaskService) CancelTask(
	ctx context.Context,
	id string,
	req taskpkg.CancelTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if f.cancelTaskFn == nil {
		return nil, errors.New("unexpected CancelTask call")
	}
	return f.cancelTaskFn(ctx, id, req, actor)
}

func (f fakeNetworkTaskService) EnqueueRun(
	ctx context.Context,
	spec taskpkg.EnqueueRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if f.enqueueRunFn == nil {
		return nil, errors.New("unexpected EnqueueRun call")
	}
	return f.enqueueRunFn(ctx, spec, actor)
}

type taskIngressAuditRecorder struct {
	mu      sync.Mutex
	records []TaskIngressAudit
}

var _ AuditWriter = (*taskIngressAuditRecorder)(nil)
var _ TaskIngressAuditWriter = (*taskIngressAuditRecorder)(nil)

func (r *taskIngressAuditRecorder) RecordSent(context.Context, string, Envelope) error {
	return nil
}

func (r *taskIngressAuditRecorder) RecordReceived(context.Context, string, Envelope) error {
	return nil
}

func (r *taskIngressAuditRecorder) RecordRejected(context.Context, string, Envelope, string) error {
	return nil
}

func (r *taskIngressAuditRecorder) RecordDelivered(context.Context, string, Envelope) error {
	return nil
}

func (r *taskIngressAuditRecorder) RecordTaskIngress(_ context.Context, audit TaskIngressAudit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, audit)
	return nil
}

func (r *taskIngressAuditRecorder) snapshot() []TaskIngressAudit {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]TaskIngressAudit(nil), r.records...)
}

func TestEnqueueRunFromPeerAttachesNetworkWorkMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should attach server-derived network metadata to task runs", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 14, 18, 1, 0, 0, time.UTC)
		peerID := "reviewer.sess-ops"
		auditor := &taskIngressAuditRecorder{}
		var captured taskpkg.EnqueueRun
		manager := &Manager{
			logger:  discardManagerLogger(),
			now:     func() time.Time { return now },
			peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
			auditor: auditor,
			tasks: fakeNetworkTaskService{
				enqueueRunFn: func(_ context.Context, spec taskpkg.EnqueueRun, _ taskpkg.ActorContext) (*taskpkg.Run, error) {
					captured = spec
					return &taskpkg.Run{
						ID:             "run-1",
						TaskID:         spec.TaskID,
						IdempotencyKey: spec.IdempotencyKey,
						Metadata:       spec.Metadata,
					}, nil
				},
			},
		}

		run, err := manager.EnqueueRunFromPeer(context.Background(), TaskIngressContext{
			WorkspaceID: testWorkspaceID,
			PeerID:      peerID,
			Channel:     "ops",
			RequestID:   "msg-enqueue-task",
			Surface:     SurfaceThread,
			ThreadID:    "thread_task_ingress",
			WorkID:      "work_task_ingress",
			ReplyTo:     "msg-root-task",
			TraceID:     "trace-task-ingress",
			CausationID: "msg-root-task",
		}, taskpkg.EnqueueRun{
			TaskID:         "task-1",
			IdempotencyKey: "idem-1",
			Metadata:       json.RawMessage(`{"user":"kept"}`),
		})
		if err != nil {
			t.Fatalf("EnqueueRunFromPeer() error = %v", err)
		}
		if got, want := run.Metadata, captured.Metadata; string(got) != string(want) {
			t.Fatalf("run.Metadata = %s, want captured metadata %s", got, want)
		}
		if captured.NetworkParticipation != nil {
			t.Fatalf(
				"captured.NetworkParticipation = %#v, want nil ingress participation",
				captured.NetworkParticipation,
			)
		}

		var metadata map[string]string
		if err := json.Unmarshal(captured.Metadata, &metadata); err != nil {
			t.Fatalf("json.Unmarshal(captured.Metadata) error = %v", err)
		}
		for key, want := range map[string]string{
			"user":                  "kept",
			"network_work_id":       "work_task_ingress",
			"network_message_id":    "msg-enqueue-task",
			"participation_channel": "ops",
			"network_surface":       string(SurfaceThread),
			"network_thread_id":     "thread_task_ingress",
			"network_reply_to":      "msg-root-task",
			"network_trace_id":      "trace-task-ingress",
			"network_causation_id":  "msg-root-task",
		} {
			if got := metadata[key]; got != want {
				t.Fatalf("metadata[%q] = %q, want %q in %s", key, got, want, captured.Metadata)
			}
		}
	})
}

func TestEnqueueRunFromPeerNormalizesIngressIdentifiersBeforePeerLookup(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 2, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"
	var capturedActor taskpkg.ActorContext
	manager := &Manager{
		logger: discardManagerLogger(),
		now:    func() time.Time { return now },
		peers:  newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
		tasks: fakeNetworkTaskService{
			enqueueRunFn: func(_ context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
				capturedActor = actor
				return &taskpkg.Run{ID: "run-1", TaskID: spec.TaskID}, nil
			},
		},
	}

	run, err := manager.EnqueueRunFromPeer(context.Background(), TaskIngressContext{
		WorkspaceID: " " + testWorkspaceID + " ",
		PeerID:      " " + peerID + " ",
		Channel:     " ops ",
		RequestID:   "req-normalized-1",
		Surface:     SurfaceThread,
		ThreadID:    "thread_task_ingress",
		WorkID:      "work_task_ingress",
	}, taskpkg.EnqueueRun{
		TaskID:         "task-1",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("EnqueueRunFromPeer() error = %v", err)
	}
	if run == nil || run.ID != "run-1" {
		t.Fatalf("EnqueueRunFromPeer() run = %#v, want run-1", run)
	}
	if got, want := capturedActor.Actor.Ref, peerID; got != want {
		t.Fatalf("captured actor ref = %q, want %q", got, want)
	}
	if got, want := capturedActor.Origin.Ref, "workspace:wks_test/channel:ops/peer:"+peerID; got != want {
		t.Fatalf("captured actor origin = %q, want %q", got, want)
	}
}

func TestCreateTaskFromPeerUsesServerDerivedIdentityAndAcceptedAudit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 2, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"
	auditor := &taskIngressAuditRecorder{}
	var createActor taskpkg.ActorContext
	manager := &Manager{
		logger:  discardManagerLogger(),
		now:     func() time.Time { return now },
		peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
		auditor: auditor,
		tasks: fakeNetworkTaskService{
			createTaskFn: func(_ context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
				createActor = actor
				return &taskpkg.Task{
					ID:        "task-1",
					Scope:     taskpkg.ScopeGlobal,
					Title:     spec.Title,
					CreatedBy: actor.Actor,
					Origin:    actor.Origin,
				}, nil
			},
		},
	}

	record, err := manager.CreateTaskFromPeer(context.Background(), TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-create-1",
	}, taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Peer task",
	})
	if err != nil {
		t.Fatalf("CreateTaskFromPeer() error = %v", err)
	}
	if got, want := createActor.Actor.Kind, taskpkg.ActorKindNetworkPeer; got != want {
		t.Fatalf("CreateTask actor kind = %q, want %q", got, want)
	}
	if got, want := createActor.Origin.Ref, "workspace:wks_test/channel:ops/peer:"+peerID; got != want {
		t.Fatalf("CreateTask origin ref = %q, want %q", got, want)
	}
	if got, want := record.CreatedBy.Ref, peerID; got != want {
		t.Fatalf("record.CreatedBy.Ref = %q, want %q", got, want)
	}

	records := auditor.snapshot()
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(task ingress audit records) = %d, want %d", got, want)
	}
	if got, want := records[0].Direction, AuditDirectionReceived; got != want {
		t.Fatalf("audit direction = %q, want %q", got, want)
	}
	if got, want := records[0].Action, networkTaskActionCreate; got != want {
		t.Fatalf("audit action = %q, want %q", got, want)
	}
}

func TestUpdateTaskFromPeerDoesNotBindTaskToIngressChannel(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 5, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"

	t.Run("Should forward mutable updates without reading task identity channel state", func(t *testing.T) {
		t.Parallel()

		auditor := &taskIngressAuditRecorder{}
		updateCalled := false
		manager := &Manager{
			logger:  discardManagerLogger(),
			now:     func() time.Time { return now },
			peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
			auditor: auditor,
			tasks: fakeNetworkTaskService{
				updateTaskFn: func(_ context.Context, id string, patch taskpkg.Patch, _ taskpkg.ActorContext) (*taskpkg.Task, error) {
					updateCalled = true
					if patch.Title == nil || *patch.Title != "Renamed" {
						t.Fatalf("update patch title = %#v, want Renamed", patch.Title)
					}
					return &taskpkg.Task{
						ID:    id,
						Scope: taskpkg.ScopeGlobal,
						Title: "Renamed",
					}, nil
				},
			},
		}

		title := "Renamed"
		record, err := manager.UpdateTaskFromPeer(context.Background(), TaskIngressContext{
			WorkspaceID: testWorkspaceID,
			PeerID:      peerID,
			Channel:     "ops",
			RequestID:   "req-update-title",
		}, "task-1", taskpkg.Patch{Title: &title})
		if err != nil {
			t.Fatalf("UpdateTaskFromPeer() error = %v", err)
		}
		if !updateCalled {
			t.Fatal("UpdateTaskFromPeer() did not call task service update")
		}
		if got, want := record.ID, "task-1"; got != want {
			t.Fatalf("updated record id = %q, want %q", got, want)
		}

		records := auditor.snapshot()
		if got, want := len(records), 1; got != want {
			t.Fatalf("len(task ingress audit records) = %d, want %d", got, want)
		}
		if got, want := records[0].Direction, AuditDirectionReceived; got != want {
			t.Fatalf("audit direction = %q, want %q", got, want)
		}
	})
}

func TestCancelTaskFromPeerRejectsPeerWithoutTaskWriteCapability(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 6, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"
	auditor := &taskIngressAuditRecorder{}
	cancelCalled := false
	manager := &Manager{
		logger:  discardManagerLogger(),
		now:     func() time.Time { return now },
		peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{"task.read"}),
		auditor: auditor,
		tasks: fakeNetworkTaskService{
			cancelTaskFn: func(context.Context, string, taskpkg.CancelTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				cancelCalled = true
				return nil, nil
			},
		},
	}

	_, err := manager.CancelTaskFromPeer(context.Background(), TaskIngressContext{
		WorkspaceID: testWorkspaceID,
		PeerID:      peerID,
		Channel:     "ops",
		RequestID:   "req-cancel-1",
	}, "task-1", taskpkg.CancelTask{})
	if !errors.Is(err, ErrTaskIngressCapabilityDenied) {
		t.Fatalf("CancelTaskFromPeer() error = %v, want %v", err, ErrTaskIngressCapabilityDenied)
	}
	if cancelCalled {
		t.Fatal("CancelTaskFromPeer() called task service cancel without task.write capability")
	}

	records := auditor.snapshot()
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(task ingress audit records) = %d, want %d", got, want)
	}
	if got, want := records[0].Reason, "capability_denied"; got != want {
		t.Fatalf("audit reason = %q, want %q", got, want)
	}
}

func TestTaskIngressHelpersCoverValidationAndReasonMapping(t *testing.T) {
	t.Parallel()

	t.Run("Should validates ingress context fields", func(t *testing.T) {
		t.Parallel()

		if err := (TaskIngressContext{}).Validate(); err == nil {
			t.Fatal("TaskIngressContext{}.Validate() error = nil, want non-nil")
		}
		if err := (TaskIngressContext{PeerID: "bad peer", Channel: "ops", RequestID: "req-1"}).Validate(); err == nil {
			t.Fatal("TaskIngressContext(invalid peer).Validate() error = nil, want non-nil")
		}
	})

	t.Run("Should covers reason mapping", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			err  error
			want string
		}{
			{err: ErrTaskIngressCapabilityDenied, want: "capability_denied"},
			{err: ErrTaskIngressPeerNotFound, want: "peer_not_found"},
			{err: ErrTaskIngressUnavailable, want: "task_ingress_unavailable"},
			{err: taskpkg.ErrTaskNotFound, want: "task_not_found"},
			{err: taskpkg.ErrValidation, want: "validation_failed"},
			{err: taskpkg.ErrPermissionDenied, want: "permission_denied"},
			{err: ErrMissingField, want: "invalid_request"},
			{err: errors.New("boom"), want: "task_ingress_failed"},
		}

		for _, tc := range testCases {
			if got := taskIngressReason(tc.err); got != tc.want {
				t.Fatalf("taskIngressReason(%v) = %q, want %q", tc.err, got, tc.want)
			}
		}
	})

	t.Run("Should applies manager task service option", func(t *testing.T) {
		t.Parallel()

		opts := managerOptions{}
		service := fakeNetworkTaskService{}
		WithManagerTaskService(service)(&opts)
		if opts.tasks == nil {
			t.Fatal("WithManagerTaskService() did not assign opts.tasks")
		}
	})
}

func newRemotePeerRegistry(
	t *testing.T,
	now time.Time,
	channel string,
	peerID string,
	capabilities []string,
) *PeerRegistry {
	t.Helper()

	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	card, err := DefaultPeerCard(peerID)
	if err != nil {
		t.Fatalf("DefaultPeerCard(%q) error = %v", peerID, err)
	}
	card.Capabilities = append([]string(nil), capabilities...)
	if _, stored, err := registry.RefreshRemote(testWorkspaceID, channel, card, now); err != nil {
		t.Fatalf("RefreshRemote(%q, %q) error = %v", channel, peerID, err)
	} else if !stored {
		t.Fatalf("RefreshRemote(%q, %q) stored = false, want true", channel, peerID)
	}

	return registry
}
