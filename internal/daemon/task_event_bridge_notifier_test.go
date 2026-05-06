package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestBridgeTerminalTaskNotificationObserver(t *testing.T) {
	t.Run("Should deliver terminal task notification and advance cursor", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 5, 6, 1, 0, 0, 0, time.UTC)
		store := newDaemonBridgeNotificationStore(now)
		observer := newBridgeTerminalTaskNotificationObserver(
			store,
			store,
			store,
			store,
			store,
			discardLogger(),
			func() time.Time { return now },
		)
		if observer == nil {
			t.Fatal("newBridgeTerminalTaskNotificationObserver() = nil, want observer")
		}

		observer.OnTaskEvent(context.Background(), store.records[0])

		if got, want := len(store.deliveries), 1; got != want {
			t.Fatalf("len(deliveries) = %d, want %d", got, want)
		}
		delivery := store.deliveries[0]
		if got, want := delivery.Event.BridgeInstanceID, store.bridge.ID; got != want {
			t.Fatalf("delivery bridge id = %q, want %q", got, want)
		}
		if got, want := delivery.Event.Seq, store.records[0].Sequence; got != want {
			t.Fatalf("delivery sequence = %d, want %d", got, want)
		}
		cursor, err := store.GetCursor(context.Background(), store.subscription.CursorKey())
		if err != nil {
			t.Fatalf("GetCursor() error = %v", err)
		}
		if got, want := cursor.LastSequence, store.records[0].Sequence; got != want {
			t.Fatalf("cursor.LastSequence = %d, want %d", got, want)
		}
		if cursor.LastDeliveryID != delivery.Event.DeliveryID {
			t.Fatalf("cursor.LastDeliveryID = %q, want %q", cursor.LastDeliveryID, delivery.Event.DeliveryID)
		}
	})

	t.Run("Should ignore non-terminal task events", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 5, 6, 1, 5, 0, 0, time.UTC)
		store := newDaemonBridgeNotificationStore(now)
		observer := newBridgeTerminalTaskNotificationObserver(
			store,
			store,
			store,
			store,
			store,
			discardLogger(),
			func() time.Time { return now },
		)
		if observer == nil {
			t.Fatal("newBridgeTerminalTaskNotificationObserver() = nil, want observer")
		}

		record := store.records[0]
		record.Event.EventType = "task.run_started"
		observer.OnTaskEvent(context.Background(), record)

		if len(store.deliveries) != 0 {
			t.Fatalf("len(deliveries) = %d, want 0", len(store.deliveries))
		}
		if _, err := store.GetCursor(context.Background(), store.subscription.CursorKey()); !errors.Is(
			err,
			notifications.ErrCursorNotFound,
		) {
			t.Fatalf("GetCursor(non-terminal) error = %v, want ErrCursorNotFound", err)
		}
	})
}

func TestTaskEventObserverFanout(t *testing.T) {
	t.Run("Should notify every task event observer", func(t *testing.T) {
		t.Parallel()

		first := &recordingTaskEventObserver{}
		second := &recordingTaskEventObserver{}
		fanout := newTaskEventObserverFanout(discardLogger(), first, nil, second)
		if fanout == nil {
			t.Fatal("newTaskEventObserverFanout() = nil, want fanout")
		}

		record := taskpkg.EventRecord{Event: taskpkg.Event{ID: "evt-1", TaskID: "task-1"}}
		fanout.OnTaskEvent(context.Background(), record)

		if got, want := first.count, 1; got != want {
			t.Fatalf("first observer count = %d, want %d", got, want)
		}
		if got, want := second.count, 1; got != want {
			t.Fatalf("second observer count = %d, want %d", got, want)
		}
	})
}

type daemonBridgeNotificationStore struct {
	subscription bridgepkg.BridgeTaskSubscription
	task         taskpkg.Task
	run          taskpkg.Run
	bridge       bridgepkg.BridgeInstance
	records      []taskpkg.EventRecord
	cursors      map[string]notifications.Cursor
	deliveries   []bridgepkg.DeliveryRequest
	now          time.Time
}

var _ bridgepkg.BridgeTaskSubscriptionStore = (*daemonBridgeNotificationStore)(nil)
var _ bridgepkg.TerminalTaskEventReader = (*daemonBridgeNotificationStore)(nil)
var _ bridgepkg.BridgeInstanceReader = (*daemonBridgeNotificationStore)(nil)
var _ notifications.CursorStore = (*daemonBridgeNotificationStore)(nil)
var _ bridgepkg.DeliveryTransport = (*daemonBridgeNotificationStore)(nil)

