package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	terminalTaskEventRunCompleted      = "task.run_completed"
	terminalTaskEventRunFailed         = "task.run_failed"
	terminalTaskEventRunCanceled       = "task.run_canceled"
	terminalTaskEventCanceled          = "task.canceled"
	terminalTaskEventRunReviewApproved = "task.run_review_approved"

	defaultTerminalTaskNotifierLimit = 100
	maxTerminalTaskCursorErrorBytes  = 2048
)

// ErrTerminalTaskNotificationMismatch reports a replayed terminal task event
// that claims finality but no longer agrees with the durable task/run state.
var ErrTerminalTaskNotificationMismatch = errors.New("bridges: terminal task notification state mismatch")

// TerminalTaskEventReader reads the durable task state used to replay terminal notifications.
type TerminalTaskEventReader interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskEventRecords(ctx context.Context, query taskpkg.EventRecordQuery) ([]taskpkg.EventRecord, error)
}

// BridgeInstanceReader loads bridge instances for direct adapter delivery.
type BridgeInstanceReader interface {
	GetBridgeInstance(ctx context.Context, id string) (BridgeInstance, error)
}

// TerminalTaskNotifierConfig wires the bridge terminal notifier.
type TerminalTaskNotifierConfig struct {
	Subscriptions BridgeTaskSubscriptionStore
	TaskEvents    TerminalTaskEventReader
	Instances     BridgeInstanceReader
	Cursors       notifications.CursorStore
	Transport     DeliveryTransport
	Now           func() time.Time
	EventLimit    int
}

// TerminalTaskNotificationSweep summarizes one notifier replay pass.
type TerminalTaskNotificationSweep struct {
	Subscriptions int `json:"subscriptions"`
	Delivered     int `json:"delivered"`
	Deferred      int `json:"deferred"`
	Failed        int `json:"failed"`
}

// TerminalTaskNotifier replays durable task events into subscribed bridge targets.
type TerminalTaskNotifier struct {
	subscriptions BridgeTaskSubscriptionStore
	taskEvents    TerminalTaskEventReader
	instances     BridgeInstanceReader
	cursorStore   notifications.CursorStore
	cursors       *notifications.Service
	transport     DeliveryTransport
	now           func() time.Time
	eventLimit    int
}

// NewTerminalTaskNotifier constructs a bridge terminal notifier.
func NewTerminalTaskNotifier(cfg TerminalTaskNotifierConfig) *TerminalTaskNotifier {
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	limit := cfg.EventLimit
	if limit <= 0 {
		limit = defaultTerminalTaskNotifierLimit
	}
	return &TerminalTaskNotifier{
		subscriptions: cfg.Subscriptions,
		taskEvents:    cfg.TaskEvents,
		instances:     cfg.Instances,
		cursorStore:   cfg.Cursors,
		cursors:       notifications.NewService(cfg.Cursors),
		transport:     cfg.Transport,
		now:           now,
		eventLimit:    limit,
	}
}

// DeliverDue replays task events for subscriptions matching the query.
func (n *TerminalTaskNotifier) DeliverDue(
	ctx context.Context,
	query BridgeTaskSubscriptionQuery,
) (TerminalTaskNotificationSweep, error) {
	if err := n.checkReady(); err != nil {
		return TerminalTaskNotificationSweep{}, err
	}
	subscriptions, err := n.subscriptions.ListBridgeTaskSubscriptions(ctx, query)
	if err != nil {
		return TerminalTaskNotificationSweep{}, fmt.Errorf("bridges: list task notification subscriptions: %w", err)
	}

	sweep := TerminalTaskNotificationSweep{Subscriptions: len(subscriptions)}
	var joined error
	for _, subscription := range subscriptions {
		outcome, processErr := n.deliverSubscription(ctx, subscription)
		switch outcome {
		case terminalTaskNotificationDelivered:
			sweep.Delivered++
		case terminalTaskNotificationDeferred:
			sweep.Deferred++
		case terminalTaskNotificationFailed:
			sweep.Failed++
		}
		if processErr != nil {
			joined = errors.Join(joined, processErr)
		}
	}
	return sweep, joined
}

type terminalTaskNotificationOutcome string

const (
	terminalTaskNotificationNoop      terminalTaskNotificationOutcome = ""
	terminalTaskNotificationDelivered terminalTaskNotificationOutcome = "delivered"
	terminalTaskNotificationDeferred  terminalTaskNotificationOutcome = "deferred"
	terminalTaskNotificationFailed    terminalTaskNotificationOutcome = "failed"
)

type terminalTaskNotificationDecision string

