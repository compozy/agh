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
	"github.com/compozy/agh/internal/network"
	storepkg "github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	taskEventCanceled     = "task.canceled"
	taskEventRunStarted   = "task.run_started"
	taskEventRunCompleted = "task.run_completed"
	taskEventRunFailed    = "task.run_failed"
	taskEventRunCanceled  = "task.run_canceled"
)

type networkTaskStatusObserver struct {
	network networkRuntime
	tasks   taskStore
	prefs   storepkg.NetworkPreferenceStore
	logger  *slog.Logger
	now     func() time.Time
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	queue   chan taskpkg.EventRecord
	timeout time.Duration
}

var _ taskpkg.EventObserver = (*networkTaskStatusObserver)(nil)

func newNetworkTaskStatusObserver(
	networkRuntime networkRuntime,
	tasks taskStore,
	logger *slog.Logger,
	now func() time.Time,
	queueSize int,
	timeout time.Duration,
) *networkTaskStatusObserver {
	prefs, ok := tasks.(storepkg.NetworkPreferenceStore)
	if networkRuntime == nil || tasks == nil || !ok {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	if queueSize <= 0 {
		queueSize = aghconfig.DefaultTaskNetworkStatusQueueSize
	}
	if timeout <= 0 {
		timeout = aghconfig.DefaultTaskNetworkStatusTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())
	observer := &networkTaskStatusObserver{
		network: networkRuntime,
		tasks:   tasks,
		prefs:   prefs,
		logger:  logger,
		now:     now,
		ctx:     ctx,
		cancel:  cancel,
		queue:   make(chan taskpkg.EventRecord, queueSize),
		timeout: timeout,
	}
	observer.start()
	return observer
}

func (o *networkTaskStatusObserver) OnTaskEvent(_ context.Context, record taskpkg.EventRecord) {
	if o == nil || !networkTaskStatusEvent(record.Event.EventType) {
		return
	}
	select {
	case <-o.ctx.Done():
		return
	case o.queue <- record:
	default:
		o.logger.Warn(
			"daemon: dropped network task status event because queue is full",
			"event_id", record.Event.ID,
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
		)
	}
}

func (o *networkTaskStatusObserver) shutdown() {
	if o == nil {
		return
	}
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
}

func (o *networkTaskStatusObserver) start() {
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

func (o *networkTaskStatusObserver) process(record taskpkg.EventRecord) {
	ctx, cancel := context.WithTimeout(o.ctx, o.timeout)
	defer cancel()
	if err := o.processWithContext(ctx, record); err != nil {
		o.logger.Warn(
			"daemon: network task status observer failed",
			"error", err,
			"event_id", record.Event.ID,
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
		)
	}
}

func (o *networkTaskStatusObserver) processWithContext(
	ctx context.Context,
	record taskpkg.EventRecord,
) error {
	event := record.Event
	if strings.TrimSpace(event.TaskID) == "" {
		return nil
	}
	origin, ok, err := o.originForTask(ctx, event.TaskID)
	if err != nil || !ok {
		return err
	}
	if strings.TrimSpace(event.RunID) == "" {
		return o.sendTaskStatus(ctx, origin, event, nil)
	}
	run, err := o.tasks.GetTaskRun(ctx, event.RunID)
	if err != nil {
		return fmt.Errorf("load task run for network status: %w", err)
	}
	if strings.TrimSpace(run.DesignationGroupID) != "" {
		if taskpkg.IsTerminalRunStatus(run.Status) {
			return o.recomputeAndSendDesignationRollup(ctx, origin, event, run)
		}
		return nil
	}
	if err := o.sendTaskStatus(ctx, origin, event, &run); err != nil {
		return err
	}
	return nil
}

func (o *networkTaskStatusObserver) originForTask(
	ctx context.Context,
	taskID string,
) (storepkg.NetworkTaskThreadOrigin, bool, error) {
	origins, err := o.prefs.ListNetworkTaskThreadOrigins(
		ctx,
		storepkg.NetworkTaskThreadOriginQuery{TaskID: strings.TrimSpace(taskID), Limit: 1},
	)
	if err != nil {
		return storepkg.NetworkTaskThreadOrigin{}, false, fmt.Errorf(
			"load network task thread origin: %w",
			err,
		)
	}
	if len(origins) == 0 {
		return storepkg.NetworkTaskThreadOrigin{}, false, nil
	}
	return origins[0], true, nil
}

func (o *networkTaskStatusObserver) sendTaskStatus(
	ctx context.Context,
	origin storepkg.NetworkTaskThreadOrigin,
	event taskpkg.Event,
	run *taskpkg.Run,
) error {
	text := o.taskStatusText(ctx, event, run)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return o.sendThreadSay(ctx, origin, event, text, "task_status")
}

func (o *networkTaskStatusObserver) taskStatusText(
	ctx context.Context,
	event taskpkg.Event,
	run *taskpkg.Run,
) string {
	taskLabel := strings.TrimSpace(event.TaskID)
	if taskRecord, err := o.tasks.GetTask(ctx, event.TaskID); err == nil {
		taskLabel = quotedTaskLabel(taskRecord)
	}
	switch strings.TrimSpace(event.EventType) {
	case taskEventCanceled:
		return fmt.Sprintf("Task status: %s was canceled.", taskLabel)
	case taskEventRunStarted:
		if run == nil {
			return ""
		}
		return fmt.Sprintf("Task status: %s run %s started.", taskLabel, shortTaskRunID(run.ID))
	case taskEventRunCompleted:
		if run == nil {
			return ""
		}
		return fmt.Sprintf("Task status: %s run %s completed.", taskLabel, shortTaskRunID(run.ID))
	case taskEventRunFailed:
		if run == nil {
			return ""
		}
		return fmt.Sprintf("Task status: %s run %s failed.", taskLabel, shortTaskRunID(run.ID))
	case taskEventRunCanceled:
		if run == nil {
			return ""
		}
		return fmt.Sprintf("Task status: %s run %s was canceled.", taskLabel, shortTaskRunID(run.ID))
	default:
		return ""
	}
}

func (o *networkTaskStatusObserver) recomputeAndSendDesignationRollup(
	ctx context.Context,
	origin storepkg.NetworkTaskThreadOrigin,
	event taskpkg.Event,
	run taskpkg.Run,
) error {
	rollup, err := o.designationRollup(ctx, event, run)
	if err != nil {
		return err
	}
	wasComplete, err := o.previousDesignationRollupComplete(ctx, rollup.DesignationGroupID)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(rollup)
	if err != nil {
		return fmt.Errorf("marshal task designation rollup: %w", err)
	}
	if err := o.prefs.PutTaskDesignationRollup(ctx, storepkg.TaskDesignationRollup{
		DesignationGroupID: rollup.DesignationGroupID,
		TaskID:             rollup.TaskID,
		SummaryJSON:        encoded,
		CreatedAt:          o.now().UTC(),
	}); err != nil {
		return fmt.Errorf("persist task designation rollup: %w", err)
	}
	if !rollup.Complete {
		return nil
	}
	if wasComplete {
		return nil
	}
	return o.sendThreadSay(ctx, origin, event, designationRollupStatusText(rollup), "task_fanout_rollup")
}

func (o *networkTaskStatusObserver) previousDesignationRollupComplete(
	ctx context.Context,
	designationGroupID string,
) (bool, error) {
	rollups, err := o.prefs.ListTaskDesignationRollups(
		ctx,
		storepkg.TaskDesignationRollupQuery{
			DesignationGroupID: strings.TrimSpace(designationGroupID),
			Limit:              1,
		},
	)
	if err != nil {
		return false, fmt.Errorf("load previous task designation rollup: %w", err)
	}
	if len(rollups) == 0 || len(rollups[0].SummaryJSON) == 0 {
		return false, nil
	}
	var previous taskDesignationRollupStatus
	if err := json.Unmarshal(rollups[0].SummaryJSON, &previous); err != nil {
		return false, fmt.Errorf("decode previous task designation rollup: %w", err)
	}
	return previous.Complete, nil
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

func (o *networkTaskStatusObserver) designationRollup(
	ctx context.Context,
	event taskpkg.Event,
	run taskpkg.Run,
) (taskDesignationRollupStatus, error) {
	groupID := strings.TrimSpace(run.DesignationGroupID)
	runs, err := o.tasks.ListTaskRuns(ctx, taskpkg.RunQuery{
		TaskID:             strings.TrimSpace(run.TaskID),
		DesignationGroupID: groupID,
	})
	if err != nil {
		return taskDesignationRollupStatus{}, fmt.Errorf("list designated task runs: %w", err)
	}
	rollup := taskDesignationRollupStatus{
		DesignationGroupID: groupID,
		TaskID:             strings.TrimSpace(run.TaskID),
		RunIDs:             make([]string, 0, len(runs)),
		Statuses:           make(map[string]int),
		UpdatedAt:          o.now().UTC().Format(time.RFC3339Nano),
		LatestEvent:        strings.TrimSpace(event.EventType),
		LatestEventID:      strings.TrimSpace(event.ID),
	}
	for _, item := range runs {
		runID := strings.TrimSpace(item.ID)
		if runID != "" {
			rollup.RunIDs = append(rollup.RunIDs, runID)
		}
		status := item.Status.Normalize()
		statusKey := string(status)
		rollup.Statuses[statusKey]++
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

func (o *networkTaskStatusObserver) sendThreadSay(
	ctx context.Context,
	origin storepkg.NetworkTaskThreadOrigin,
	event taskpkg.Event,
	text string,
	intent string,
) error {
	body, err := json.Marshal(network.SayBody{
		Text:   strings.TrimSpace(text),
		Intent: strings.TrimSpace(intent),
	})
	if err != nil {
		return fmt.Errorf("marshal network task status body: %w", err)
	}
	surface := network.SurfaceThread
	threadID := strings.TrimSpace(origin.ThreadID)
	replyTo := strings.TrimSpace(origin.OriginMessageID)
	causationID := strings.TrimSpace(event.ID)
	_, err = o.network.SendFromRuntimePeer(ctx, network.RuntimeSendRequest{
		WorkspaceID: strings.TrimSpace(origin.WorkspaceID),
		Channel:     strings.TrimSpace(origin.Channel),
		Surface:     &surface,
		ThreadID:    &threadID,
		Kind:        network.KindSay,
		Body:        body,
		ReplyTo:     &replyTo,
		CausationID: &causationID,
	})
	if err != nil {
		return fmt.Errorf("send network task status: %w", err)
	}
	return nil
}

func networkTaskStatusEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case taskEventCanceled,
		taskEventRunStarted,
		taskEventRunCompleted,
		taskEventRunFailed,
		taskEventRunCanceled:
		return true
	default:
		return false
	}
}

func quotedTaskLabel(taskRecord taskpkg.Task) string {
	title := strings.TrimSpace(taskRecord.Title)
	if title == "" {
		return strings.TrimSpace(taskRecord.ID)
	}
	return fmt.Sprintf("%q", truncateStatusText(title, 96))
}

func shortTaskRunID(runID string) string {
	trimmed := strings.TrimSpace(runID)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:12]
}

func truncateStatusText(value string, maxBytes int) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if maxBytes <= 0 || len(trimmed) <= maxBytes {
		return trimmed
	}
	return truncateUTF8Bytes(trimmed, maxBytes) + "..."
}

func designationRollupStatusText(rollup taskDesignationRollupStatus) string {
	return fmt.Sprintf(
		"Task fan-out complete: %d completed, %d failed, %d canceled across %d designated runs.",
		rollup.Completed,
		rollup.Failed,
		rollup.Canceled,
		rollup.Total,
	)
}
