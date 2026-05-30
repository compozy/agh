package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	eventspkg "github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/heartbeat"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	harnessSummaryDetachedCompleted       = eventspkg.HarnessDetachedRunCompleted
	harnessSummarySyntheticReentryEmitted = eventspkg.HarnessSyntheticReentryEmitted
	harnessSummarySyntheticReentryDropped = eventspkg.HarnessSyntheticReentryDropped

	harnessReentryShutdownFinalizationTimeout = time.Second

	harnessTaskEventRunCompleted = eventspkg.TaskRunCompleted
	harnessTaskEventRunFailed    = eventspkg.TaskRunFailed
	harnessTaskEventRunCanceled  = eventspkg.TaskRunCanceled

	harnessReentryOutcomeEmitted = "emitted"
	harnessReentryOutcomeSilent  = "silent"
	harnessReentryOutcomeDropped = "dropped"

	harnessReentryReasonCompleted            = "task_run_completed"
	harnessReentryReasonFailed               = "task_run_failed"
	harnessReentryReasonCanceled             = "task_run_canceled"
	harnessReentryReasonPolicySilent         = "policy_silent"
	harnessReentryReasonTargetMissing        = "target_missing"
	harnessReentryReasonTargetInactivePrefix = "target_inactive"
	harnessReentryReasonDispatchFailed       = "synthetic_dispatch_failed"
	harnessReentryReasonEventMissing         = "synthetic_event_missing"
	harnessReentryReasonAlreadyRecorded      = "synthetic_event_already_recorded"
)

type harnessReentryStore interface {
	taskStore
	store.EventSummaryStore
}

type harnessReentrySessionManager interface {
	Status(ctx context.Context, id string) (*session.Info, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	PromptSynthetic(ctx context.Context, id string, opts session.SyntheticPromptOpts) (<-chan acp.AgentEvent, error)
}

type harnessWakeTargetSnapshot struct {
	SessionID   string
	AgentName   string
	Type        session.Type
	State       session.State
	WorkspaceID string
	Channel     string
	Missing     bool
}

type harnessReentryDecision struct {
	outcome          string
	reason           string
	target           harnessWakeTargetSnapshot
	syntheticMessage string
	syntheticMeta    acp.PromptSyntheticMeta
}

type harnessSyntheticWake struct {
	taskID            string
	runID             string
	targetSessionID   string
	targetAgentName   string
	targetWorkspaceID string
	reason            string
	summary           string
	completedAt       time.Time
	completionSeq     int64
	syntheticMessage  string
	syntheticMeta     acp.PromptSyntheticMeta
}

type harnessWakeQueue struct {
	dispatching bool
	items       []harnessSyntheticWake
}

type recoveredDetachedHarnessRun struct {
	run           taskpkg.Run
	completionSeq int64
	completedAt   time.Time
}

type harnessReentryBridge struct {
	ctx           context.Context
	cancel        context.CancelFunc
	workers       sync.WaitGroup
	resolver      *HarnessContextResolver
	recorder      *harnessLifecycleRecorder
	store         harnessReentryStore
	sessions      harnessReentrySessionManager
	heartbeatWake heartbeat.WakeService
	logger        *slog.Logger

	events chan taskpkg.EventRecord
	rescan chan struct{}

	processingMu sync.Mutex
	processing   map[string]struct{}

	queueMu sync.Mutex
	queues  map[string]*harnessWakeQueue
}

var _ taskpkg.EventObserver = (*harnessReentryBridge)(nil)

func newHarnessReentryBridge(
	ctx context.Context,
	resolver *HarnessContextResolver,
	recorder *harnessLifecycleRecorder,
	store harnessReentryStore,
	sessions harnessReentrySessionManager,
	logger *slog.Logger,
	options ...harnessReentryOption,
) (*harnessReentryBridge, error) {
	if ctx == nil {
		return nil, errors.New("daemon: harness reentry bridge context is required")
	}
	if resolver == nil {
		return nil, errors.New("daemon: harness reentry bridge requires a harness resolver")
	}
	if store == nil {
		return nil, errors.New("daemon: harness reentry bridge requires a task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: harness reentry bridge requires a session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}

	bridgeCtx, cancel := context.WithCancel(ctx)
	bridge := &harnessReentryBridge{
		ctx:        bridgeCtx,
		cancel:     cancel,
		resolver:   resolver,
		recorder:   recorder,
		store:      store,
		sessions:   sessions,
		logger:     logger,
		events:     make(chan taskpkg.EventRecord, 256),
		rescan:     make(chan struct{}, 1),
		processing: make(map[string]struct{}),
		queues:     make(map[string]*harnessWakeQueue),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(bridge); err != nil {
			cancel()
			return nil, err
		}
	}
	bridge.startWorker(bridge.run)
	return bridge, nil
}

type harnessReentryOption func(*harnessReentryBridge) error

func withHarnessHeartbeatWake(
	store any,
	sessions harnessReentrySessionManager,
	config aghconfig.HeartbeatConfig,
) harnessReentryOption {
	return func(bridge *harnessReentryBridge) error {
		if bridge == nil || !config.Enabled {
			return nil
		}
		wakeStore, ok := store.(heartbeat.WakeStore)
		if !ok {
			return nil
		}
		healthReader, ok := sessions.(heartbeat.SessionHealthReader)
		if !ok {
			return nil
		}
		service, err := heartbeat.NewManagedWakeService(wakeStore, healthReader, bridge, config)
		if err != nil {
			return fmt.Errorf("daemon: create harness heartbeat wake service: %w", err)
		}
		bridge.heartbeatWake = service
		return nil
	}
}

func (b *harnessReentryBridge) shutdown() {
	if b == nil || b.cancel == nil {
		return
	}
	b.cancel()
	b.workers.Wait()
}

func (b *harnessReentryBridge) startWorker(worker func()) {
	b.workers.Go(func() {
		worker()
	})
}

func (b *harnessReentryBridge) OnTaskEvent(_ context.Context, record taskpkg.EventRecord) {
	if b == nil || !isDetachedHarnessTerminalTaskEvent(record.Event.EventType) {
		return
	}

	select {
	case <-b.ctx.Done():
		return
	case b.events <- record:
		return
	default:
	}

	b.logger.Warn(
		"daemon: detached harness reentry queue saturated; scheduling recovery rescan",
		"task_id", record.Event.TaskID,
		"run_id", record.Event.RunID,
		"event_type", record.Event.EventType,
	)
	b.requestRescan()
}

func (b *harnessReentryBridge) run() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case record := <-b.events:
			b.processTaskEvent(record)
		case <-b.rescan:
			if err := b.recoverPendingRuns(b.operationContext()); err != nil {
				b.logger.Error("daemon: recover detached harness runs after queue saturation", "error", err)
			}
		}
	}
}

