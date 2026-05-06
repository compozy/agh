package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestBridgeTaskSubscriptionValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should derive durable cursor and delivery target identities", func(t *testing.T) {
		t.Parallel()

		subscription := bridgeTaskSubscriptionForNotifierTest()

		if err := subscription.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got, want := subscription.CursorKey().ConsumerID, "bridge_task_subscription:sub-1"; got != want {
			t.Fatalf("CursorKey().ConsumerID = %q, want %q", got, want)
		}
		if got, want := subscription.CursorKey().StreamName, "task_events"; got != want {
			t.Fatalf("CursorKey().StreamName = %q, want %q", got, want)
		}
		if got, want := subscription.CursorKey().SubjectID, "task-1"; got != want {
			t.Fatalf("CursorKey().SubjectID = %q, want %q", got, want)
		}
		if got, want := subscription.DeliveryTarget().PeerID, "peer-1"; got != want {
			t.Fatalf("DeliveryTarget().PeerID = %q, want %q", got, want)
		}
	})
}

func TestTerminalTaskNotifierDeliverDue(t *testing.T) {
	t.Parallel()

	t.Run("Should deliver an accepted final task event and advance the cursor", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusCompleted,
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				7,
				"evt-7",
				"task-1",
				"run-1",
				terminalTaskEventRunCompleted,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue() error = %v", err)
		}
		if sweep.Delivered != 1 || sweep.Failed != 0 || sweep.Deferred != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want one delivered", sweep)
		}

		calls := transport.snapshotCalls()
		if len(calls) != 1 {
			t.Fatalf("delivery calls = %d, want 1", len(calls))
		}
		if got, want := calls[0].extensionName, "telegram-extension"; got != want {
			t.Fatalf("extensionName = %q, want %q", got, want)
		}
		event := calls[0].request.Event
		if got, want := event.DeliveryID, "notif:sub-1:7"; got != want {
			t.Fatalf("DeliveryID = %q, want %q", got, want)
		}
		if got, want := event.EventType, DeliveryEventTypeFinal; got != want {
			t.Fatalf("delivery event type = %q, want %q", got, want)
		}

		var envelope TerminalTaskNotification
		if err := json.Unmarshal(event.ProviderMetadata, &envelope); err != nil {
			t.Fatalf("Unmarshal(provider metadata) error = %v", err)
		}
		if got, want := envelope.EventType, terminalTaskEventRunCompleted; got != want {
			t.Fatalf("envelope.EventType = %q, want %q", got, want)
		}
		if got, want := envelope.Status, taskpkg.TaskStatusCompleted; got != want {
			t.Fatalf("envelope.Status = %q, want %q", got, want)
		}

		cursor, err := cursorStore.GetCursor(ctx, subscription.CursorKey())
		if err != nil {
			t.Fatalf("GetCursor() error = %v", err)
		}
		if cursor.LastSequence != 7 || cursor.LastDeliveryID != "notif:sub-1:7" {
			t.Fatalf("cursor = %#v, want delivered sequence 7", cursor)
		}
		if cursor.LastError != "" {
			t.Fatalf("cursor.LastError = %q, want no diagnostic after delivery", cursor.LastError)
		}
	})

	t.Run("Should not advance cursor when bridge delivery fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{
			handler: func(context.Context, string, DeliveryRequest) (DeliveryAck, error) {
				return DeliveryAck{}, errors.New("bridge adapter rejected send")
			},
		}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusCompleted,
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				7,
				"evt-7",
				"task-1",
				"run-1",
				terminalTaskEventRunCompleted,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err == nil {
			t.Fatal("DeliverDue() error = nil, want delivery failure")
		}
		if sweep.Failed != 1 || sweep.Delivered != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want one failed delivery", sweep)
		}

		cursor, cursorErr := cursorStore.GetCursor(ctx, subscription.CursorKey())
		if cursorErr != nil {
			t.Fatalf("GetCursor(after failure) error = %v", cursorErr)
		}
		if cursor.LastSequence != 0 {
			t.Fatalf("cursor.LastSequence = %d, want 0 after failed delivery", cursor.LastSequence)
		}
		if !strings.Contains(cursor.LastError, "bridge adapter rejected send") {
			t.Fatalf("cursor.LastError = %q, want delivery failure diagnostic", cursor.LastError)
		}
	})

	t.Run("Should defer non-terminal task state without recording mismatch diagnostics", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusInProgress,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusCompleted,
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				7,
				"evt-7",
				"task-1",
				"run-1",
				terminalTaskEventRunCompleted,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue() error = %v", err)
		}
		if sweep.Deferred != 1 || sweep.Delivered != 0 || sweep.Failed != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want one deferred notification", sweep)
		}
		if calls := transport.snapshotCalls(); len(calls) != 0 {
			t.Fatalf("delivery calls = %d, want 0 for deferred notification", len(calls))
		}
		if _, cursorErr := cursorStore.GetCursor(ctx, subscription.CursorKey()); !errors.Is(
			cursorErr,
			notifications.ErrCursorNotFound,
		) {
			t.Fatalf("GetCursor(after defer) error = %v, want ErrCursorNotFound", cursorErr)
		}
	})

	t.Run("Should fail closed and record a diagnostic for accepted-final status mismatch", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusFailed,
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				4,
				"evt-4",
				"task-1",
				"run-1",
				terminalTaskEventRunFailed,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if !errors.Is(err, ErrTerminalTaskNotificationMismatch) {
			t.Fatalf("DeliverDue() error = %v, want ErrTerminalTaskNotificationMismatch", err)
		}
		if sweep.Failed != 1 || sweep.Delivered != 0 || sweep.Deferred != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want one fail-closed mismatch", sweep)
		}
		if calls := transport.snapshotCalls(); len(calls) != 0 {
			t.Fatalf("delivery calls = %d, want 0 for mismatch", len(calls))
		}

		cursor, cursorErr := cursorStore.GetCursor(ctx, subscription.CursorKey())
		if cursorErr != nil {
			t.Fatalf("GetCursor(after mismatch) error = %v", cursorErr)
		}
		if cursor.LastSequence != 4 || cursor.LastDeliveryID != "" {
			t.Fatalf("cursor = %#v, want mismatch progress without a delivery id", cursor)
		}
		if !strings.Contains(cursor.LastError, "terminal task notification state mismatch") ||
			!strings.Contains(cursor.LastError, "current task status is \"completed\"") {
			t.Fatalf("cursor.LastError = %q, want mismatch diagnostic", cursor.LastError)
		}
	})

	t.Run("Should page past ignored records to reach a later terminal event", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusCompleted,
			}},
			records: []taskpkg.EventRecord{
				terminalTaskEventRecordForTest(1, "evt-1", "task-1", "run-1", "task.run_started"),
				terminalTaskEventRecordForTest(2, "evt-2", "task-1", "run-1", "task.run_heartbeat"),
				terminalTaskEventRecordForTest(3, "evt-3", "task-1", "run-1", terminalTaskEventRunCompleted),
			},
		})
		notifier.eventLimit = 2

		firstSweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue(first sweep) error = %v", err)
		}
		if firstSweep.Delivered != 0 || firstSweep.Deferred != 0 || firstSweep.Failed != 0 {
			t.Fatalf("DeliverDue(first sweep) = %#v, want pagination-only progress", firstSweep)
		}

		cursor, err := cursorStore.GetCursor(ctx, subscription.CursorKey())
		if err != nil {
			t.Fatalf("GetCursor(first sweep) error = %v", err)
		}
		if cursor.LastSequence != 2 || cursor.LastDeliveryID != "" {
			t.Fatalf("cursor after first sweep = %#v, want progress through ignored records", cursor)
		}

		secondSweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue(second sweep) error = %v", err)
		}
		if secondSweep.Delivered != 1 || secondSweep.Failed != 0 || secondSweep.Deferred != 0 {
			t.Fatalf("DeliverDue(second sweep) = %#v, want one delivered notification", secondSweep)
		}
		if calls := transport.snapshotCalls(); len(calls) != 1 {
			t.Fatalf("delivery calls = %d, want 1 after second sweep", len(calls))
		}
	})

	t.Run("Should redact claim tokens before emitting bridge notification errors", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusFailed,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusFailed,
				Error:  "bridge error agh_claim_secret-123 leaked",
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				7,
				"evt-7",
				"task-1",
				"run-1",
				terminalTaskEventRunFailed,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue() error = %v", err)
		}
		if sweep.Delivered != 1 {
			t.Fatalf("DeliverDue() sweep = %#v, want one delivered notification", sweep)
		}
		calls := transport.snapshotCalls()
		if len(calls) != 1 {
			t.Fatalf("delivery calls = %d, want 1", len(calls))
		}

		var envelope TerminalTaskNotification
		if err := json.Unmarshal(calls[0].request.Event.ProviderMetadata, &envelope); err != nil {
			t.Fatalf("Unmarshal(provider metadata) error = %v", err)
		}
		if strings.Contains(envelope.Error, "agh_claim_secret-123") {
			t.Fatalf("envelope error = %q, want redacted claim token", envelope.Error)
		}
		if !strings.Contains(envelope.Error, "agh_claim_[REDACTED]") {
			t.Fatalf("envelope error = %q, want redacted token marker", envelope.Error)
		}
		if strings.Contains(calls[0].request.Event.Content.Text, "agh_claim_secret-123") {
			t.Fatalf("delivery text = %q, want redacted claim token", calls[0].request.Event.Content.Text)
		}
	})

	t.Run("Should skip superseded terminal events and deliver the accepted final event", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{
				{
					ID:     "run-1",
					TaskID: "task-1",
					Status: taskpkg.TaskRunStatusFailed,
				},
				{
					ID:     "run-2",
					TaskID: "task-1",
					Status: taskpkg.TaskRunStatusCompleted,
				},
			},
			records: []taskpkg.EventRecord{
				terminalTaskEventRecordForTest(4, "evt-4", "task-1", "run-1", terminalTaskEventRunFailed),
				terminalTaskEventRecordForTest(9, "evt-9", "task-1", "run-2", terminalTaskEventRunCompleted),
			},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue() error = %v", err)
		}
		if sweep.Delivered != 1 || sweep.Failed != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want later accepted event delivered", sweep)
		}
		cursor, err := cursorStore.GetCursor(ctx, subscription.CursorKey())
		if err != nil {
			t.Fatalf("GetCursor() error = %v", err)
		}
		if cursor.LastSequence != 9 || cursor.LastDeliveryID != "notif:sub-1:9" {
			t.Fatalf("cursor = %#v, want final sequence 9", cursor)
		}
		if cursor.LastError != "" {
			t.Fatalf(
				"cursor.LastError = %q, want superseded mismatch to stay unrecorded after final delivery",
				cursor.LastError,
			)
		}
	})

	t.Run("Should not replay events at or before the stored cursor sequence", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		subscription := bridgeTaskSubscriptionForNotifierTest()
		cursorStore := newMemoryCursorStore()
		if _, err := cursorStore.AdvanceCursor(ctx, notifications.AdvanceCursor{
			Key:          subscription.CursorKey(),
			LastSequence: 7,
			DeliveryID:   "notif:sub-1:7",
			Now:          terminalTaskNotifierTestTime(),
		}); err != nil {
			t.Fatalf("AdvanceCursor(seed) error = %v", err)
		}
		transport := &fakeDeliveryTransport{}
		notifier := terminalTaskNotifierForTest(subscription, cursorStore, transport, terminalTaskReaderFixture{
			task: taskpkg.Task{
				ID:     "task-1",
				Status: taskpkg.TaskStatusCompleted,
			},
			runs: []taskpkg.Run{{
				ID:     "run-1",
				TaskID: "task-1",
				Status: taskpkg.TaskRunStatusCompleted,
			}},
			records: []taskpkg.EventRecord{terminalTaskEventRecordForTest(
				7,
				"evt-7",
				"task-1",
				"run-1",
				terminalTaskEventRunCompleted,
			)},
		})

		sweep, err := notifier.DeliverDue(ctx, BridgeTaskSubscriptionQuery{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("DeliverDue() error = %v", err)
		}
		if sweep.Delivered != 0 || sweep.Deferred != 0 || sweep.Failed != 0 {
			t.Fatalf("DeliverDue() sweep = %#v, want no replayed delivery", sweep)
		}
		if calls := transport.snapshotCalls(); len(calls) != 0 {
			t.Fatalf("delivery calls = %d, want 0 for cursor replay", len(calls))
		}
	})
}