func newDaemonBridgeNotificationStore(now time.Time) *daemonBridgeNotificationStore {
	subscription := bridgepkg.BridgeTaskSubscription{
		SubscriptionID:   "sub-1",
		TaskID:           "task-1",
		BridgeInstanceID: "brg-1",
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      "ws-1",
		PeerID:           "12345",
		ThreadID:         "1",
		DeliveryMode:     bridgepkg.DeliveryModeReply,
		CreatedBy:        taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return &daemonBridgeNotificationStore{
		subscription: subscription,
		task: taskpkg.Task{
			ID:             "task-1",
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-1",
			Status:         taskpkg.TaskStatusCompleted,
			LatestEventSeq: 7,
		},
		run: taskpkg.Run{
			ID:     "run-1",
			TaskID: "task-1",
			Status: taskpkg.TaskRunStatusCompleted,
		},
		bridge: bridgepkg.BridgeInstance{
			ID:            "brg-1",
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-1",
			Platform:      "telegram",
			ExtensionName: "telegram",
			DisplayName:   "Telegram",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		records: []taskpkg.EventRecord{{
			Sequence: 7,
			Event: taskpkg.Event{
				ID:        "evt-7",
				TaskID:    "task-1",
				RunID:     "run-1",
				EventType: "task.run_completed",
				Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindAgentSession, Ref: "sess-1"},
				Timestamp: now,
			},
		}},
		cursors: make(map[string]notifications.Cursor),
		now:     now,
	}
}

func (s *daemonBridgeNotificationStore) PutBridgeTaskSubscription(
	context.Context,
	bridgepkg.BridgeTaskSubscription,
) error {
	return nil
}

func (s *daemonBridgeNotificationStore) GetBridgeTaskSubscription(
	context.Context,
	string,
) (bridgepkg.BridgeTaskSubscription, error) {
	return s.subscription, nil
}

func (s *daemonBridgeNotificationStore) ListBridgeTaskSubscriptions(
	_ context.Context,
	query bridgepkg.BridgeTaskSubscriptionQuery,
) ([]bridgepkg.BridgeTaskSubscription, error) {
	if query.TaskID != "" && query.TaskID != s.subscription.TaskID {
		return nil, nil
	}
	return []bridgepkg.BridgeTaskSubscription{s.subscription}, nil
}

func (s *daemonBridgeNotificationStore) DeleteBridgeTaskSubscription(context.Context, string) error {
	return nil
}

func (s *daemonBridgeNotificationStore) GetTask(context.Context, string) (taskpkg.Task, error) {
	return s.task, nil
}

func (s *daemonBridgeNotificationStore) GetTaskRun(context.Context, string) (taskpkg.Run, error) {
	return s.run, nil
}

func (s *daemonBridgeNotificationStore) ListTaskEventRecords(
	_ context.Context,
	query taskpkg.EventRecordQuery,
) ([]taskpkg.EventRecord, error) {
	records := make([]taskpkg.EventRecord, 0, len(s.records))
	for _, record := range s.records {
		if query.TaskID != "" && record.Event.TaskID != query.TaskID {
			continue
		}
		if record.Sequence <= query.AfterSequence {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *daemonBridgeNotificationStore) GetBridgeInstance(
	context.Context,
	string,
) (bridgepkg.BridgeInstance, error) {
	return s.bridge, nil
}

func (s *daemonBridgeNotificationStore) GetCursor(
	_ context.Context,
	key notifications.CursorKey,
) (notifications.Cursor, error) {
	cursor, ok := s.cursors[daemonCursorStoreKey(key)]
	if !ok {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func (s *daemonBridgeNotificationStore) ListCursors(
	context.Context,
	notifications.CursorQuery,
) ([]notifications.Cursor, error) {
	return nil, nil
}

func (s *daemonBridgeNotificationStore) AdvanceCursor(
	_ context.Context,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	cursor := notifications.Cursor{
		Key:             update.Key,
		LastSequence:    update.LastSequence,
		LastDeliveryID:  update.DeliveryID,
		LastDeliveredAt: update.LastDeliveredAt,
		UpdatedAt:       update.Now,
	}
	s.cursors[daemonCursorStoreKey(update.Key)] = cursor
	return cursor, nil
}

func (s *daemonBridgeNotificationStore) ResetCursor(
	context.Context,
	notifications.ResetCursor,
) (notifications.Cursor, error) {
	return notifications.Cursor{}, nil
}

func (s *daemonBridgeNotificationStore) RecordCursorError(
	_ context.Context,
	report notifications.CursorError,
) (notifications.Cursor, error) {
	cursor := notifications.Cursor{
		Key:       report.Key,
		LastError: report.LastError,
		UpdatedAt: report.Now,
	}
	s.cursors[daemonCursorStoreKey(report.Key)] = cursor
	return cursor, nil
}

func (s *daemonBridgeNotificationStore) DeliverBridge(
	_ context.Context,
	_ string,
	req bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
	s.deliveries = append(s.deliveries, req)
	return bridgepkg.DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
}

type recordingTaskEventObserver struct {
	count int
}

var _ taskpkg.EventObserver = (*recordingTaskEventObserver)(nil)

func (o *recordingTaskEventObserver) OnTaskEvent(context.Context, taskpkg.EventRecord) {
	o.count++
}

func daemonCursorStoreKey(key notifications.CursorKey) string {
	return key.ConsumerID + "\x00" + key.StreamName + "\x00" + key.SubjectID
}
