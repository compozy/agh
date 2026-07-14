package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/network/participation"
	storepkg "github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	taskEventCanceled     = "task.canceled"
	taskEventRunStarted   = "task.run_started"
	taskEventRunCompleted = "task.run_completed"
	taskEventRunFailed    = "task.run_failed"
	taskEventRunCanceled  = "task.run_canceled"
	taskHookRunStarted    = "task.run.started"
	taskHookRunCompleted  = "task.run.completed"
	taskHookRunFailed     = "task.run.failed"
	taskHookRunCanceled   = "task.run.canceled"
)

const (
	taskConversationAvailable         = "available"
	taskConversationNotLinked         = "not_linked"
	taskConversationNetworkDisabled   = "network_disabled"
	taskConversationAvailabilityError = "availability_error"
)

type taskStatusProjection struct {
	EventID                  string
	EventType                string
	TaskID                   string
	RunID                    string
	TaskStatus               taskpkg.Status
	RunStatus                taskpkg.RunStatus
	NetworkParticipation     participation.Spec
	ThreadOrigin             *storepkg.NetworkTaskThreadOrigin
	ConversationAvailable    bool
	ConversationAvailability string
	DesignationRollup        *taskDesignationRollupStatus
	ProjectedAt              time.Time
}

type taskStatusProjectionPublisher interface {
	PublishTaskStatusProjection(context.Context, taskStatusProjection)
}

type taskStatusProjectionObserver struct {
	tasks        taskStore
	prefs        storepkg.NetworkPreferenceStore
	availability storepkg.NetworkAvailabilityStore
	publisher    taskStatusProjectionPublisher
	logger       *slog.Logger
	now          func() time.Time
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	queue        chan taskpkg.EventRecord
	timeout      time.Duration
}

var _ taskpkg.EventObserver = (*taskStatusProjectionObserver)(nil)

type taskStatusProjectionObserverOptions struct {
	logger    *slog.Logger
	now       func() time.Time
	queueSize int
	timeout   time.Duration
}

type taskStatusProjectionObserverOption func(*taskStatusProjectionObserverOptions)

func withTaskStatusProjectionObserverLogger(logger *slog.Logger) taskStatusProjectionObserverOption {
	return func(options *taskStatusProjectionObserverOptions) { options.logger = logger }
}

func withTaskStatusProjectionObserverClock(now func() time.Time) taskStatusProjectionObserverOption {
	return func(options *taskStatusProjectionObserverOptions) { options.now = now }
}

func withTaskStatusProjectionObserverQueueSize(queueSize int) taskStatusProjectionObserverOption {
	return func(options *taskStatusProjectionObserverOptions) { options.queueSize = queueSize }
}

func withTaskStatusProjectionObserverTimeout(timeout time.Duration) taskStatusProjectionObserverOption {
	return func(options *taskStatusProjectionObserverOptions) { options.timeout = timeout }
}