const (
	terminalTaskNotificationDecisionIgnore   terminalTaskNotificationDecision = ""
	terminalTaskNotificationDecisionDeliver  terminalTaskNotificationDecision = "deliver"
	terminalTaskNotificationDecisionDefer    terminalTaskNotificationDecision = "defer"
	terminalTaskNotificationDecisionMismatch terminalTaskNotificationDecision = "mismatch"
)

type terminalTaskNotificationResolution struct {
	notification TerminalTaskNotification
	decision     terminalTaskNotificationDecision
	diagnostic   error
}

func (n *TerminalTaskNotifier) checkReady() error {
	switch {
	case n == nil:
		return errors.New("bridges: terminal task notifier is required")
	case n.subscriptions == nil:
		return errors.New("bridges: terminal task notifier subscriptions store is required")
	case n.taskEvents == nil:
		return errors.New("bridges: terminal task notifier task event reader is required")
	case n.instances == nil:
		return errors.New("bridges: terminal task notifier bridge instance reader is required")
	case n.cursorStore == nil || n.cursors == nil:
		return errors.New("bridges: terminal task notifier cursor store is required")
	case n.transport == nil:
		return ErrDeliveryTransportUnavailable
	default:
		return nil
	}
}

func (n *TerminalTaskNotifier) deliverSubscription(
	ctx context.Context,
	subscription BridgeTaskSubscription,
) (terminalTaskNotificationOutcome, error) {
	normalized := subscription.Normalize()
	if err := normalized.Validate(); err != nil {
		return terminalTaskNotificationFailed, err
	}

	cursorKey, cursor, err := n.loadTaskNotificationCursor(ctx, normalized)
	if err != nil {
		return terminalTaskNotificationFailed, err
	}
	records, err := n.listTaskNotificationRecords(ctx, normalized, cursor)
	if err != nil {
		return terminalTaskNotificationFailed, err
	}

	deferred := false
	var mismatchErr error
	for _, record := range records {
		if !isTerminalTaskNotificationCandidate(record.Event.EventType) {
			continue
		}

		resolution, err := n.resolveTerminalTaskNotification(ctx, normalized, record)
		if err != nil {
			if recordErr := n.recordCursorError(ctx, cursorKey, err); recordErr != nil {
				err = errors.Join(err, recordErr)
			}
			return terminalTaskNotificationFailed, err
		}
		switch resolution.decision {
		case terminalTaskNotificationDecisionDeliver:
		case terminalTaskNotificationDecisionMismatch:
			mismatchErr = errors.Join(mismatchErr, resolution.diagnostic)
			continue
		case terminalTaskNotificationDecisionDefer:
			deferred = true
			continue
		default:
			continue
		}
		if err := n.deliverNotification(ctx, normalized, resolution.notification); err != nil {
			if recordErr := n.recordCursorError(ctx, cursorKey, err); recordErr != nil {
				err = errors.Join(err, recordErr)
			}
			return terminalTaskNotificationFailed, err
		}
		if _, err := n.cursors.Advance(ctx, notifications.AdvanceCursor{
			Key:          cursorKey,
			LastSequence: record.Sequence,
			DeliveryID:   resolution.notification.DeliveryID,
			Now:          n.now(),
		}); err != nil {
			return terminalTaskNotificationFailed, fmt.Errorf(
				"bridges: advance task notification cursor for subscription %q: %w",
				normalized.SubscriptionID,
				err,
			)
		}
		return terminalTaskNotificationDelivered, nil
	}

	if mismatchErr != nil {
		if recordErr := n.recordCursorError(ctx, cursorKey, mismatchErr); recordErr != nil {
			mismatchErr = errors.Join(mismatchErr, recordErr)
		}
		return terminalTaskNotificationFailed, mismatchErr
	}
	if deferred {
		return terminalTaskNotificationDeferred, nil
	}
	return terminalTaskNotificationNoop, nil
}

func (n *TerminalTaskNotifier) loadTaskNotificationCursor(
	ctx context.Context,
	subscription BridgeTaskSubscription,
) (notifications.CursorKey, notifications.Cursor, error) {
	cursorKey := subscription.CursorKey()
	cursor, err := n.cursors.Get(ctx, cursorKey)
	if err != nil && !errors.Is(err, notifications.ErrCursorNotFound) {
		return notifications.CursorKey{}, notifications.Cursor{}, fmt.Errorf(
			"bridges: load task notification cursor for subscription %q: %w",
			subscription.SubscriptionID,
			err,
		)
	}
	return cursorKey, cursor, nil
}

