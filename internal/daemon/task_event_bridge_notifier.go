package daemon

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const defaultBridgeTerminalNotificationQueueSize = 32

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
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	pending  map[string]struct{}
	backlog  []bridgeTerminalTaskNotificationWake
	queue    chan bridgeTerminalTaskNotificationWake
}

var _ taskpkg.EventObserver = (*bridgeTerminalTaskNotificationObserver)(nil)

type bridgeTerminalTaskNotificationWake struct {
	taskID    string
	eventID   string
	runID     string
	eventType string
}

func (d *Daemon) composeTaskEventObserver(
	state *bootState,
	store taskStore,
	reentry taskpkg.EventObserver,
) (taskpkg.EventObserver, *bridgeTerminalTaskNotificationObserver) {
	if state == nil {
		return reentry, nil
	}
	bridgeEventObserver := newBridgeTerminalTaskNotificationObserver(
		state.bridges,
		store,
		state.bridges,
		state.bridges,
		state.bridges,
		state.logger,
		d.now,
		state.cfg.Task.Orchestration.BridgeNotificationTimeout,
	)
	return newTaskEventObserverFanout(state.logger, reentry, bridgeEventObserver), bridgeEventObserver
}

func newBridgeTerminalTaskNotificationObserver(
	subscriptions bridgepkg.BridgeTaskSubscriptionStore,
	taskEvents bridgepkg.TerminalTaskEventReader,
	instances bridgepkg.BridgeInstanceReader,
	cursors notifications.CursorStore,
	transport bridgepkg.DeliveryTransport,
	logger *slog.Logger,
	now func() time.Time,
	timeout time.Duration,
) *bridgeTerminalTaskNotificationObserver {
	if subscriptions == nil || taskEvents == nil || instances == nil || cursors == nil || transport == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	observer := &bridgeTerminalTaskNotificationObserver{
		notifier: bridgepkg.NewTerminalTaskNotifier(bridgepkg.TerminalTaskNotifierConfig{
			Subscriptions: subscriptions,
			TaskEvents:    taskEvents,
			Instances:     instances,
			Cursors:       cursors,
			Transport:     transport,
			Now:           now,
		}),
		logger:  logger,
		timeout: timeout,
		ctx:     ctx,
		cancel:  cancel,
		pending: make(map[string]struct{}),
		queue:   make(chan bridgeTerminalTaskNotificationWake, defaultBridgeTerminalNotificationQueueSize),
	}
	observer.start()
	return observer
}

func (o *bridgeTerminalTaskNotificationObserver) OnTaskEvent(
	_ context.Context,
	record taskpkg.EventRecord,
) {
	if o == nil || o.notifier == nil || !isBridgeTerminalNotificationWake(record.Event.EventType) {
		return
	}
	o.enqueue(record)
}

func (o *bridgeTerminalTaskNotificationObserver) shutdown() {
	if o == nil {
		return
	}
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
}

func (o *bridgeTerminalTaskNotificationObserver) start() {
	if o == nil {
		return
	}
	o.wg.Go(func() {
		for {
			select {
			case <-o.ctx.Done():
				return
			case wake := <-o.queue:
				o.processWake(wake)
			}
		}
	})
}

func (o *bridgeTerminalTaskNotificationObserver) enqueue(record taskpkg.EventRecord) {
	if o == nil {
		return
	}
	wake := bridgeTerminalTaskNotificationWake{
		taskID:    strings.TrimSpace(record.Event.TaskID),
		eventID:   strings.TrimSpace(record.Event.ID),
		runID:     strings.TrimSpace(record.Event.RunID),
		eventType: strings.TrimSpace(record.Event.EventType),
	}
	if wake.taskID == "" {
		return
	}
	o.mu.Lock()
	if _, exists := o.pending[wake.taskID]; exists {
		o.mu.Unlock()
		return
	}
	o.pending[wake.taskID] = struct{}{}
	o.backlog = append(o.backlog, wake)
	o.mu.Unlock()
	o.drainQueue()
}

func (o *bridgeTerminalTaskNotificationObserver) drainQueue() {
	if o == nil {
		return
	}
	for {
		o.mu.Lock()
		if len(o.backlog) == 0 {
			o.mu.Unlock()
			return
		}
		wake := o.backlog[0]
		select {
		case o.queue <- wake:
			o.backlog = o.backlog[1:]
			o.mu.Unlock()
		default:
			o.mu.Unlock()
			return
		}
	}
}

func (o *bridgeTerminalTaskNotificationObserver) processWake(wake bridgeTerminalTaskNotificationWake) {
	defer func() {
		o.mu.Lock()
		delete(o.pending, wake.taskID)
		o.mu.Unlock()
		o.drainQueue()
	}()

	notifyCtx := context.Background()
	cancel := func() {}
	if o.timeout > 0 {
		notifyCtx, cancel = context.WithTimeout(notifyCtx, o.timeout)
	}
	defer cancel()

	sweep, err := o.notifier.DeliverDue(
		notifyCtx,
		bridgepkg.BridgeTaskSubscriptionQuery{TaskID: wake.taskID},
	)
	if err != nil {
		o.log().Warn(
			"daemon: bridge terminal task notification delivery failed",
			"error", err,
			"event_id", wake.eventID,
			"task_id", wake.taskID,
			"run_id", wake.runID,
			"event_type", wake.eventType,
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
			"event_id", wake.eventID,
			"task_id", wake.taskID,
			"run_id", wake.runID,
			"event_type", wake.eventType,
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
	case harnessTaskEventRunCompleted,
		"task.run_failed",
		"task.run_canceled",
		"task.run_review_approved",
		"task.canceled":
		return true
	default:
		return false
	}
}