func (b *harnessReentryBridge) recover(ctx context.Context) error {
	return b.recoverPendingRuns(ctx)
}

func (b *harnessReentryBridge) recoverPendingRuns(ctx context.Context) error {
	if b == nil {
		return errors.New("daemon: harness reentry bridge is required")
	}
	if ctx == nil {
		return errors.New("daemon: harness reentry recovery context is required")
	}

	recovered, err := b.loadRecoveredDetachedHarnessRuns(ctx)
	if err != nil {
		return err
	}

	for i := range recovered {
		item := &recovered[i]
		if err := b.processTerminalRun(
			item.run.TaskID,
			item.run.ID,
			item.completionSeq,
			item.completedAt,
		); err != nil {
			return err
		}
	}

	return nil
}

func (b *harnessReentryBridge) loadRecoveredDetachedHarnessRuns(
	ctx context.Context,
) ([]recoveredDetachedHarnessRun, error) {
	runs, err := b.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{
		taskpkg.TaskRunStatusCompleted,
		taskpkg.TaskRunStatusFailed,
		taskpkg.TaskRunStatusCanceled,
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: list detached terminal runs for reentry recovery: %w", err)
	}

	recovered := make([]recoveredDetachedHarnessRun, 0, len(runs))
	for _, run := range runs {
		metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
		if err != nil {
			return nil, err
		}
		if !ok || detachedHarnessReentryProcessed(metadata.Reentry) {
			continue
		}
		sequence, timestamp, lookupErr := b.latestDetachedTerminalSequence(ctx, run.TaskID, run.ID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if timestamp.IsZero() {
			timestamp = run.EndedAt
		}
		recovered = append(recovered, recoveredDetachedHarnessRun{
			run:           run,
			completionSeq: sequence,
			completedAt:   timestamp,
		})
	}

	sort.SliceStable(recovered, func(i, j int) bool {
		if !recovered[i].completedAt.Equal(recovered[j].completedAt) {
			return recovered[i].completedAt.Before(recovered[j].completedAt)
		}
		if recovered[i].completionSeq != recovered[j].completionSeq {
			return recovered[i].completionSeq < recovered[j].completionSeq
		}
		return recovered[i].run.ID < recovered[j].run.ID
	})

	return recovered, nil
}

func (b *harnessReentryBridge) latestDetachedTerminalSequence(
	ctx context.Context,
	taskID string,
	runID string,
) (int64, time.Time, error) {
	records, err := b.store.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{TaskID: strings.TrimSpace(taskID)})
	if err != nil {
		return 0, time.Time{}, err
	}

	var matched *taskpkg.EventRecord
	for i := range records {
		record := records[i]
		if strings.TrimSpace(record.Event.RunID) != strings.TrimSpace(runID) {
			continue
		}
		if !isDetachedHarnessTerminalTaskEvent(record.Event.EventType) {
			continue
		}
		if matched == nil || matched.Sequence < record.Sequence {
			item := record
			matched = &item
		}
	}
	if matched == nil {
		return 0, time.Time{}, nil
	}
	return matched.Sequence, matched.Event.Timestamp, nil
}

