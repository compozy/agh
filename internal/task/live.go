package task

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store"
)

const taskStreamBufferSize = 256

const (
	sessionEventTypeToolCall   = "tool_call"
	sessionEventTypeToolResult = "tool_result"
)

type taskLiveContext struct {
	record    Task
	reference Reference
	runsByID  map[string]Run
}

type taskStreamSubscriber struct {
	id        uint64
	taskID    string
	actor     ActorContext
	deliver   chan StreamEvent
	done      chan struct{}
	out       chan StreamEvent
	closeOnce sync.Once
}

func newTaskStreamSubscriber(id uint64, taskID string, actor ActorContext) *taskStreamSubscriber {
	return &taskStreamSubscriber{
		id:      id,
		taskID:  taskID,
		actor:   actor,
		deliver: make(chan StreamEvent, taskStreamBufferSize),
		done:    make(chan struct{}),
		out:     make(chan StreamEvent),
	}
}

func (s *taskStreamSubscriber) enqueue(event StreamEvent) bool {
	select {
	case <-s.done:
		return false
	case s.deliver <- event:
		return true
	default:
		return false
	}
}

func (s *taskStreamSubscriber) stop() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

func (m *Service) Timeline(
	ctx context.Context,
	taskID string,
	query TimelineQuery,
	actor ActorContext,
) ([]TimelineItem, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	if err := query.Validate("task_timeline_query"); err != nil {
		return nil, err
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return nil, ErrValidation
	}
	if _, err := m.loadAuthorizedTask(ctx, m.store, trimmedTaskID, actor); err != nil {
		return nil, err
	}

	records, err := m.store.ListTaskEventRecords(ctx, EventRecordQuery{
		TaskID:        trimmedTaskID,
		AfterSequence: query.AfterSequence,
		Limit:         query.Limit,
	})
	if err != nil {
		return nil, err
	}
	visible, err := m.filterAuthorizedEventRecords(ctx, actor, records)
	if err != nil {
		return nil, err
	}
	return m.timelineItemsFromRecords(ctx, visible)
}

func (m *Service) Stream(
	ctx context.Context,
	taskID string,
	query StreamQuery,
	actor ActorContext,
) (<-chan StreamEvent, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}
	if err := query.Validate("task_stream_query"); err != nil {
		return nil, err
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return nil, ErrValidation
	}
	if _, err := m.loadAuthorizedTask(ctx, m.store, trimmedTaskID, actor); err != nil {
		return nil, err
	}

	subscriber := m.registerTaskSubscriber(trimmedTaskID, actor)
	backlog, err := m.streamBacklog(ctx, trimmedTaskID, query.AfterSequence, actor)
	if err != nil {
		m.unregisterTaskSubscriber(subscriber.id)
		subscriber.stop()
		close(subscriber.out)
		return nil, err
	}

	go m.runTaskStreamSubscriber(ctx, subscriber, query.AfterSequence, backlog)
	return subscriber.out, nil
}

func (m *Service) Tree(ctx context.Context, taskID string, actor ActorContext) (*TreeView, error) {
	if err := requireReadAuthority(actor); err != nil {
		return nil, err
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return nil, ErrValidation
	}
	tree, err := m.collectTaskTree(ctx, trimmedTaskID)
	if err != nil {
		return nil, err
	}
	if len(tree) == 0 {
		return nil, ErrTaskNotFound
	}
	if err := m.authorizeTaskResource(ctx, actor, tree[0]); err != nil {
		return nil, err
	}
	tree = m.filterAuthorizedTaskTree(ctx, actor, tree)

	nodes, err := m.buildTreeNodes(ctx, tree, actor)
	if err != nil {
		return nil, err
	}
	view := splitTreeView(trimmedTaskID, nodes)
	return &view, nil
}