func (n *TerminalTaskNotifier) listTaskNotificationRecords(
	ctx context.Context,
	subscription BridgeTaskSubscription,
	cursor notifications.Cursor,
) ([]taskpkg.EventRecord, error) {
	records, err := n.taskEvents.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{
		TaskID:        subscription.TaskID,
		AfterSequence: cursor.LastSequence,
		Limit:         n.eventLimit,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"bridges: list task events for subscription %q: %w",
			subscription.SubscriptionID,
			err,
		)
	}
	return records, nil
}

func (n *TerminalTaskNotifier) resolveTerminalTaskNotification(
	ctx context.Context,
	subscription BridgeTaskSubscription,
	record taskpkg.EventRecord,
) (terminalTaskNotificationResolution, error) {
	taskRecord, err := n.taskEvents.GetTask(ctx, subscription.TaskID)
	if err != nil {
		return terminalTaskNotificationResolution{}, fmt.Errorf(
			"bridges: load task %q for terminal notification: %w",
			subscription.TaskID,
			err,
		)
	}
	taskStatus := taskRecord.Status.Normalize()
	eventStatus, ok := taskStatusForTerminalEvent(record.Event.EventType)
	if !ok {
		return terminalTaskNotificationResolution{decision: terminalTaskNotificationDecisionIgnore}, nil
	}
	if !isTaskTerminalStatus(taskStatus) {
		return terminalTaskNotificationResolution{decision: terminalTaskNotificationDecisionDefer}, nil
	}
	if taskStatus != eventStatus {
		return terminalTaskNotificationResolution{
			decision: terminalTaskNotificationDecisionMismatch,
			diagnostic: terminalTaskNotificationStatusMismatchError(
				subscription,
				record,
				taskStatus,
				eventStatus,
			),
		}, nil
	}

	var run taskpkg.Run
	if strings.TrimSpace(record.Event.RunID) != "" {
		run, err = n.taskEvents.GetTaskRun(ctx, record.Event.RunID)
		if err != nil {
			return terminalTaskNotificationResolution{}, fmt.Errorf(
				"bridges: load task run %q for terminal notification: %w",
				record.Event.RunID,
				err,
			)
		}
		if strings.TrimSpace(run.TaskID) != subscription.TaskID {
			return terminalTaskNotificationResolution{}, fmt.Errorf(
				"bridges: terminal event run %q belongs to task %q, want %q",
				run.ID,
				run.TaskID,
				subscription.TaskID,
			)
		}
		if statusFromRun, runOK := taskStatusForRunStatus(run.Status); !runOK || statusFromRun != eventStatus {
			return terminalTaskNotificationResolution{
				decision: terminalTaskNotificationDecisionMismatch,
				diagnostic: terminalTaskNotificationRunMismatchError(
					subscription,
					record,
					run.Status.Normalize(),
					eventStatus,
				),
			}, nil
		}
	}

	deliveryID := terminalTaskNotificationDeliveryID(subscription.SubscriptionID, record.Sequence)
	return terminalTaskNotificationResolution{
		decision: terminalTaskNotificationDecisionDeliver,
		notification: TerminalTaskNotification{
			DeliveryID:     deliveryID,
			EventType:      record.Event.EventType,
			Final:          true,
			Seq:            record.Sequence,
			TaskID:         subscription.TaskID,
			RunID:          strings.TrimSpace(record.Event.RunID),
			Status:         eventStatus,
			Error:          strings.TrimSpace(run.Error),
			Payload:        cloneRawJSON(record.Event.Payload),
			SubscriptionID: subscription.SubscriptionID,
		},
	}, nil
}

func terminalTaskNotificationStatusMismatchError(
	subscription BridgeTaskSubscription,
	record taskpkg.EventRecord,
	current taskpkg.Status,
	expected taskpkg.Status,
) error {
	return fmt.Errorf(
		"%w: subscription %q task %q event %q sequence %d expects task status %q but current task status is %q",
		ErrTerminalTaskNotificationMismatch,
		subscription.SubscriptionID,
		subscription.TaskID,
		strings.TrimSpace(record.Event.EventType),
		record.Sequence,
		expected.Normalize(),
		current.Normalize(),
	)
}

func terminalTaskNotificationRunMismatchError(
	subscription BridgeTaskSubscription,
	record taskpkg.EventRecord,
	current taskpkg.RunStatus,
	expected taskpkg.Status,
) error {
	return fmt.Errorf(
		"%w: subscription %q task %q run %q event %q sequence %d "+
			"expects run-derived task status %q but current run status is %q",
		ErrTerminalTaskNotificationMismatch,
		subscription.SubscriptionID,
		subscription.TaskID,
		strings.TrimSpace(record.Event.RunID),
		strings.TrimSpace(record.Event.EventType),
		record.Sequence,
		expected.Normalize(),
		current.Normalize(),
	)
}