func (b *harnessReentryBridge) processTaskEvent(record taskpkg.EventRecord) {
	if err := b.processTerminalRun(
		record.Event.TaskID,
		record.Event.RunID,
		record.Sequence,
		record.Event.Timestamp,
	); err != nil {
		b.logger.Error(
			"daemon: process detached harness terminal run",
			"task_id", record.Event.TaskID,
			"run_id", record.Event.RunID,
			"event_type", record.Event.EventType,
			"error", err,
		)
	}
}

func (b *harnessReentryBridge) processTerminalRun(
	taskID string,
	runID string,
	completionSequence int64,
	completedAt time.Time,
) error {
	if b == nil {
		return errors.New("daemon: harness reentry bridge is required")
	}

	opCtx := b.operationContext()
	run, err := b.store.GetTaskRun(opCtx, strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	if !isDetachedHarnessTerminalRun(run.Status.Normalize()) {
		return nil
	}

	metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if detachedHarnessReentryProcessed(metadata.Reentry) {
		return nil
	}
	if !b.claimProcessing(run.ID) {
		return nil
	}

	task, err := b.store.GetTask(opCtx, strings.TrimSpace(taskID))
	if err != nil {
		b.releaseProcessing(run.ID)
		return err
	}

	if completedAt.IsZero() {
		completedAt = run.EndedAt
	}
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}

	target := b.resolveWakeTargetSnapshot(metadata)
	agentName := b.resolveSummaryAgentName(target, metadata)
	b.writeEventSummary(
		target.SessionID,
		agentName,
		harnessSummaryDetachedCompleted,
		detachedCompletionSummary(task.ID, run, metadata),
		completedAt,
	)

	decision := b.evaluateDecision(task, run, metadata, target)
	return b.applyWakeDecision(
		task,
		run,
		target,
		agentName,
		completionSequence,
		completedAt,
		decision,
	)
}

func (b *harnessReentryBridge) applyWakeDecision(
	task taskpkg.Task,
	run taskpkg.Run,
	target harnessWakeTargetSnapshot,
	agentName string,
	completionSequence int64,
	completedAt time.Time,
	decision harnessReentryDecision,
) error {
	switch decision.outcome {
	case harnessReentryOutcomeSilent:
		b.finalizeRunOutcome(
			run.ID,
			target.SessionID,
			agentName,
			harnessReentryOutcomeSilent,
			decision.reason,
		)
		return nil
	case harnessReentryOutcomeDropped:
		b.finalizeRunOutcome(
			run.ID,
			target.SessionID,
			agentName,
			harnessReentryOutcomeDropped,
			decision.reason,
		)
		return nil
	}

	existing, err := b.syntheticEventExists(target.SessionID, run.ID)
	if err != nil {
		b.releaseProcessing(run.ID)
		return err
	}
	if existing {
		b.finalizeRunOutcome(
			run.ID,
			target.SessionID,
			agentName,
			harnessReentryOutcomeEmitted,
			harnessReentryReasonAlreadyRecorded,
		)
		return nil
	}
	if target.Missing {
		b.finalizeRunOutcome(
			run.ID,
			target.SessionID,
			agentName,
			harnessReentryOutcomeDropped,
			harnessReentryReasonTargetMissing,
		)
		return nil
	}
	if target.State != session.StateActive {
		b.finalizeRunOutcome(
			run.ID,
			target.SessionID,
			agentName,
			harnessReentryOutcomeDropped,
			inactiveTargetReason(target.State),
		)
		return nil
	}

	b.enqueueWake(harnessSyntheticWake{
		taskID:            task.ID,
		runID:             run.ID,
		targetSessionID:   target.SessionID,
		targetAgentName:   agentName,
		targetWorkspaceID: target.WorkspaceID,
		reason:            decision.reason,
		summary:           decision.syntheticMeta.Summary,
		completedAt:       completedAt,
		completionSeq:     completionSequence,
		syntheticMessage:  decision.syntheticMessage,
		syntheticMeta:     decision.syntheticMeta,
	})
	return nil
}