func newTaskStatusProjectionObserver(
	tasks taskStore,
	publisher taskStatusProjectionPublisher,
	opts ...taskStatusProjectionObserverOption,
) *taskStatusProjectionObserver {
	prefs, ok := tasks.(storepkg.NetworkPreferenceStore)
	availability, availabilityOK := tasks.(storepkg.NetworkAvailabilityStore)
	if tasks == nil || !ok || !availabilityOK || publisher == nil {
		return nil
	}
	options := taskStatusProjectionObserverOptions{
		logger: slog.Default(), now: time.Now,
		queueSize: aghconfig.DefaultTaskNetworkStatusQueueSize,
		timeout:   aghconfig.DefaultTaskNetworkStatusTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.logger == nil {
		options.logger = slog.Default()
	}
	if options.now == nil {
		options.now = time.Now
	}
	if options.queueSize <= 0 {
		options.queueSize = aghconfig.DefaultTaskNetworkStatusQueueSize
	}
	if options.timeout <= 0 {
		options.timeout = aghconfig.DefaultTaskNetworkStatusTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())
	observer := &taskStatusProjectionObserver{
		tasks: tasks, prefs: prefs, availability: availability, publisher: publisher,
		logger: options.logger, now: options.now, ctx: ctx, cancel: cancel,
		queue: make(chan taskpkg.EventRecord, options.queueSize), timeout: options.timeout,
	}
	observer.start()
	return observer
}

func (o *taskStatusProjectionObserver) OnTaskEvent(_ context.Context, record taskpkg.EventRecord) {
	if o == nil || !taskStatusProjectionEvent(record.Event.EventType) {
		return
	}
	select {
	case <-o.ctx.Done():
		return
	case o.queue <- record:
	default:
		o.logger.Warn("daemon: dropped task status projection because queue is full",
			"event_id", record.Event.ID, "task_id", record.Event.TaskID,
			"run_id", record.Event.RunID, "event_type", record.Event.EventType)
	}
}

func (o *taskStatusProjectionObserver) shutdown() {
	if o == nil {
		return
	}
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
}

func (o *taskStatusProjectionObserver) start() {
	o.wg.Go(func() {
		for {
			select {
			case <-o.ctx.Done():
				return
			case record := <-o.queue:
				o.process(record)
			}
		}
	})
}

func (o *taskStatusProjectionObserver) process(record taskpkg.EventRecord) {
	ctx, cancel := context.WithTimeout(o.ctx, o.timeout)
	defer cancel()
	if err := o.processWithContext(ctx, record); err != nil {
		o.logger.Warn("daemon: task status projection failed", "error", err,
			"event_id", record.Event.ID, "task_id", record.Event.TaskID,
			"run_id", record.Event.RunID, "event_type", record.Event.EventType)
	}
}

func (o *taskStatusProjectionObserver) processWithContext(
	ctx context.Context,
	record taskpkg.EventRecord,
) error {
	event := record.Event
	taskID := strings.TrimSpace(event.TaskID)
	if taskID == "" {
		return nil
	}
	taskRecord, err := o.tasks.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task for status projection: %w", err)
	}
	projection := taskStatusProjection{
		EventID: strings.TrimSpace(event.ID), EventType: taskStatusProjectionEventType(event.EventType),
		TaskID: taskID, RunID: strings.TrimSpace(event.RunID), TaskStatus: taskRecord.Status,
		NetworkParticipation: participation.LocalSpec(), ProjectedAt: o.now().UTC(),
		ConversationAvailability: taskConversationNotLinked,
	}
	if projection.RunID != "" {
		run, loadErr := o.tasks.GetTaskRun(ctx, projection.RunID)
		if loadErr != nil {
			return fmt.Errorf("load task run for status projection: %w", loadErr)
		}
		projection.RunStatus = run.Status
		projection.NetworkParticipation = run.NetworkSpecSnapshot()
		if strings.TrimSpace(run.DesignationGroupID) != "" && taskpkg.IsTerminalRunStatus(run.Status) {
			rollup, rollupErr := o.persistDesignationRollup(ctx, event, run)
			if rollupErr != nil {
				return rollupErr
			}
			projection.DesignationRollup = &rollup
		}
	}
	origin, ok, err := o.originForTask(ctx, taskID)
	if err != nil {
		return err
	}
	if ok {
		projection.ThreadOrigin = &origin
		availability, availabilityErr := o.availability.GetNetworkAvailability(ctx)
		if availabilityErr != nil {
			projection.ConversationAvailability = taskConversationAvailabilityError
			return fmt.Errorf("load network availability for task status projection: %w", availabilityErr)
		}
		if availability.Enabled {
			projection.ConversationAvailable = true
			projection.ConversationAvailability = taskConversationAvailable
		} else {
			projection.ConversationAvailability = taskConversationNetworkDisabled
		}
	}
	o.publisher.PublishTaskStatusProjection(ctx, projection)
	return nil
}

func (o *taskStatusProjectionObserver) originForTask(
	ctx context.Context,
	taskID string,
) (storepkg.NetworkTaskThreadOrigin, bool, error) {
	origins, err := o.prefs.ListNetworkTaskThreadOrigins(
		ctx,
		storepkg.NetworkTaskThreadOriginQuery{TaskID: strings.TrimSpace(taskID), Limit: 1},
	)
	if err != nil {
		return storepkg.NetworkTaskThreadOrigin{}, false, fmt.Errorf("load network task thread origin: %w", err)
	}
	if len(origins) == 0 {
		return storepkg.NetworkTaskThreadOrigin{}, false, nil
	}
	return origins[0], true, nil
}

