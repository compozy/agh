package network

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

type fakeNetworkTaskService struct {
	getTaskFn    func(context.Context, string, taskpkg.ActorContext) (*taskpkg.TaskView, error)
	createTaskFn func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	updateTaskFn func(context.Context, string, taskpkg.TaskPatch, taskpkg.ActorContext) (*taskpkg.Task, error)
	cancelTaskFn func(context.Context, string, taskpkg.CancelTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	enqueueRunFn func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.TaskRun, error)
}

func (f fakeNetworkTaskService) GetTask(ctx context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.TaskView, error) {
	if f.getTaskFn == nil {
		return nil, errors.New("unexpected GetTask call")
	}
	return f.getTaskFn(ctx, id, actor)
}

func (f fakeNetworkTaskService) CreateTask(ctx context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
	if f.createTaskFn == nil {
		return nil, errors.New("unexpected CreateTask call")
	}
	return f.createTaskFn(ctx, spec, actor)
}

func (f fakeNetworkTaskService) UpdateTask(ctx context.Context, id string, patch taskpkg.TaskPatch, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
	if f.updateTaskFn == nil {
		return nil, errors.New("unexpected UpdateTask call")
	}
	return f.updateTaskFn(ctx, id, patch, actor)
}

func (f fakeNetworkTaskService) CancelTask(ctx context.Context, id string, req taskpkg.CancelTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
	if f.cancelTaskFn == nil {
		return nil, errors.New("unexpected CancelTask call")
	}
	return f.cancelTaskFn(ctx, id, req, actor)
}