func TestTruncateTerminalTaskCursorError(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve UTF-8 rune boundaries at the byte limit", func(t *testing.T) {
		t.Parallel()

		input := strings.Repeat("a", maxTerminalTaskCursorErrorBytes-1) + "界 trailing"

		got := truncateTerminalTaskCursorError(input)

		if len(got) > maxTerminalTaskCursorErrorBytes {
			t.Fatalf("len(truncated) = %d, want <= %d", len(got), maxTerminalTaskCursorErrorBytes)
		}
		if !utf8.ValidString(got) {
			t.Fatalf("truncateTerminalTaskCursorError() returned invalid UTF-8: %q", got)
		}
		if got != strings.Repeat("a", maxTerminalTaskCursorErrorBytes-1) {
			t.Fatalf("truncateTerminalTaskCursorError() = %q, want safe cut before multi-byte rune", got)
		}
	})
}

type terminalTaskReaderFixture struct {
	task    taskpkg.Task
	runs    []taskpkg.Run
	records []taskpkg.EventRecord
}

func terminalTaskNotifierForTest(
	subscription BridgeTaskSubscription,
	cursorStore notifications.CursorStore,
	transport DeliveryTransport,
	fixture terminalTaskReaderFixture,
) *TerminalTaskNotifier {
	instances := &fakeBridgeInstanceReader{
		instances: map[string]BridgeInstance{
			"brg-1": {
				ID:            "brg-1",
				Scope:         ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "telegram-extension",
				DisplayName:   "Telegram",
				Source:        BridgeInstanceSourceDynamic,
				Enabled:       true,
				Status:        BridgeStatusReady,
				DMPolicy:      BridgeDMPolicyOpen,
				RoutingPolicy: RoutingPolicy{IncludePeer: true},
				CreatedAt:     terminalTaskNotifierTestTime(),
				UpdatedAt:     terminalTaskNotifierTestTime(),
			},
		},
	}
	return NewTerminalTaskNotifier(TerminalTaskNotifierConfig{
		Subscriptions: &fakeBridgeTaskSubscriptionStore{subscriptions: []BridgeTaskSubscription{subscription}},
		TaskEvents:    fakeTerminalTaskEventReaderFromFixture(fixture),
		Instances:     instances,
		Cursors:       cursorStore,
		Transport:     transport,
		Now:           terminalTaskNotifierTestTime,
	})
}