type taskDesignationRollupStatus struct {
	DesignationGroupID string         `json:"designation_group_id"`
	TaskID             string         `json:"task_id"`
	RunIDs             []string       `json:"run_ids"`
	Total              int            `json:"total"`
	TerminalCount      int            `json:"terminal_count"`
	Completed          int            `json:"completed"`
	Failed             int            `json:"failed"`
	Canceled           int            `json:"canceled"`
	Running            int            `json:"running"`
	Queued             int            `json:"queued"`
	NeedsAttention     int            `json:"needs_attention"`
	Statuses           map[string]int `json:"statuses"`
	Complete           bool           `json:"complete"`
	UpdatedAt          string         `json:"updated_at"`
	LatestEvent        string         `json:"latest_event,omitempty"`
	LatestEventID      string         `json:"latest_event_id,omitempty"`
}

func (o *taskStatusProjectionObserver) persistDesignationRollup(
	ctx context.Context,
	event taskpkg.Event,
	run taskpkg.Run,
) (taskDesignationRollupStatus, error) {
	rollup, err := o.designationRollup(ctx, event, run)
	if err != nil {
		return taskDesignationRollupStatus{}, err
	}
	encoded, err := json.Marshal(rollup)
	if err != nil {
		return taskDesignationRollupStatus{}, fmt.Errorf("marshal task designation rollup: %w", err)
	}
	if err := o.prefs.PutTaskDesignationRollup(ctx, storepkg.TaskDesignationRollup{
		DesignationGroupID: rollup.DesignationGroupID, TaskID: rollup.TaskID,
		SummaryJSON: encoded, CreatedAt: o.now().UTC(),
	}); err != nil {
		return taskDesignationRollupStatus{}, fmt.Errorf("persist task designation rollup: %w", err)
	}
	return rollup, nil
}

func (o *taskStatusProjectionObserver) designationRollup(
	ctx context.Context,
	event taskpkg.Event,
	run taskpkg.Run,
) (taskDesignationRollupStatus, error) {
	groupID := strings.TrimSpace(run.DesignationGroupID)
	runs, err := o.tasks.ListTaskRuns(ctx, taskpkg.RunQuery{
		TaskID: strings.TrimSpace(run.TaskID), DesignationGroupID: groupID,
	})
	if err != nil {
		return taskDesignationRollupStatus{}, fmt.Errorf("list designated task runs: %w", err)
	}
	rollup := taskDesignationRollupStatus{
		DesignationGroupID: groupID, TaskID: strings.TrimSpace(run.TaskID),
		RunIDs: make([]string, 0, len(runs)), Statuses: make(map[string]int),
		UpdatedAt:   o.now().UTC().Format(time.RFC3339Nano),
		LatestEvent: strings.TrimSpace(event.EventType), LatestEventID: strings.TrimSpace(event.ID),
	}
	for _, item := range runs {
		if runID := strings.TrimSpace(item.ID); runID != "" {
			rollup.RunIDs = append(rollup.RunIDs, runID)
		}
		status := item.Status.Normalize()
		rollup.Statuses[status.String()]++
		switch status {
		case taskpkg.TaskRunStatusCompleted:
			rollup.Completed++
			rollup.TerminalCount++
		case taskpkg.TaskRunStatusFailed:
			rollup.Failed++
			rollup.TerminalCount++
		case taskpkg.TaskRunStatusCanceled:
			rollup.Canceled++
			rollup.TerminalCount++
		case taskpkg.TaskRunStatusQueued:
			rollup.Queued++
		case taskpkg.TaskRunStatusNeedsAttention:
			rollup.NeedsAttention++
		default:
			rollup.Running++
		}
	}
	slices.Sort(rollup.RunIDs)
	rollup.Total = len(rollup.RunIDs)
	rollup.Complete = rollup.Total > 0 && rollup.TerminalCount == rollup.Total
	return rollup, nil
}

func taskStatusProjectionEvent(eventType string) bool {
	return taskStatusProjectionEventType(eventType) != ""
}

func taskStatusProjectionEventType(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case taskEventCanceled, taskEventRunStarted, taskEventRunCompleted,
		taskEventRunFailed, taskEventRunCanceled:
		return strings.TrimSpace(eventType)
	case taskHookRunStarted:
		return taskEventRunStarted
	case taskHookRunCompleted:
		return taskEventRunCompleted
	case taskHookRunFailed:
		return taskEventRunFailed
	case taskHookRunCanceled:
		return taskEventRunCanceled
	default:
		return ""
	}
}