func (f fakeNetworkTaskService) EnqueueRun(ctx context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
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

func TestEnqueueRunFromPeerRejectsChannelMismatchAndAudits(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"
	auditor := &taskIngressAuditRecorder{}
	var getActor taskpkg.ActorContext
	enqueueCalled := false
	manager := &Manager{
		logger:  discardManagerLogger(),
		now:     func() time.Time { return now },
		peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
		auditor: auditor,
		tasks: fakeNetworkTaskService{
			getTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.TaskView, error) {
				getActor = actor
				return &taskpkg.TaskView{
					Task: taskpkg.Task{
						ID:             id,
						Scope:          taskpkg.ScopeGlobal,
						Title:          "Bound task",
						NetworkChannel: "finance",
					},
				}, nil
			},
			enqueueRunFn: func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.TaskRun, error) {
				enqueueCalled = true
				return nil, nil
			},
		},
	}

	_, err := manager.EnqueueRunFromPeer(context.Background(), TaskIngressContext{
		PeerID:    peerID,
		Channel:   "ops",
		RequestID: "req-enqueue-1",
	}, taskpkg.EnqueueRun{
		TaskID:         "task-1",
		IdempotencyKey: "idem-1",
	})
	if !errors.Is(err, ErrTaskChannelMismatch) {
		t.Fatalf("EnqueueRunFromPeer() error = %v, want %v", err, ErrTaskChannelMismatch)
	}
	if enqueueCalled {
		t.Fatal("EnqueueRunFromPeer() called task service enqueue on channel mismatch")
	}
	if got, want := getActor.Actor.Kind, taskpkg.ActorKindNetworkPeer; got != want {
		t.Fatalf("GetTask actor kind = %q, want %q", got, want)
	}
	if got, want := getActor.Actor.Ref, peerID; got != want {
		t.Fatalf("GetTask actor ref = %q, want %q", got, want)
	}
	if got, want := getActor.Origin.Ref, "peer:"+peerID+"/channel:ops"; got != want {
		t.Fatalf("GetTask origin ref = %q, want %q", got, want)
	}

	records := auditor.snapshot()
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(task ingress audit records) = %d, want %d", got, want)
	}
	if got, want := records[0].Action, networkTaskActionEnqueue; got != want {
		t.Fatalf("audit action = %q, want %q", got, want)
	}
	if got, want := records[0].Direction, AuditDirectionRejected; got != want {
		t.Fatalf("audit direction = %q, want %q", got, want)
	}
	if got, want := records[0].Reason, "channel_mismatch"; got != want {
		t.Fatalf("audit reason = %q, want %q", got, want)
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
				if got, want := spec.NetworkChannel, "ops"; got != want {
					t.Fatalf("CreateTask spec.NetworkChannel = %q, want %q", got, want)
				}
				return &taskpkg.Task{
					ID:             "task-1",
					Scope:          taskpkg.ScopeGlobal,
					Title:          spec.Title,
					NetworkChannel: spec.NetworkChannel,
					CreatedBy:      actor.Actor,
					Origin:         actor.Origin,
				}, nil
			},
		},
	}

	record, err := manager.CreateTaskFromPeer(context.Background(), TaskIngressContext{
		PeerID:    peerID,
		Channel:   "ops",
		RequestID: "req-create-1",
	}, taskpkg.CreateTask{
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Peer task",
		NetworkChannel: "ops",
	})
	if err != nil {
		t.Fatalf("CreateTaskFromPeer() error = %v", err)
	}
	if got, want := createActor.Actor.Kind, taskpkg.ActorKindNetworkPeer; got != want {
		t.Fatalf("CreateTask actor kind = %q, want %q", got, want)
	}
	if got, want := createActor.Origin.Ref, "peer:"+peerID+"/channel:ops"; got != want {
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

func TestUpdateTaskFromPeerAllowsOnlyStaleChannelRepair(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 18, 5, 0, 0, time.UTC)
	peerID := "reviewer.sess-ops"

	t.Run("allows clearing stale channel", func(t *testing.T) {
		t.Parallel()

		auditor := &taskIngressAuditRecorder{}
		updateCalled := false
		manager := &Manager{
			logger:  discardManagerLogger(),
			now:     func() time.Time { return now },
			peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
			auditor: auditor,
			tasks: fakeNetworkTaskService{
				getTaskFn: func(_ context.Context, id string, _ taskpkg.ActorContext) (*taskpkg.TaskView, error) {
					return &taskpkg.TaskView{
						Task: taskpkg.Task{
							ID:             id,
							Scope:          taskpkg.ScopeGlobal,
							Title:          "Stale task",
							NetworkChannel: "Finance",
						},
					}, nil
				},
				updateTaskFn: func(_ context.Context, id string, patch taskpkg.TaskPatch, _ taskpkg.ActorContext) (*taskpkg.Task, error) {
					updateCalled = true
					if patch.NetworkChannel == nil || *patch.NetworkChannel != "" {
						t.Fatalf("update patch network_channel = %#v, want explicit clear", patch.NetworkChannel)
					}
					return &taskpkg.Task{
						ID:    id,
						Scope: taskpkg.ScopeGlobal,
						Title: "Stale task",
					}, nil
				},
			},
		}

		clear := ""
		record, err := manager.UpdateTaskFromPeer(context.Background(), TaskIngressContext{
			PeerID:    peerID,
			Channel:   "ops",
			RequestID: "req-update-clear",
		}, "task-1", taskpkg.TaskPatch{NetworkChannel: &clear})
		if err != nil {
			t.Fatalf("UpdateTaskFromPeer(clear stale channel) error = %v", err)
		}
		if !updateCalled {
			t.Fatal("UpdateTaskFromPeer(clear stale channel) did not call task service update")
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

	t.Run("rejects unrelated writes while stale channel remains", func(t *testing.T) {
		t.Parallel()

		auditor := &taskIngressAuditRecorder{}
		updateCalled := false
		manager := &Manager{
			logger:  discardManagerLogger(),
			now:     func() time.Time { return now },
			peers:   newRemotePeerRegistry(t, now, "ops", peerID, []string{networkTaskWriteCapability}),
			auditor: auditor,
			tasks: fakeNetworkTaskService{
				getTaskFn: func(_ context.Context, id string, _ taskpkg.ActorContext) (*taskpkg.TaskView, error) {
					return &taskpkg.TaskView{
						Task: taskpkg.Task{
							ID:             id,
							Scope:          taskpkg.ScopeGlobal,
							Title:          "Stale task",
							NetworkChannel: "Finance",
						},
					}, nil
				},
				updateTaskFn: func(context.Context, string, taskpkg.TaskPatch, taskpkg.ActorContext) (*taskpkg.Task, error) {
					updateCalled = true
					return nil, nil
				},
			},
		}

		title := "Renamed"
		_, err := manager.UpdateTaskFromPeer(context.Background(), TaskIngressContext{
			PeerID:    peerID,
			Channel:   "ops",
			RequestID: "req-update-title",
		}, "task-1", taskpkg.TaskPatch{Title: &title})
		if !errors.Is(err, ErrTaskChannelStale) {
			t.Fatalf("UpdateTaskFromPeer(unrelated stale update) error = %v, want %v", err, ErrTaskChannelStale)
		}
		if updateCalled {
			t.Fatal("UpdateTaskFromPeer(unrelated stale update) called task service update")
		}

		records := auditor.snapshot()
		if got, want := len(records), 1; got != want {
			t.Fatalf("len(task ingress audit records) = %d, want %d", got, want)
		}
		if got, want := records[0].Direction, AuditDirectionRejected; got != want {
			t.Fatalf("audit direction = %q, want %q", got, want)
		}
		if got, want := records[0].Reason, "stale_channel"; got != want {
			t.Fatalf("audit reason = %q, want %q", got, want)
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
		PeerID:    peerID,
		Channel:   "ops",
		RequestID: "req-cancel-1",
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

	t.Run("validates ingress context fields", func(t *testing.T) {
		t.Parallel()

		if err := (TaskIngressContext{}).Validate(); err == nil {
			t.Fatal("TaskIngressContext{}.Validate() error = nil, want non-nil")
		}
		if err := (TaskIngressContext{PeerID: "bad peer", Channel: "ops", RequestID: "req-1"}).Validate(); err == nil {
			t.Fatal("TaskIngressContext(invalid peer).Validate() error = nil, want non-nil")
		}
	})

	t.Run("covers reason mapping", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			err  error
			want string
		}{
			{err: ErrTaskChannelMismatch, want: "channel_mismatch"},
			{err: ErrTaskChannelStale, want: "stale_channel"},
			{err: ErrTaskIngressCapabilityDenied, want: "capability_denied"},
			{err: ErrTaskIngressPeerNotFound, want: "peer_not_found"},
			{err: ErrTaskIngressUnavailable, want: "task_ingress_unavailable"},
			{err: taskpkg.ErrTaskNotFound, want: "task_not_found"},
			{err: taskpkg.ErrValidation, want: "validation_failed"},
			{err: taskpkg.ErrPermissionDenied, want: "permission_denied"},
			{err: taskpkg.ErrStaleNetworkChannel, want: "stale_channel"},
			{err: ErrMissingField, want: "invalid_request"},
			{err: errors.New("boom"), want: "task_ingress_failed"},
		}

		for _, tc := range testCases {
			if got := taskIngressReason(tc.err); got != tc.want {
				t.Fatalf("taskIngressReason(%v) = %q, want %q", tc.err, got, tc.want)
			}
		}
	})

	t.Run("applies manager task service option", func(t *testing.T) {
		t.Parallel()

		opts := managerOptions{}
		service := fakeNetworkTaskService{}
		WithManagerTaskService(service)(&opts)
		if opts.tasks == nil {
			t.Fatal("WithManagerTaskService() did not assign opts.tasks")
		}
	})
}

func newRemotePeerRegistry(t *testing.T, now time.Time, channel string, peerID string, capabilities []string) *PeerRegistry {
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
	if _, stored, err := registry.RefreshRemote(channel, card, now); err != nil {
		t.Fatalf("RefreshRemote(%q, %q) error = %v", channel, peerID, err)
	} else if !stored {
		t.Fatalf("RefreshRemote(%q, %q) stored = false, want true", channel, peerID)
	}

	return registry
}