func (b *harnessReentryBridge) resolveWakeTargetSnapshot(
	metadata detachedHarnessRunMetadata,
) harnessWakeTargetSnapshot {
	target := harnessWakeTargetSnapshot{
		SessionID:   strings.TrimSpace(metadata.WakeTarget.SessionID),
		Type:        session.Type(strings.TrimSpace(metadata.WakeTarget.SessionType)),
		WorkspaceID: strings.TrimSpace(metadata.WakeTarget.WorkspaceID),
		Channel:     strings.TrimSpace(metadata.WakeTarget.Channel),
		Missing:     true,
	}

	info, err := b.sessions.Status(b.operationContext(), target.SessionID)
	switch {
	case err == nil && info != nil:
		target.AgentName = strings.TrimSpace(info.AgentName)
		target.Type = info.Type
		target.State = info.State
		if workspaceID := strings.TrimSpace(info.WorkspaceID); workspaceID != "" {
			target.WorkspaceID = workspaceID
		}
		if channel := strings.TrimSpace(info.Channel); channel != "" {
			target.Channel = channel
		}
		target.Missing = false
	case errors.Is(err, session.ErrSessionNotFound):
		target.Missing = true
	default:
		target.Missing = true
	}

	return target
}

func (b *harnessReentryBridge) resolveSummaryAgentName(
	target harnessWakeTargetSnapshot,
	metadata detachedHarnessRunMetadata,
) string {
	if agentName := strings.TrimSpace(target.AgentName); agentName != "" {
		return agentName
	}
	if ownerID := strings.TrimSpace(metadata.OwnerSessionID); ownerID != "" {
		info, err := b.sessions.Status(b.operationContext(), ownerID)
		if err == nil && info != nil && strings.TrimSpace(info.AgentName) != "" {
			return strings.TrimSpace(info.AgentName)
		}
	}
	return harnessSummaryDefaultAgentName
}

func (b *harnessReentryBridge) evaluateDecision(
	task taskpkg.Task,
	run taskpkg.Run,
	metadata detachedHarnessRunMetadata,
	target harnessWakeTargetSnapshot,
) harnessReentryDecision {
	reason, trigger := syntheticReasonForTerminalRun(run.Status.Normalize())
	summary := detachedHarnessSummary(metadata.Summary)
	input := HarnessResolutionInput{
		Surface: ResolutionSurfaceTurn,
		Session: HarnessSessionInput{
			Type:        target.Type,
			Channel:     target.Channel,
			WorkspaceID: target.WorkspaceID,
		},
		Turn: HarnessTurnRequest{
			Source: session.TurnSourceSynthetic,
			Synthetic: &SyntheticTurnMetadata{
				Reason:      reason,
				Trigger:     trigger,
				SourceTask:  task.ID,
				SourceRunID: run.ID,
			},
			Detached: &DetachedRunMetadata{
				TaskID:    task.ID,
				TaskRunID: run.ID,
			},
		},
	}

	resolved, err := b.resolver.Resolve(input)
	if err != nil || resolved.Policy.ReentryMode != ReentryModeSynthetic {
		if err == nil && b.recorder != nil {
			b.recorder.RecordSyntheticContextResolved(
				b.ctx,
				target.SessionID,
				b.resolveSummaryAgentName(target, metadata),
				resolved,
				run.EndedAt,
			)
		}
		return harnessReentryDecision{
			outcome: harnessReentryOutcomeSilent,
			reason:  harnessReentryReasonPolicySilent,
			target:  target,
		}
	}
	if b.recorder != nil {
		b.recorder.RecordSyntheticContextResolved(
			b.ctx,
			target.SessionID,
			b.resolveSummaryAgentName(target, metadata),
			resolved,
			run.EndedAt,
		)
	}

	return harnessReentryDecision{
		outcome:          harnessReentryOutcomeEmitted,
		reason:           reason,
		target:           target,
		syntheticMessage: buildDetachedHarnessSyntheticMessage(task, run, summary),
		syntheticMeta: acp.PromptSyntheticMeta{
			TaskID:         task.ID,
			TaskRunID:      run.ID,
			ClaimTokenHash: strings.TrimSpace(run.ClaimTokenHash),
			Reason:         reason,
			Summary:        summary,
		},
	}
}