func fakeTerminalTaskEventReaderFromFixture(fixture terminalTaskReaderFixture) *fakeTerminalTaskEventReader {
	reader := &fakeTerminalTaskEventReader{
		tasks:   map[string]taskpkg.Task{fixture.task.ID: fixture.task},
		runs:    make(map[string]taskpkg.Run, len(fixture.runs)),
		records: make([]taskpkg.EventRecord, 0, len(fixture.records)),
	}
	for _, run := range fixture.runs {
		reader.runs[run.ID] = run
	}
	reader.records = append(reader.records, fixture.records...)
	return reader
}

func bridgeTaskSubscriptionForNotifierTest() BridgeTaskSubscription {
	now := terminalTaskNotifierTestTime()
	return BridgeTaskSubscription{
		SubscriptionID:   "sub-1",
		TaskID:           "task-1",
		BridgeInstanceID: "brg-1",
		Scope:            ScopeGlobal,
		PeerID:           "peer-1",
		ThreadID:         "thread-1",
		DeliveryMode:     DeliveryModeReply,
		CreatedBy: taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKindDaemon,
			Ref:  "task-terminal-notifier",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func terminalTaskEventRecordForTest(
	sequence int64,
	eventID string,
	taskID string,
	runID string,
	eventType string,
) taskpkg.EventRecord {
	return taskpkg.EventRecord{
		Sequence: sequence,
		Event: taskpkg.Event{
			ID:        eventID,
			TaskID:    taskID,
			RunID:     runID,
			EventType: eventType,
			Actor: taskpkg.ActorIdentity{
				Kind: taskpkg.ActorKindDaemon,
				Ref:  "task-manager",
			},
			Origin: taskpkg.Origin{
				Kind: taskpkg.OriginKindDaemon,
				Ref:  "task-manager",
			},
			Payload:   json.RawMessage(`{"summary":"done"}`),
			Timestamp: terminalTaskNotifierTestTime(),
		},
	}
}

func terminalTaskNotifierTestTime() time.Time {
	return time.Date(2026, 5, 5, 17, 0, 0, 0, time.UTC)
}

type fakeBridgeTaskSubscriptionStore struct {
	subscriptions []BridgeTaskSubscription
}

func (s *fakeBridgeTaskSubscriptionStore) PutBridgeTaskSubscription(
	context.Context,
	BridgeTaskSubscription,
) error {
	return errors.New("unexpected PutBridgeTaskSubscription call")
}

func (s *fakeBridgeTaskSubscriptionStore) GetBridgeTaskSubscription(
	context.Context,
	string,
) (BridgeTaskSubscription, error) {
	return BridgeTaskSubscription{}, errors.New("unexpected GetBridgeTaskSubscription call")
}

func (s *fakeBridgeTaskSubscriptionStore) ListBridgeTaskSubscriptions(
	_ context.Context,
	query BridgeTaskSubscriptionQuery,
) ([]BridgeTaskSubscription, error) {
	normalized := query.Normalize()
	matches := make([]BridgeTaskSubscription, 0, len(s.subscriptions))
	for _, subscription := range s.subscriptions {
		subscription = subscription.Normalize()
		if normalized.TaskID != "" && subscription.TaskID != normalized.TaskID {
			continue
		}
		if normalized.BridgeInstanceID != "" && subscription.BridgeInstanceID != normalized.BridgeInstanceID {
			continue
		}
		if normalized.Scope != "" && subscription.Scope != normalized.Scope {
			continue
		}
		if normalized.WorkspaceID != "" && subscription.WorkspaceID != normalized.WorkspaceID {
			continue
		}
		matches = append(matches, subscription)
	}
	if normalized.Limit > 0 && len(matches) > normalized.Limit {
		matches = matches[:normalized.Limit]
	}
	return matches, nil
}

func (s *fakeBridgeTaskSubscriptionStore) DeleteBridgeTaskSubscription(context.Context, string) error {
	return errors.New("unexpected DeleteBridgeTaskSubscription call")
}

type fakeTerminalTaskEventReader struct {
	tasks   map[string]taskpkg.Task
	runs    map[string]taskpkg.Run
	records []taskpkg.EventRecord
}

func (r *fakeTerminalTaskEventReader) GetTask(_ context.Context, id string) (taskpkg.Task, error) {
	taskRecord, ok := r.tasks[id]
	if !ok {
		return taskpkg.Task{}, errors.New("task not found")
	}
	return taskRecord, nil
}

func (r *fakeTerminalTaskEventReader) GetTaskRun(_ context.Context, id string) (taskpkg.Run, error) {
	run, ok := r.runs[id]
	if !ok {
		return taskpkg.Run{}, errors.New("task run not found")
	}
	return run, nil
}

func (r *fakeTerminalTaskEventReader) ListTaskEventRecords(
	_ context.Context,
	query taskpkg.EventRecordQuery,
) ([]taskpkg.EventRecord, error) {
	if err := query.Validate("query"); err != nil {
		return nil, err
	}
	matches := make([]taskpkg.EventRecord, 0, len(r.records))
	for _, record := range r.records {
		if record.Event.TaskID != query.TaskID || record.Sequence <= query.AfterSequence {
			continue
		}
		matches = append(matches, record)
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].Sequence < matches[right].Sequence
	})
	if query.Limit > 0 && len(matches) > query.Limit {
		matches = matches[:query.Limit]
	}
	return matches, nil
}