func (m *Service) buildTreeNodes(
	ctx context.Context,
	tree []Task,
	actor ActorContext,
) ([]TreeNode, error) {
	depthByID := make(map[string]int, len(tree))
	depthByID[tree[0].ID] = 0
	childCountByID := make(map[string]int, len(tree))
	for _, record := range tree[1:] {
		childCountByID[record.ParentTaskID]++
	}

	nodes := make([]TreeNode, 0, len(tree))
	for _, record := range tree {
		node, err := m.buildTreeNode(ctx, record, depthByID, childCountByID, actor)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth
		}
		if !nodes[i].LastActivityAt.Equal(nodes[j].LastActivityAt) {
			return nodes[i].LastActivityAt.After(nodes[j].LastActivityAt)
		}
		return nodes[i].Task.ID < nodes[j].Task.ID
	})
	return nodes, nil
}

func (m *Service) buildTreeNode(
	ctx context.Context,
	record Task,
	depthByID map[string]int,
	childCountByID map[string]int,
	actor ActorContext,
) (TreeNode, error) {
	if strings.TrimSpace(record.ParentTaskID) != "" {
		depthByID[record.ID] = depthByID[record.ParentTaskID] + 1
	}

	dependencies, err := m.store.ListDependencies(ctx, record.ID)
	if err != nil {
		return TreeNode{}, err
	}
	runs, err := m.store.ListTaskRuns(ctx, RunQuery{TaskID: record.ID})
	if err != nil {
		return TreeNode{}, err
	}
	visibleRuns, err := m.filterAuthorizedRuns(ctx, actor, record, runs)
	if err != nil {
		return TreeNode{}, err
	}
	events, err := m.store.ListTaskEvents(ctx, EventQuery{TaskID: record.ID, Limit: 1})
	if err != nil {
		return TreeNode{}, err
	}
	summary, err := m.enrichTaskSummaryFromState(ctx, record, childCountByID[record.ID], dependencies, runs, events)
	if err != nil {
		return TreeNode{}, err
	}

	return TreeNode{
		Task:           taskReferenceFromTask(record, summary.Status),
		ParentTaskID:   record.ParentTaskID,
		Depth:          depthByID[record.ID],
		ChildCount:     childCountByID[record.ID],
		ActiveRun:      activeRunSummary(visibleRuns, record.MaxAttempts),
		LastActivityAt: summary.LastActivityAt,
	}, nil
}

func splitTreeView(rootTaskID string, nodes []TreeNode) TreeView {
	rootIndex := 0
	for idx := range nodes {
		if nodes[idx].Task.ID == rootTaskID {
			rootIndex = idx
			break
		}
	}

	root := nodes[rootIndex]
	descendants := make([]TreeNode, 0, len(nodes)-1)
	for idx, node := range nodes {
		if idx == rootIndex {
			continue
		}
		descendants = append(descendants, node)
	}
	return TreeView{Root: root, Descendants: descendants}
}

func (m *Service) streamBacklog(
	ctx context.Context,
	rootTaskID string,
	afterSequence int64,
	actor ActorContext,
) ([]StreamEvent, error) {
	records, err := m.listTaskTreeEventRecords(ctx, rootTaskID, afterSequence, actor)
	if err != nil {
		return nil, err
	}
	items, err := m.timelineItemsFromRecords(ctx, records)
	if err != nil {
		return nil, err
	}

	events := make([]StreamEvent, 0, len(items))
	for _, item := range items {
		events = append(events, StreamEvent{
			Sequence: item.Sequence,
			Type:     item.EventType,
			Timeline: item,
		})
	}
	return events, nil
}