func (b *harnessReentryBridge) enqueueWake(item harnessSyntheticWake) {
	b.queueMu.Lock()
	queue := b.queues[item.targetSessionID]
	if queue == nil {
		queue = &harnessWakeQueue{}
		b.queues[item.targetSessionID] = queue
	}
	queue.items = insertSyntheticWake(queue.items, item)
	shouldStart := !queue.dispatching
	if shouldStart {
		queue.dispatching = true
	}
	b.queueMu.Unlock()

	if shouldStart {
		targetSessionID := item.targetSessionID
		b.startWorker(func() {
			b.drainWakeQueue(targetSessionID)
		})
	}
}

func insertSyntheticWake(items []harnessSyntheticWake, item harnessSyntheticWake) []harnessSyntheticWake {
	index := sort.Search(len(items), func(i int) bool {
		return compareSyntheticWake(item, items[i]) < 0
	})
	items = append(items, harnessSyntheticWake{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func compareSyntheticWake(left harnessSyntheticWake, right harnessSyntheticWake) int {
	if !left.completedAt.Equal(right.completedAt) {
		if left.completedAt.Before(right.completedAt) {
			return -1
		}
		return 1
	}
	if left.completionSeq != right.completionSeq {
		if left.completionSeq < right.completionSeq {
			return -1
		}
		return 1
	}
	return strings.Compare(left.runID, right.runID)
}

func (b *harnessReentryBridge) drainWakeQueue(sessionID string) {
	for {
		select {
		case <-b.ctx.Done():
			b.discardWakeQueue(sessionID)
			return
		default:
		}
		item, ok := b.nextWake(sessionID)
		if !ok {
			return
		}
		b.dispatchWake(item)
	}
}

func (b *harnessReentryBridge) discardWakeQueue(sessionID string) {
	b.queueMu.Lock()
	defer b.queueMu.Unlock()

	queue := b.queues[sessionID]
	if queue == nil {
		return
	}
	queue.dispatching = false
	queue.items = nil
	delete(b.queues, sessionID)
}

func (b *harnessReentryBridge) nextWake(sessionID string) (harnessSyntheticWake, bool) {
	b.queueMu.Lock()
	defer b.queueMu.Unlock()

	queue := b.queues[sessionID]
	if queue == nil || len(queue.items) == 0 {
		if queue != nil {
			queue.dispatching = false
			delete(b.queues, sessionID)
		}
		return harnessSyntheticWake{}, false
	}

	item := queue.items[0]
	if len(queue.items) == 1 {
		queue.items = nil
	} else {
		queue.items = append([]harnessSyntheticWake(nil), queue.items[1:]...)
	}
	return item, true
}

func (b *harnessReentryBridge) dispatchWake(item harnessSyntheticWake) {
	existing, err := b.syntheticEventExists(item.targetSessionID, item.runID)
	if err == nil && existing {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeEmitted,
			harnessReentryReasonAlreadyRecorded,
		)
		return
	}
	if b.dispatchHeartbeatWake(item) {
		return
	}

	eventsCh, promptErr := b.sessions.PromptSynthetic(b.ctx, item.targetSessionID, session.SyntheticPromptOpts{
		Message:                 item.syntheticMessage,
		InterruptIfAgentWaiting: true,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:         item.syntheticMeta.TaskID,
			TaskRunID:      item.syntheticMeta.TaskRunID,
			ClaimTokenHash: item.syntheticMeta.ClaimTokenHash,
			Reason:         item.syntheticMeta.Reason,
			Summary:        item.syntheticMeta.Summary,
		},
	})
	if promptErr != nil {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			classifySyntheticPromptError(promptErr),
		)
		return
	}
	if eventsCh == nil {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			harnessReentryReasonEventMissing,
		)
		return
	}

	b.startWorker(func() {
		b.awaitSyntheticWake(item, eventsCh)
	})
}