type fakeBridgeInstanceReader struct {
	instances map[string]BridgeInstance
}

func (r *fakeBridgeInstanceReader) GetBridgeInstance(_ context.Context, id string) (BridgeInstance, error) {
	instance, ok := r.instances[id]
	if !ok {
		return BridgeInstance{}, ErrBridgeInstanceNotFound
	}
	return instance, nil
}

type memoryCursorStore struct {
	mu      sync.Mutex
	cursors map[string]notifications.Cursor
}

func newMemoryCursorStore() *memoryCursorStore {
	return &memoryCursorStore{cursors: make(map[string]notifications.Cursor)}
}

func (s *memoryCursorStore) GetCursor(
	_ context.Context,
	key notifications.CursorKey,
) (notifications.Cursor, error) {
	normalized, err := key.Normalize()
	if err != nil {
		return notifications.Cursor{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cursor, ok := s.cursors[memoryCursorStoreKey(normalized)]
	if !ok {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func (s *memoryCursorStore) ListCursors(
	_ context.Context,
	query notifications.CursorQuery,
) ([]notifications.Cursor, error) {
	normalized := query.Normalize()
	s.mu.Lock()
	defer s.mu.Unlock()
	cursors := make([]notifications.Cursor, 0, len(s.cursors))
	for _, cursor := range s.cursors {
		if normalized.ConsumerID != "" && cursor.Key.ConsumerID != normalized.ConsumerID {
			continue
		}
		if normalized.StreamName != "" && cursor.Key.StreamName != normalized.StreamName {
			continue
		}
		if normalized.SubjectID != "" && cursor.Key.SubjectID != normalized.SubjectID {
			continue
		}
		cursors = append(cursors, cursor)
	}
	sort.Slice(cursors, func(left int, right int) bool {
		if cursors[left].Key.StreamName != cursors[right].Key.StreamName {
			return cursors[left].Key.StreamName < cursors[right].Key.StreamName
		}
		if cursors[left].Key.SubjectID != cursors[right].Key.SubjectID {
			return cursors[left].Key.SubjectID < cursors[right].Key.SubjectID
		}
		return cursors[left].Key.ConsumerID < cursors[right].Key.ConsumerID
	})
	if normalized.Limit > 0 && len(cursors) > normalized.Limit {
		cursors = cursors[:normalized.Limit]
	}
	return cursors, nil
}

func (s *memoryCursorStore) AdvanceCursor(
	_ context.Context,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	normalized, err := update.Normalize(terminalTaskNotifierTestTime())
	if err != nil {
		return notifications.Cursor{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryCursorStoreKey(normalized.Key)
	current, ok := s.cursors[key]
	if ok {
		if normalized.LastSequence < current.LastSequence ||
			(normalized.LastSequence == current.LastSequence && normalized.DeliveryID != current.LastDeliveryID) {
			return notifications.Cursor{}, notifications.ErrNonMonotonicCursor
		}
		if normalized.LastSequence == current.LastSequence && normalized.DeliveryID == current.LastDeliveryID {
			return current, nil
		}
	}
	cursor := notifications.Cursor{
		Key:             normalized.Key,
		LastSequence:    normalized.LastSequence,
		LastDeliveryID:  normalized.DeliveryID,
		LastDeliveredAt: normalized.LastDeliveredAt,
		UpdatedAt:       normalized.Now,
	}
	s.cursors[key] = cursor
	return cursor, nil
}

func (s *memoryCursorStore) ResetCursor(
	_ context.Context,
	reset notifications.ResetCursor,
) (notifications.Cursor, error) {
	normalized, err := reset.Normalize(terminalTaskNotifierTestTime())
	if err != nil {
		return notifications.Cursor{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cursor := notifications.Cursor{
		Key:             normalized.Key,
		LastSequence:    normalized.LastSequence,
		LastDeliveryID:  normalized.LastDeliveryID,
		LastDeliveredAt: normalized.LastDeliveredAt,
		LastError:       normalized.Reason,
		UpdatedAt:       normalized.Now,
	}
	s.cursors[memoryCursorStoreKey(normalized.Key)] = cursor
	return cursor, nil
}

func (s *memoryCursorStore) RecordCursorError(
	_ context.Context,
	report notifications.CursorError,
) (notifications.Cursor, error) {
	normalized, err := report.Normalize(terminalTaskNotifierTestTime())
	if err != nil {
		return notifications.Cursor{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryCursorStoreKey(normalized.Key)
	cursor := s.cursors[key]
	cursor.Key = normalized.Key
	cursor.LastError = normalized.LastError
	cursor.UpdatedAt = normalized.Now
	s.cursors[key] = cursor
	return cursor, nil
}

func memoryCursorStoreKey(key notifications.CursorKey) string {
	return key.ConsumerID + "\x00" + key.StreamName + "\x00" + key.SubjectID
}