func (m *Service) listTaskTreeEventRecords(
	ctx context.Context,
	rootTaskID string,
	afterSequence int64,
	actor ActorContext,
) ([]EventRecord, error) {
	tree, err := m.collectTaskTree(ctx, rootTaskID)
	if err != nil {
		return nil, err
	}
	tree = m.filterAuthorizedTaskTree(ctx, actor, tree)

	records := make([]EventRecord, 0)
	for _, record := range tree {
		taskRecords, err := m.store.ListTaskEventRecords(ctx, EventRecordQuery{
			TaskID:        record.ID,
			AfterSequence: afterSequence,
		})
		if err != nil {
			return nil, err
		}
		records = append(records, taskRecords...)
	}

	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].Event.Timestamp.Equal(records[j].Event.Timestamp) {
			return records[i].Event.Timestamp.Before(records[j].Event.Timestamp)
		}
		if records[i].Sequence != records[j].Sequence {
			return records[i].Sequence < records[j].Sequence
		}
		return records[i].Event.ID < records[j].Event.ID
	})
	return m.filterAuthorizedEventRecords(ctx, actor, records)
}

func (m *Service) filterAuthorizedTaskTree(
	ctx context.Context,
	actor ActorContext,
	tree []Task,
) []Task {
	if isTaskOperator(actor) {
		return tree
	}
	hidden := make(map[string]struct{})
	visible := make([]Task, 0, len(tree))
	for _, record := range tree {
		if _, parentHidden := hidden[strings.TrimSpace(record.ParentTaskID)]; parentHidden {
			hidden[record.ID] = struct{}{}
			continue
		}
		if err := m.authorizeTaskResource(ctx, actor, record); err != nil {
			hidden[record.ID] = struct{}{}
			continue
		}
		visible = append(visible, record)
	}
	return visible
}

func (m *Service) filterAuthorizedEventRecords(
	ctx context.Context,
	actor ActorContext,
	records []EventRecord,
) ([]EventRecord, error) {
	if isTaskOperator(actor) {
		return records, nil
	}
	visible := make([]EventRecord, 0, len(records))
	for _, record := range records {
		allowed, err := m.authorizeEventRecord(ctx, actor, record)
		if err != nil {
			return nil, err
		}
		if allowed {
			visible = append(visible, record)
		}
	}
	return visible, nil
}

func (m *Service) authorizeEventRecord(
	ctx context.Context,
	actor ActorContext,
	record EventRecord,
) (bool, error) {
	taskRecord, err := m.store.GetTask(ctx, record.Event.TaskID)
	if err != nil {
		return false, err
	}
	if err := m.authorizeTaskResource(ctx, actor, taskRecord); err != nil {
		return false, nil
	}
	if strings.TrimSpace(record.Event.RunID) == "" {
		return true, nil
	}
	run, err := m.store.GetTaskRun(ctx, record.Event.RunID)
	if err != nil {
		return false, err
	}
	if err := m.runReadAuthorizer.AuthorizeRunRead(ctx, actor, run, &taskRecord); err != nil {
		return false, nil
	}
	return true, nil
}

func (m *Service) timelineItemsFromRecords(
	ctx context.Context,
	records []EventRecord,
) ([]TimelineItem, error) {
	cache := make(map[string]*taskLiveContext)
	items := make([]TimelineItem, 0, len(records))
	for _, record := range records {
		contextForTask, err := m.loadTaskLiveContext(ctx, record.Event.TaskID, cache)
		if err != nil {
			return nil, err
		}

		var runSummary *RunSummary
		if strings.TrimSpace(record.Event.RunID) != "" {
			if run, ok := contextForTask.runsByID[record.Event.RunID]; ok {
				runSummary = runSummaryFromRun(run, contextForTask.record.MaxAttempts)
			}
		}

		items = append(items, TimelineItem{
			Sequence:  record.Sequence,
			EventID:   record.Event.ID,
			Task:      contextForTask.reference,
			Run:       runSummary,
			EventType: record.Event.EventType,
			Actor:     record.Event.Actor,
			Origin:    record.Event.Origin,
			Payload:   cloneRawJSON(record.Event.Payload),
			Timestamp: record.Event.Timestamp,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].Timestamp.Equal(items[j].Timestamp) {
			return items[i].Timestamp.Before(items[j].Timestamp)
		}
		if items[i].Sequence != items[j].Sequence {
			return items[i].Sequence < items[j].Sequence
		}
		return items[i].EventID < items[j].EventID
	})
	return items, nil
}