func (b *harnessReentryBridge) dispatchHeartbeatWake(item harnessSyntheticWake) bool {
	if b == nil || b.heartbeatWake == nil {
		return false
	}
	if strings.TrimSpace(item.targetWorkspaceID) == "" || strings.TrimSpace(item.targetAgentName) == "" {
		return false
	}
	decision, err := b.heartbeatWake.Wake(b.ctx, heartbeat.WakeRequest{
		WorkspaceID: strings.TrimSpace(item.targetWorkspaceID),
		AgentName:   strings.TrimSpace(item.targetAgentName),
		SessionID:   strings.TrimSpace(item.targetSessionID),
		Source:      heartbeat.WakeSourceHarnessReentry,
		SyntheticCorrelation: heartbeat.WakeSyntheticCorrelation{
			TaskID:               item.syntheticMeta.TaskID,
			TaskRunID:            item.syntheticMeta.TaskRunID,
			WorkflowID:           item.syntheticMeta.WorkflowID,
			ClaimTokenHash:       item.syntheticMeta.ClaimTokenHash,
			CoordinatorSessionID: item.syntheticMeta.CoordinatorSessionID,
		},
	})
	if err != nil {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			harnessReentryReasonDispatchFailed,
		)
		return true
	}
	if decision.Result == heartbeat.WakeResultSkipped &&
		decision.Reason == heartbeat.WakeReasonHeartbeatNoPolicy {
		return false
	}
	switch decision.Result {
	case heartbeat.WakeResultSent:
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeEmitted,
			string(decision.Reason),
		)
	case heartbeat.WakeResultSkipped:
		if decision.Reason == heartbeat.WakeReasonSessionPromptActive ||
			decision.Reason == heartbeat.WakeReasonSessionPromptRace {
			return false
		}
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			string(decision.Reason),
		)
	case heartbeat.WakeResultFailed:
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			harnessReentryReasonDispatchFailed,
		)
	default:
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			string(decision.Reason),
		)
	}
	return true
}

func (b *harnessReentryBridge) PromptHeartbeatWake(
	ctx context.Context,
	req heartbeat.SyntheticWakePromptRequest,
) (heartbeat.SyntheticWakePromptResult, error) {
	if b == nil || b.sessions == nil {
		return heartbeat.SyntheticWakePromptResult{}, errors.New("daemon: harness heartbeat prompter requires sessions")
	}
	events, err := b.sessions.PromptSynthetic(ctx, req.SessionID, session.SyntheticPromptOpts{
		Message: req.Message,
		TurnID:  req.TurnID,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:               req.SyntheticCorrelation.TaskID,
			TaskRunID:            req.SyntheticCorrelation.TaskRunID,
			WorkflowID:           req.SyntheticCorrelation.WorkflowID,
			ClaimTokenHash:       req.SyntheticCorrelation.ClaimTokenHash,
			CoordinatorSessionID: req.SyntheticCorrelation.CoordinatorSessionID,
			Reason:               heartbeat.SyntheticReasonHeartbeatWake,
			Summary:              req.Summary,
			WakeEventID:          req.WakeEventID,
			PolicySnapshotID:     req.PolicySnapshotID,
			PolicyDigest:         req.PolicyDigest,
			ConfigDigest:         req.ConfigDigest,
		},
		SkipIfBusy: true,
	})
	if err != nil {
		if errors.Is(err, session.ErrPromptInProgress) {
			return heartbeat.SyntheticWakePromptResult{}, heartbeat.ErrSyntheticPromptBusy
		}
		return heartbeat.SyntheticWakePromptResult{}, err
	}
	b.drainHeartbeatWakeEvents(req.SessionID, req.WakeEventID, events)
	return heartbeat.SyntheticWakePromptResult{SyntheticPromptID: req.TurnID}, nil
}

func (b *harnessReentryBridge) drainHeartbeatWakeEvents(
	sessionID string,
	wakeEventID string,
	events <-chan acp.AgentEvent,
) {
	if b == nil || events == nil {
		return
	}
	b.startWorker(func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if event.Type == acp.EventTypeError {
					b.logger.Warn(
						"daemon: heartbeat harness wake agent error",
						"session_id", sessionID,
						"wake_event_id", wakeEventID,
					)
				}
			}
		}
	})
}

func (b *harnessReentryBridge) awaitSyntheticWake(
	item harnessSyntheticWake,
	events <-chan acp.AgentEvent,
) {
	sawError := false
Loop:
	for {
		select {
		case <-b.ctx.Done():
			sawError = true
			break Loop
		case event, ok := <-events:
			if !ok {
				break Loop
			}
			if event.Type == acp.EventTypeError {
				sawError = true
			}
		}
	}

	existing, err := b.syntheticEventExists(item.targetSessionID, item.runID)
	if err == nil && existing {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeEmitted,
			item.reason,
		)
		return
	}
	if sawError || err != nil {
		b.finalizeRunOutcome(
			item.runID,
			item.targetSessionID,
			item.targetAgentName,
			harnessReentryOutcomeDropped,
			harnessReentryReasonDispatchFailed,
		)
		return
	}

	b.finalizeRunOutcome(
		item.runID,
		item.targetSessionID,
		item.targetAgentName,
		harnessReentryOutcomeDropped,
		harnessReentryReasonEventMissing,
	)
}