func (n *TerminalTaskNotifier) deliverNotification(
	ctx context.Context,
	subscription BridgeTaskSubscription,
	notification TerminalTaskNotification,
) error {
	instance, err := n.instances.GetBridgeInstance(ctx, subscription.BridgeInstanceID)
	if err != nil {
		return fmt.Errorf(
			"bridges: load bridge instance %q for task notification: %w",
			subscription.BridgeInstanceID,
			err,
		)
	}
	if !instance.Enabled || instance.Status.Normalize() != BridgeStatusReady {
		return fmt.Errorf(
			"%w: bridge instance %q status %q enabled=%t",
			ErrBridgeInstanceUnavailable,
			instance.ID,
			instance.Status.Normalize(),
			instance.Enabled,
		)
	}

	metadata, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("bridges: encode terminal task notification: %w", err)
	}
	event := DeliveryEvent{
		DeliveryID:       notification.DeliveryID,
		BridgeInstanceID: subscription.BridgeInstanceID,
		RoutingKey:       subscription.RoutingKey(),
		DeliveryTarget:   subscription.DeliveryTarget(),
		Seq:              notification.Seq,
		EventType:        DeliveryEventTypeFinal,
		Content:          MessageContent{Text: terminalTaskNotificationText(notification)},
		Final:            true,
		Operation:        DeliveryOperationPost,
		ProviderMetadata: metadata,
	}
	if err := event.Validate(); err != nil {
		return err
	}

	ack, err := n.transport.DeliverBridge(ctx, instance.ExtensionName, DeliveryRequest{Event: event})
	if err != nil {
		return fmt.Errorf(
			"bridges: deliver terminal task notification %q: %w",
			notification.DeliveryID,
			err,
		)
	}
	if err := ack.ValidateFor(event); err != nil {
		return fmt.Errorf(
			"bridges: validate terminal task notification ack %q: %w",
			notification.DeliveryID,
			err,
		)
	}
	return nil
}

func (n *TerminalTaskNotifier) recordCursorError(
	ctx context.Context,
	key notifications.CursorKey,
	cause error,
) error {
	if _, err := n.cursors.RecordError(ctx, notifications.CursorError{
		Key:       key,
		LastError: truncateTerminalTaskCursorError(cause.Error()),
		Now:       n.now(),
	}); err != nil {
		return fmt.Errorf("bridges: record terminal task notification cursor error: %w", err)
	}
	return nil
}

func isTerminalTaskNotificationCandidate(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case terminalTaskEventRunCompleted,
		terminalTaskEventRunFailed,
		terminalTaskEventRunCanceled,
		terminalTaskEventCanceled,
		terminalTaskEventRunReviewApproved:
		return true
	default:
		return false
	}
}

func taskStatusForTerminalEvent(eventType string) (taskpkg.Status, bool) {
	switch strings.TrimSpace(eventType) {
	case terminalTaskEventRunCompleted, terminalTaskEventRunReviewApproved:
		return taskpkg.TaskStatusCompleted, true
	case terminalTaskEventRunFailed:
		return taskpkg.TaskStatusFailed, true
	case terminalTaskEventRunCanceled, terminalTaskEventCanceled:
		return taskpkg.TaskStatusCanceled, true
	default:
		return "", false
	}
}

func taskStatusForRunStatus(status taskpkg.RunStatus) (taskpkg.Status, bool) {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusCompleted:
		return taskpkg.TaskStatusCompleted, true
	case taskpkg.TaskRunStatusFailed:
		return taskpkg.TaskStatusFailed, true
	case taskpkg.TaskRunStatusCanceled:
		return taskpkg.TaskStatusCanceled, true
	default:
		return "", false
	}
}

func isTaskTerminalStatus(status taskpkg.Status) bool {
	switch status.Normalize() {
	case taskpkg.TaskStatusCompleted, taskpkg.TaskStatusFailed, taskpkg.TaskStatusCanceled:
		return true
	default:
		return false
	}
}

func terminalTaskNotificationDeliveryID(subscriptionID string, sequence int64) string {
	return fmt.Sprintf("notif:%s:%d", strings.TrimSpace(subscriptionID), sequence)
}

func terminalTaskNotificationText(notification TerminalTaskNotification) string {
	status := notification.Status.Normalize()
	if notification.Error != "" {
		return fmt.Sprintf("Task %s finished as %s: %s", notification.TaskID, status, notification.Error)
	}
	return fmt.Sprintf("Task %s finished as %s", notification.TaskID, status)
}

func truncateTerminalTaskCursorError(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= maxTerminalTaskCursorErrorBytes {
		return trimmed
	}
	return trimmed[:maxTerminalTaskCursorErrorBytes]
}