func runSummaryFromRun(run Run, maxAttempts int) *RunSummary {
	summary := &RunSummary{
		ID:                           run.ID,
		TaskID:                       run.TaskID,
		Status:                       run.Status,
		Attempt:                      int(run.Attempt),
		RecoveryCount:                int(run.RecoveryCount),
		PreviousRunID:                run.PreviousRunID,
		FailureKind:                  run.FailureKind,
		MaxAttempts:                  maxAttempts,
		SessionID:                    run.SessionID,
		ClaimedBy:                    cloneActorIdentity(run.ClaimedBy),
		ClaimTokenHash:               run.ClaimTokenHash,
		LeaseUntil:                   run.LeaseUntil,
		HeartbeatAt:                  run.HeartbeatAt,
		ResolvedNetworkParticipation: participation.CloneSpec(run.NetworkSpecSnapshot()),
		DesignationGroupID:           run.DesignationGroupID,
		QueuedAt:                     run.QueuedAt,
		ClaimedAt:                    run.ClaimedAt,
		StartedAt:                    run.StartedAt,
		EndedAt:                      run.EndedAt,
		Error:                        run.Error,
	}
	ApplyRunDesignationSummary(summary, run)
	return summary
}

func baseRunSessionRef(sessionID string) *RunSessionRef {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return nil
	}
	return &RunSessionRef{SessionID: trimmed}
}

func (m *Service) bestEffortRunOperationalSummary(ctx context.Context, sessionID string) RunOperationalSummary {
	if m.runtimeViews == nil || strings.TrimSpace(sessionID) == "" {
		return RunOperationalSummary{}
	}

	events, eventsErr := m.runtimeViews.ListSessionEvents(ctx, sessionID, store.EventQuery{})
	stats, statsErr := m.runtimeViews.ListSessionTokenStats(ctx, sessionID)
	if eventsErr != nil && statsErr != nil {
		return RunOperationalSummary{}
	}

	summary := RunOperationalSummary{}
	if eventsErr == nil {
		summary = summarizeSessionEvents(events)
	}
	if statsErr == nil {
		mergeTokenStatsSummary(&summary, stats)
	}
	return summary
}

func summarizeSessionEvents(events []store.SessionEvent) RunOperationalSummary {
	sorted := append([]store.SessionEvent(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sequence != sorted[j].Sequence {
			return sorted[i].Sequence < sorted[j].Sequence
		}
		if !sorted[i].Timestamp.Equal(sorted[j].Timestamp) {
			return sorted[i].Timestamp.Before(sorted[j].Timestamp)
		}
		return sorted[i].ID < sorted[j].ID
	})

	summary := RunOperationalSummary{}
	toolCalls := make(map[string]struct{})
	for _, event := range sorted {
		if event.Timestamp.After(summary.LastActivityAt) {
			summary.LastActivityAt = event.Timestamp
			summary.LastEventType = event.Type
		}
		if event.Type != sessionEventTypeToolCall && event.Type != sessionEventTypeToolResult {
			continue
		}

		decoded := struct {
			ToolCallIDSnake string `json:"tool_call_id"`
			ToolCallIDCamel string `json:"toolCallId"`
		}{}
		if err := json.Unmarshal([]byte(event.Content), &decoded); err == nil {
			toolCallID := strings.TrimSpace(decoded.ToolCallIDSnake)
			if toolCallID == "" {
				toolCallID = strings.TrimSpace(decoded.ToolCallIDCamel)
			}
			if toolCallID != "" {
				toolCalls[toolCallID] = struct{}{}
				continue
			}
		}

		fallbackID := strings.TrimSpace(event.TurnID) + ":" + event.ID
		if fallbackID != ":" {
			toolCalls[fallbackID] = struct{}{}
			continue
		}
		toolCalls[event.ID] = struct{}{}
	}

	if len(toolCalls) > 0 {
		count := int64(len(toolCalls))
		summary.ToolCallCount = &count
	}
	return summary
}