func (b *harnessReentryBridge) syntheticEventExists(sessionID string, runID string) (bool, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(runID) == "" {
		return false, nil
	}

	events, err := b.sessions.Events(
		b.operationContext(),
		sessionID,
		store.EventQuery{Type: acp.EventTypeSyntheticReentry},
	)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, event := range events {
		var payload struct {
			Synthetic *acp.PromptSyntheticMeta `json:"synthetic,omitempty"`
		}
		if err := json.Unmarshal([]byte(event.Content), &payload); err != nil {
			continue
		}
		if payload.Synthetic == nil {
			continue
		}
		if strings.TrimSpace(payload.Synthetic.TaskRunID) == strings.TrimSpace(runID) {
			return true, nil
		}
	}
	return false, nil
}

func (b *harnessReentryBridge) finalizeRunOutcome(
	runID string,
	sessionID string,
	agentName string,
	outcome string,
	reason string,
) {
	defer b.releaseProcessing(runID)

	opCtx, cancel := b.finalizationContext()
	defer cancel()

	run, err := b.store.GetTaskRun(opCtx, runID)
	if err != nil {
		b.logger.Error("daemon: load detached harness run for finalization", "run_id", runID, "error", err)
		return
	}
	metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
	if err != nil {
		b.logger.Error("daemon: decode detached harness run metadata for finalization", "run_id", runID, "error", err)
		return
	}
	if !ok {
		return
	}

	now := time.Now().UTC()
	metadata.Reentry = &detachedHarnessReentry{
		Outcome:     strings.TrimSpace(outcome),
		Reason:      strings.TrimSpace(reason),
		ProcessedAt: now,
	}
	raw, err := marshalDetachedHarnessMetadata(metadata)
	if err != nil {
		b.logger.Error("daemon: marshal detached harness finalization metadata", "run_id", runID, "error", err)
		return
	}

	run.Metadata = raw
	if err := b.store.UpdateTaskRun(opCtx, run); err != nil {
		b.logger.Error("daemon: persist detached harness finalization", "run_id", runID, "error", err)
		return
	}

	switch outcome {
	case harnessReentryOutcomeEmitted:
		b.writeEventSummaryWithContext(
			opCtx,
			sessionID,
			agentName,
			harnessSummarySyntheticReentryEmitted,
			syntheticOutcomeSummary(run.TaskID, run.ID, outcome, reason),
			now,
		)
	case harnessReentryOutcomeSilent, harnessReentryOutcomeDropped:
		b.writeEventSummaryWithContext(
			opCtx,
			sessionID,
			agentName,
			harnessSummarySyntheticReentryDropped,
			syntheticOutcomeSummary(run.TaskID, run.ID, outcome, reason),
			now,
		)
	}
}

func (b *harnessReentryBridge) writeEventSummary(
	sessionID string,
	agentName string,
	eventType string,
	summary string,
	timestamp time.Time,
) {
	b.writeEventSummaryWithContext(b.operationContext(), sessionID, agentName, eventType, summary, timestamp)
}

