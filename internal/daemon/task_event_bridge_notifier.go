package daemon

import (
	"context"
	"log/slog"
	"strings"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const defaultBridgeTerminalNotificationTimeout = 10 * time.Second

type taskEventObserverFanout struct {
	observers []taskpkg.EventObserver
	logger    *slog.Logger
}

var _ taskpkg.EventObserver = (*taskEventObserverFanout)(nil)

func newTaskEventObserverFanout(
	logger *slog.Logger,
	observers ...taskpkg.EventObserver,
) taskpkg.EventObserver {
	filtered := make([]taskpkg.EventObserver, 0, len(observers))
	for _, observer := range observers {
		if observer != nil {
			filtered = append(filtered, observer)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &taskEventObserverFanout{observers: filtered, logger: logger}
	}
}

func (f *taskEventObserverFanout) OnTaskEvent(ctx context.Context, record taskpkg.EventRecord) {
	if f == nil {
		return
	}
	for _, observer := range f.observers {
		if observer == nil {
			continue
		}
		func(target taskpkg.EventObserver) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger := f.logger
					if logger == nil {
						logger = slog.Default()
					}
					logger.Error(
						"daemon: task event observer panicked during fanout",
						"panic", recovered,
						"event_id", record.Event.ID,
						"task_id", record.Event.TaskID,
						"run_id", record.Event.RunID,
						"event_type", record.Event.EventType,
					)
				}
			}()
			target.OnTaskEvent(ctx, record)
		}(observer)
	}
}

type bridgeTerminalTaskNotificationObserver struct {
	notifier *bridgepkg.TerminalTaskNotifier
	logger   *slog.Logger
	timeout  time.Duration
}

var _ taskpkg.EventObserver = (*bridgeTerminalTaskNotificationObserver)(nil)

func (d *Daemon) composeTaskEventObserver(
	state *bootState,
	store taskStore,
	reentry taskpkg.EventObserver,
) taskpkg.EventObserver {
	if state == nil {
		return reentry
	}
	bridgeEventObserver := newBridgeTerminalTaskNotificationObserver(
		state.bridges,
		store,
		state.bridges,
		state.bridges,
		state.bridges,
		state.logger,
		d.now,
	)
	return newTaskEventObserverFanout(state.logger, reentry, bridgeEventObserver)
}

func newBridgeTerminalTaskNotificationObserver(
	subscriptions bridgepkg.BridgeTaskSubscriptionStore,
	taskEvents bridgepkg.TerminalTaskEventReader,
	instances bridgepkg.BridgeInstanceReader,
	cursors notifications.CursorStore,
	transport bridgepkg.DeliveryTransport,
	logger *slog.Logger,
	now func() time.Time,
) taskpkg.EventObserver {
	if subscriptions == nil || taskEvents == nil || instances == nil || cursors == nil || transport == nil {
		return nil
	}
	return &bridgeTerminalTaskNotificationObserver{
		notifier: bridgepkg.NewTerminalTaskNotifier(bridgepkg.TerminalTaskNotifierConfig{
			Subscriptions: subscriptions,
			TaskEvents:    taskEvents,
			Instances:     instances,
			Cursors:       cursors,
			Transport:     transport,
			Now:           now,
		}),
		logger:  logger,
		timeout: defaultBridgeTerminalNotificationTimeout,
	}
}

func (o *bridgeTerminalTaskNotificationObserver) OnTaskEvent(
	ctx context.Context,
	record taskpkg.EventRecord,
) {
	if o == nil || o.notifier == nil || !isBridgeTerminalNotificationWake(record.Event.EventType) {
		return
	}
	if ctx == nil {
		o.log().Warn(
			"daemon: skipped bridge terminal task notification wake without context",
			"event_id", record.Event.ID,
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
		)
		return
	}
	notifyCtx := context.WithoutCancel(ctx)
	cancel := func() {}
	if o.timeout > 0 {
		notifyCtx, cancel = context.WithTimeout(notifyCtx, o.timeout)
	}
	defer cancel()

	sweep, err := o.notifier.DeliverDue(
		notifyCtx,
		bridgepkg.BridgeTaskSubscriptionQuery{TaskID: record.Event.TaskID},
	)
	if err != nil {
		o.log().Warn(
			"daemon: bridge terminal task notification delivery failed",
			"error", err,
			"event_id", record.Event.ID,
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
			"subscriptions", sweep.Subscriptions,
			"delivered", sweep.Delivered,
			"deferred", sweep.Deferred,
			"failed", sweep.Failed,
		)
		return
	}
	if sweep.Delivered > 0 || sweep.Deferred > 0 || sweep.Failed > 0 {
		o.log().Debug(
			"daemon: bridge terminal task notification sweep complete",
			"event_id", record.Event.ID,
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
			"subscriptions", sweep.Subscriptions,
			"delivered", sweep.Delivered,
			"deferred", sweep.Deferred,
			"failed", sweep.Failed,
		)
	}
}

func (o *bridgeTerminalTaskNotificationObserver) log() *slog.Logger {
	if o != nil && o.logger != nil {
		return o.logger
	}
	return slog.Default()
}

func isBridgeTerminalNotificationWake(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "task.run_completed",
		"task.run_failed",
		"task.run_canceled",
		"task.run_review_approved",
		"task.canceled":
		return true
	default:
		return false
	}
}