func (m *Service) registerTaskSubscriber(taskID string, actor ActorContext) *taskStreamSubscriber {
	m.liveMu.Lock()
	defer m.liveMu.Unlock()

	m.nextSubscriberID++
	subscriber := newTaskStreamSubscriber(m.nextSubscriberID, taskID, actor)
	m.liveSubscribers[subscriber.id] = subscriber
	return subscriber
}

func (m *Service) unregisterTaskSubscriber(id uint64) {
	m.liveMu.Lock()
	defer m.liveMu.Unlock()
	delete(m.liveSubscribers, id)
}

func (m *Service) runTaskStreamSubscriber(
	ctx context.Context,
	subscriber *taskStreamSubscriber,
	afterSequence int64,
	backlog []StreamEvent,
) {
	defer func() {
		m.unregisterTaskSubscriber(subscriber.id)
		subscriber.stop()
		close(subscriber.out)
	}()

	nextSequence := afterSequence
	emit := func(event StreamEvent) bool {
		if event.Sequence <= nextSequence {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case subscriber.out <- event:
			nextSequence = event.Sequence
			return true
		}
	}

	for _, event := range backlog {
		if !emit(event) {
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-subscriber.done:
			return
		case event := <-subscriber.deliver:
			if !emit(event) {
				return
			}
		}
	}
}

func (m *Service) emitTaskLiveEventBestEffort(ctx context.Context, eventID string) {
	record, err := m.store.GetTaskEventRecord(ctx, eventID)
	if err != nil {
		return
	}
	m.emitTaskLiveRecordBestEffort(ctx, record)
}

func (m *Service) emitTaskLiveRecordBestEffort(ctx context.Context, record EventRecord) {
	roots, err := m.ancestorTaskIDs(ctx, record.Event.TaskID)
	if err != nil {
		return
	}

	subscribers := m.taskSubscribersForRoots(roots)
	eligible := make([]*taskStreamSubscriber, 0, len(subscribers))
	for _, subscriber := range subscribers {
		allowed, authorizeErr := m.authorizeEventRecord(ctx, subscriber.actor, record)
		if authorizeErr == nil && allowed {
			eligible = append(eligible, subscriber)
		}
	}
	if len(eligible) == 0 {
		return
	}
	items, err := m.timelineItemsFromRecords(ctx, []EventRecord{record})
	if err != nil || len(items) == 0 {
		return
	}
	event := StreamEvent{Sequence: items[0].Sequence, Type: items[0].EventType, Timeline: items[0]}
	for _, subscriber := range eligible {
		if subscriber.enqueue(event) {
			continue
		}
		m.unregisterTaskSubscriber(subscriber.id)
		subscriber.stop()
	}
}

func (m *Service) ancestorTaskIDs(ctx context.Context, taskID string) ([]string, error) {
	seen := make(map[string]struct{})
	ids := make([]string, 0, 4)

	currentID := strings.TrimSpace(taskID)
	for currentID != "" {
		if _, ok := seen[currentID]; ok {
			break
		}
		seen[currentID] = struct{}{}
		ids = append(ids, currentID)

		record, err := m.store.GetTask(ctx, currentID)
		if err != nil {
			return nil, err
		}
		currentID = strings.TrimSpace(record.ParentTaskID)
	}

	return ids, nil
}

func (m *Service) taskSubscribersForRoots(rootTaskIDs []string) []*taskStreamSubscriber {
	rootSet := make(map[string]struct{}, len(rootTaskIDs))
	for _, rootTaskID := range rootTaskIDs {
		rootSet[rootTaskID] = struct{}{}
	}

	m.liveMu.Lock()
	defer m.liveMu.Unlock()

	subscribers := make([]*taskStreamSubscriber, 0, len(rootSet))
	for _, subscriber := range m.liveSubscribers {
		if _, ok := rootSet[subscriber.taskID]; ok {
			subscribers = append(subscribers, subscriber)
		}
	}
	return subscribers
}