func (b *harnessReentryBridge) writeEventSummaryWithContext(
	ctx context.Context,
	sessionID string,
	agentName string,
	eventType string,
	summary string,
	timestamp time.Time,
) {
	targetSessionID := strings.TrimSpace(sessionID)
	if targetSessionID == "" {
		return
	}
	targetAgentName := strings.TrimSpace(agentName)
	if targetAgentName == "" {
		targetAgentName = harnessSummaryDefaultAgentName
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	summaryPayload := store.EventSummary{
		SessionID: targetSessionID,
		Type:      strings.TrimSpace(eventType),
		AgentName: targetAgentName,
		Summary:   strings.TrimSpace(summary),
		Timestamp: timestamp,
	}
	if b.sessions != nil {
		info, err := b.sessions.Status(ctx, targetSessionID)
		if err == nil && info != nil {
			summaryPayload = harnessEventSummaryWithLineage(summaryPayload, info.Lineage)
		}
	}
	if err := b.store.WriteEventSummary(ctx, summaryPayload); err != nil {
		b.logger.Warn(
			"daemon: write detached harness event summary failed",
			"session_id", targetSessionID,
			"event_type", eventType,
			"error", err,
		)
	}
}

func (b *harnessReentryBridge) operationContext() context.Context {
	if b == nil || b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}

func (b *harnessReentryBridge) finalizationContext() (context.Context, context.CancelFunc) {
	if b == nil || b.ctx == nil {
		return context.WithTimeout(context.Background(), harnessReentryShutdownFinalizationTimeout)
	}
	if b.ctx.Err() == nil {
		return b.ctx, func() {}
	}
	return context.WithTimeout(context.Background(), harnessReentryShutdownFinalizationTimeout)
}

func (b *harnessReentryBridge) requestRescan() {
	if b == nil {
		return
	}

	select {
	case <-b.ctx.Done():
		return
	case b.rescan <- struct{}{}:
	default:
	}
}

func (b *harnessReentryBridge) claimProcessing(runID string) bool {
	b.processingMu.Lock()
	defer b.processingMu.Unlock()
	if _, ok := b.processing[strings.TrimSpace(runID)]; ok {
		return false
	}
	b.processing[strings.TrimSpace(runID)] = struct{}{}
	return true
}

func (b *harnessReentryBridge) releaseProcessing(runID string) {
	b.processingMu.Lock()
	defer b.processingMu.Unlock()
	delete(b.processing, strings.TrimSpace(runID))
}

func detachedHarnessReentryProcessed(state *detachedHarnessReentry) bool {
	return state != nil && !state.ProcessedAt.IsZero()
}

func isDetachedHarnessTerminalTaskEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case harnessTaskEventRunCompleted, harnessTaskEventRunFailed, harnessTaskEventRunCanceled:
		return true
	default:
		return false
	}
}

func isDetachedHarnessTerminalRun(status taskpkg.RunStatus) bool {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusCompleted, taskpkg.TaskRunStatusFailed, taskpkg.TaskRunStatusCanceled:
		return true
	default:
		return false
	}
}

func syntheticReasonForTerminalRun(status taskpkg.RunStatus) (string, string) {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusFailed:
		return harnessReentryReasonFailed, harnessTaskEventRunFailed
	case taskpkg.TaskRunStatusCanceled:
		return harnessReentryReasonCanceled, harnessTaskEventRunCanceled
	default:
		return harnessReentryReasonCompleted, harnessTaskEventRunCompleted
	}
}

func buildDetachedHarnessSyntheticMessage(task taskpkg.Task, run taskpkg.Run, summary string) string {
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusFailed:
		return fmt.Sprintf(
			"Detached harness work failed for %q (task %s, run %s): %s",
			summary,
			task.ID,
			run.ID,
			strings.TrimSpace(run.Error),
		)
	case taskpkg.TaskRunStatusCanceled:
		return fmt.Sprintf(
			"Detached harness work was canceled for %q (task %s, run %s).",
			summary,
			task.ID,
			run.ID,
		)
	default:
		return fmt.Sprintf(
			"Detached harness work completed for %q (task %s, run %s).",
			summary,
			task.ID,
			run.ID,
		)
	}
}

func detachedCompletionSummary(taskID string, run taskpkg.Run, metadata detachedHarnessRunMetadata) string {
	return fmt.Sprintf(
		"task=%s run=%s status=%s target_session=%s summary=%q",
		strings.TrimSpace(taskID),
		strings.TrimSpace(run.ID),
		run.Status.Normalize(),
		strings.TrimSpace(metadata.WakeTarget.SessionID),
		detachedHarnessSummary(metadata.Summary),
	)
}

func syntheticOutcomeSummary(taskID string, runID string, outcome string, reason string) string {
	return fmt.Sprintf(
		"task=%s run=%s disposition=%s reason=%s",
		strings.TrimSpace(taskID),
		strings.TrimSpace(runID),
		strings.TrimSpace(outcome),
		strings.TrimSpace(reason),
	)
}

func inactiveTargetReason(state session.State) string {
	trimmed := strings.TrimSpace(string(state))
	if trimmed == "" {
		return harnessReentryReasonTargetInactivePrefix
	}
	return harnessReentryReasonTargetInactivePrefix + ":" + trimmed
}

func classifySyntheticPromptError(err error) string {
	switch {
	case errors.Is(err, session.ErrSessionNotFound):
		return harnessReentryReasonTargetMissing
	case errors.Is(err, session.ErrSessionNotActive):
		return harnessReentryReasonTargetInactivePrefix
	default:
		return harnessReentryReasonDispatchFailed
	}
}
