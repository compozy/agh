package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

// ParseSessionEventQuery parses the shared session event query parameters.
func ParseSessionEventQuery(c *gin.Context) (store.EventQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventQuery{}, err
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventQuery{}, err
	}
	afterSequence, err := ParseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return store.EventQuery{}, err
	}

	return store.EventQuery{
		Type:          strings.TrimSpace(c.Query("type")),
		AgentName:     strings.TrimSpace(c.Query("agent_name")),
		TurnID:        strings.TrimSpace(c.Query("turn_id")),
		Since:         since,
		Limit:         limit,
		AfterSequence: afterSequence,
	}, nil
}

// ParseLogsQuery parses the shared logs query parameters.
func ParseLogsQuery(c *gin.Context) (store.EventSummaryQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	afterSequenceRaw, err := firstQueryAlias(c, "after_seq", "after_sequence")
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	afterSequence, err := ParseOptionalInt64(afterSequenceRaw)
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	workspaceID, err := firstQueryAlias(c, "workspace_id", "workspace")
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	runID, err := firstQueryAlias(c, "run", "run_id")
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	errorOnly, err := ParseOptionalBool(c.Query("error_only"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	actorKind, actorID, err := parseActorQuery(c)
	if err != nil {
		return store.EventSummaryQuery{}, err
	}

	return store.EventSummaryQuery{
		WorkspaceID:   strings.TrimSpace(workspaceID),
		SessionID:     strings.TrimSpace(c.Query("session_id")),
		AgentName:     strings.TrimSpace(c.Query("agent_name")),
		Type:          strings.TrimSpace(c.Query("type")),
		RunID:         strings.TrimSpace(runID),
		ActorKind:     actorKind,
		ActorID:       actorID,
		Provider:      strings.TrimSpace(c.Query("provider")),
		Outcome:       strings.TrimSpace(c.Query("outcome")),
		Component:     strings.TrimSpace(c.Query("component")),
		ErrorOnly:     errorOnly,
		AfterSequence: afterSequence,
		Since:         since,
		Limit:         limit,
	}, nil
}

func firstQueryAlias(c *gin.Context, names ...string) (string, error) {
	selected := ""
	selectedName := ""
	for _, name := range names {
		value := strings.TrimSpace(c.Query(name))
		if value == "" {
			continue
		}
		if selected != "" && value != selected {
			return "", fmt.Errorf("conflicting query values for %s, %s", selectedName, name)
		}
		selected = value
		selectedName = name
	}
	return selected, nil
}

func parseActorQuery(c *gin.Context) (string, string, error) {
	actorKind := strings.TrimSpace(c.Query("actor_kind"))
	actorID := strings.TrimSpace(c.Query("actor_id"))
	actor := strings.TrimSpace(c.Query("actor"))
	if actor == "" {
		return actorKind, actorID, nil
	}
	if actorKind != "" || actorID != "" {
		return "", "", fmt.Errorf("actor cannot be combined with actor_kind or actor_id")
	}
	parsedKind, parsedID, ok := strings.Cut(actor, ":")
	if !ok || strings.TrimSpace(parsedKind) == "" || strings.TrimSpace(parsedID) == "" {
		return "", "", fmt.Errorf("actor must use kind:id format")
	}
	return strings.TrimSpace(parsedKind), strings.TrimSpace(parsedID), nil
}

// ParseHookCatalogFilter parses the shared hook catalog query parameters.
func ParseHookCatalogFilter(c *gin.Context) (hookspkg.CatalogFilter, error) {
	filter := hookspkg.CatalogFilter{
		AgentName: strings.TrimSpace(c.Query("agent")),
	}

	if event := strings.TrimSpace(c.Query("event")); event != "" {
		parsed := hookspkg.HookEvent(event)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Event = parsed
	}

	if source := strings.TrimSpace(c.Query("source")); source != "" {
		var parsed hookspkg.HookSource
		if err := parsed.UnmarshalText([]byte(source)); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Source = &parsed
	}

	if mode := strings.TrimSpace(c.Query("mode")); mode != "" {
		parsed := hookspkg.HookMode(mode)
		if err := parsed.Validate(); err != nil {
			return hookspkg.CatalogFilter{}, err
		}
		filter.Mode = parsed
	}

	return filter, nil
}

// ParseHookRunsQuery parses the shared hook execution history query parameters.
func ParseHookRunsQuery(c *gin.Context) (store.HookRunQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.HookRunQuery{}, err
	}
	last, err := ParseOptionalInt(c.Query("last"))
	if err != nil {
		return store.HookRunQuery{}, err
	}

	query := store.HookRunQuery{
		SessionID: strings.TrimSpace(c.Query("session")),
		Event:     strings.TrimSpace(c.Query("event")),
		Since:     since,
		Limit:     last,
	}
	if outcome := strings.TrimSpace(c.Query("outcome")); outcome != "" {
		query.Outcome = hookspkg.HookRunOutcome(outcome)
		if err := query.Outcome.Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if event := query.Event; event != "" {
		if err := hookspkg.HookEvent(event).Validate(); err != nil {
			return store.HookRunQuery{}, err
		}
	}
	if err := query.Validate(); err != nil {
		return store.HookRunQuery{}, err
	}
	return query, nil
}

// ParseHookEventFilter parses the shared hook taxonomy query parameters.
func ParseHookEventFilter(c *gin.Context) (hookspkg.EventFilter, error) {
	syncOnly, err := ParseOptionalBool(c.Query("sync_only"))
	if err != nil {
		return hookspkg.EventFilter{}, err
	}

	filter := hookspkg.EventFilter{
		SyncOnly: syncOnly,
	}
	if family := strings.TrimSpace(c.Query("family")); family != "" {
		filter.Family = hookspkg.HookEventFamily(family)
		if err := filter.Family.Validate(); err != nil {
			return hookspkg.EventFilter{}, err
		}
	}
	return filter, nil
}

// ParseLogsCursor parses a Last-Event-ID cursor for logs streaming.
func ParseLogsCursor(raw string) (LogsCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return LogsCursor{}, nil
	}

	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return LogsCursor{}, fmt.Errorf("invalid Last-Event-ID %q", value)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return LogsCursor{}, fmt.Errorf("invalid Last-Event-ID timestamp %q: %w", parts[0], err)
	}

	cursor := LogsCursor{
		Timestamp: timestamp.UTC(),
	}

	cursorValue := strings.TrimSpace(parts[1])
	if cursorValue == "" {
		return cursor, nil
	}

	sequence, err := strconv.ParseInt(cursorValue, 10, 64)
	if err == nil && sequence > 0 {
		cursor.Sequence = sequence
		return cursor, nil
	}

	cursor.ID = cursorValue
	return cursor, nil
}

// ParseTaskListQuery parses the shared task-list query parameters.
func ParseTaskListQuery(c *gin.Context) (contract.TaskListQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return contract.TaskListQuery{}, NewTaskValidationError(err)
	}
	includeDrafts, err := ParseOptionalBool(c.Query("include_drafts"))
	if err != nil {
		return contract.TaskListQuery{}, NewTaskValidationError(err)
	}

	query := contract.TaskListQuery{
		Scope:          taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Workspace:      strings.TrimSpace(c.Query("workspace")),
		Status:         taskpkg.Status(strings.TrimSpace(c.Query("status"))).Normalize(),
		Priority:       taskpkg.Priority(strings.TrimSpace(c.Query("priority"))).Normalize(),
		IncludeDrafts:  includeDrafts,
		ApprovalState:  taskpkg.ApprovalState(strings.TrimSpace(c.Query("approval_state"))).Normalize(),
		OwnerKind:      taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:       strings.TrimSpace(c.Query("owner_ref")),
		ParentTaskID:   strings.TrimSpace(c.Query("parent_task_id")),
		NetworkChannel: strings.TrimSpace(c.Query("network_channel")),
		Query:          strings.TrimSpace(c.Query("query")),
		Limit:          limit,
	}
	if err := validateParsedTaskListQuery(query); err != nil {
		return contract.TaskListQuery{}, err
	}
	return query, nil
}

// ParseTaskRunListQuery parses the shared task-run list query parameters.
func ParseTaskRunListQuery(c *gin.Context) (contract.TaskRunListQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return contract.TaskRunListQuery{}, NewTaskValidationError(err)
	}

	return contract.TaskRunListQuery{
		Status:    taskpkg.RunStatus(strings.TrimSpace(c.Query("status"))).Normalize(),
		SessionID: strings.TrimSpace(c.Query("session_id")),
		Limit:     limit,
	}, nil
}

// ParseTaskTimelineQuery parses the shared task timeline query parameters.
func ParseTaskTimelineQuery(c *gin.Context) (contract.TaskTimelineQuery, error) {
	afterSequence, err := ParseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return contract.TaskTimelineQuery{}, NewTaskValidationError(err)
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return contract.TaskTimelineQuery{}, NewTaskValidationError(err)
	}

	return contract.TaskTimelineQuery{
		AfterSequence: afterSequence,
		Limit:         limit,
	}, nil
}

// ParseTaskStreamQuery parses the shared task stream query parameters.
func ParseTaskStreamQuery(c *gin.Context) (contract.TaskStreamQuery, error) {
	afterSequence, err := ParseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return contract.TaskStreamQuery{}, NewTaskValidationError(err)
	}

	return contract.TaskStreamQuery{AfterSequence: afterSequence}, nil
}

// ParseTaskDashboardQuery parses the shared task dashboard query parameters.
func ParseTaskDashboardQuery(c *gin.Context) (contract.TaskDashboardQuery, error) {
	query := contract.TaskDashboardQuery{
		Scope:          taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Workspace:      strings.TrimSpace(c.Query("workspace")),
		OwnerKind:      taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:       strings.TrimSpace(c.Query("owner_ref")),
		NetworkChannel: strings.TrimSpace(c.Query("network_channel")),
		OriginKind:     taskpkg.OriginKind(strings.TrimSpace(c.Query("origin_kind"))).Normalize(),
	}
	if err := validateParsedTaskDashboardQuery(query); err != nil {
		return contract.TaskDashboardQuery{}, err
	}
	return query, nil
}

// ParseTaskInboxQuery parses the shared task inbox query parameters.
func ParseTaskInboxQuery(c *gin.Context) (contract.TaskInboxQuery, error) {
	unread, err := ParseOptionalBool(c.Query("unread"))
	if err != nil {
		return contract.TaskInboxQuery{}, NewTaskValidationError(err)
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return contract.TaskInboxQuery{}, NewTaskValidationError(err)
	}

	query := contract.TaskInboxQuery{
		Scope:     taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Workspace: strings.TrimSpace(c.Query("workspace")),
		OwnerKind: taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:  strings.TrimSpace(c.Query("owner_ref")),
		Lane:      contract.TaskInboxLane(strings.TrimSpace(strings.ToLower(c.Query("lane")))),
		Unread:    unread,
		Query:     strings.TrimSpace(c.Query("query")),
		Limit:     limit,
	}
	if err := validateParsedTaskInboxQuery(query); err != nil {
		return contract.TaskInboxQuery{}, err
	}
	return query, nil
}

func validateParsedTaskListQuery(query contract.TaskListQuery) error {
	if err := validateOptionalTaskScope(query.Scope, "task_query.scope"); err != nil {
		return err
	}
	if err := validateOptionalTaskStatus(query.Status, "task_query.status"); err != nil {
		return err
	}
	if err := validateOptionalTaskPriority(query.Priority, "task_query.priority"); err != nil {
		return err
	}
	if err := validateOptionalTaskApprovalState(query.ApprovalState, "task_query.approval_state"); err != nil {
		return err
	}
	if err := validateOptionalTaskOwnerKind(query.OwnerKind, "task_query.owner_kind"); err != nil {
		return err
	}
	return nil
}

func validateParsedTaskDashboardQuery(query contract.TaskDashboardQuery) error {
	summaryQuery := observe.TaskSummaryQuery{
		Scope:      query.Scope,
		OwnerKind:  query.OwnerKind,
		OwnerRef:   query.OwnerRef,
		OriginKind: query.OriginKind,
	}
	if err := summaryQuery.Validate(); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateParsedTaskInboxQuery(query contract.TaskInboxQuery) error {
	summaryQuery := observe.TaskSummaryQuery{
		Scope:     query.Scope,
		OwnerKind: query.OwnerKind,
		OwnerRef:  query.OwnerRef,
	}
	if err := summaryQuery.Validate(); err != nil {
		return NewTaskValidationError(err)
	}
	if err := validateOptionalTaskInboxLane(query.Lane, "task_inbox_query.lane"); err != nil {
		return err
	}
	return nil
}

func validateOptionalTaskScope(scope taskpkg.Scope, path string) error {
	if scope.Normalize() == "" {
		return nil
	}
	if err := scope.Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateOptionalTaskStatus(status taskpkg.Status, path string) error {
	if status.Normalize() == "" {
		return nil
	}
	if err := status.Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateOptionalTaskPriority(priority taskpkg.Priority, path string) error {
	if priority.Normalize() == "" {
		return nil
	}
	if err := priority.Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateOptionalTaskApprovalState(state taskpkg.ApprovalState, path string) error {
	if state.Normalize() == "" {
		return nil
	}
	if err := state.Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateOptionalTaskOwnerKind(kind taskpkg.OwnerKind, path string) error {
	if kind.Normalize() == "" {
		return nil
	}
	if err := kind.Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func validateOptionalTaskInboxLane(lane contract.TaskInboxLane, path string) error {
	if observe.TaskInboxLane(lane).Normalize() == "" {
		return nil
	}
	if err := observe.TaskInboxLane(lane).Validate(path); err != nil {
		return NewTaskValidationError(err)
	}
	return nil
}

func (h *BaseHandlers) taskListDomainQuery(
	ctx context.Context,
	query contract.TaskListQuery,
) (taskpkg.Query, error) {
	domainQuery := taskpkg.Query{
		Scope:          query.Scope.Normalize(),
		Status:         query.Status.Normalize(),
		Priority:       query.Priority.Normalize(),
		ApprovalState:  query.ApprovalState.Normalize(),
		OwnerKind:      query.OwnerKind.Normalize(),
		OwnerRef:       strings.TrimSpace(query.OwnerRef),
		ParentTaskID:   strings.TrimSpace(query.ParentTaskID),
		NetworkChannel: strings.TrimSpace(query.NetworkChannel),
		Search:         strings.TrimSpace(query.Query),
		Limit:          query.Limit,
	}

	if workspaceRef := strings.TrimSpace(query.Workspace); workspaceRef != "" {
		if domainQuery.Scope.Normalize() == taskpkg.ScopeGlobal {
			return taskpkg.Query{}, taskpkg.ValidateScopeBinding(
				domainQuery.Scope,
				workspaceRef,
				"task_query",
				"workspace",
			)
		}
		workspaceID, err := h.lookupWorkspaceID(ctx, workspaceRef)
		if err != nil {
			return taskpkg.Query{}, err
		}
		domainQuery.WorkspaceID = workspaceID
	}

	if err := validateTaskChannel("task_query.network_channel", domainQuery.NetworkChannel); err != nil {
		return taskpkg.Query{}, err
	}
	if err := domainQuery.Validate("task_query"); err != nil {
		return taskpkg.Query{}, err
	}
	return domainQuery, nil
}

func (h *BaseHandlers) parseTaskListQuery(ctx context.Context, c *gin.Context) (taskpkg.Query, error) {
	query, err := ParseTaskListQuery(c)
	if err != nil {
		return taskpkg.Query{}, err
	}
	return h.taskListDomainQuery(ctx, query)
}

func taskRunListDomainQuery(query contract.TaskRunListQuery) (taskpkg.RunQuery, error) {
	domainQuery := taskpkg.RunQuery{
		Status:    query.Status.Normalize(),
		SessionID: strings.TrimSpace(query.SessionID),
		Limit:     query.Limit,
	}
	if err := domainQuery.Validate("task_run_query"); err != nil {
		return taskpkg.RunQuery{}, err
	}
	return domainQuery, nil
}

func parseTaskRunListQuery(c *gin.Context) (taskpkg.RunQuery, error) {
	query, err := ParseTaskRunListQuery(c)
	if err != nil {
		return taskpkg.RunQuery{}, err
	}
	return taskRunListDomainQuery(query)
}

func taskTimelineDomainQuery(query contract.TaskTimelineQuery) (taskpkg.TimelineQuery, error) {
	domainQuery := taskpkg.TimelineQuery{
		AfterSequence: query.AfterSequence,
		Limit:         query.Limit,
	}
	if err := domainQuery.Validate("task_timeline_query"); err != nil {
		return taskpkg.TimelineQuery{}, err
	}
	return domainQuery, nil
}

func (h *BaseHandlers) taskStreamDomainQuery(
	c *gin.Context,
	query contract.TaskStreamQuery,
) (taskpkg.StreamQuery, error) {
	domainQuery := taskpkg.StreamQuery{AfterSequence: query.AfterSequence}

	if rawLastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); rawLastEventID != "" {
		afterSequence, err := parseLastEventID(rawLastEventID, h.transportName())
		if err != nil {
			return taskpkg.StreamQuery{}, NewTaskValidationError(err)
		}
		domainQuery.AfterSequence = afterSequence
	}

	if err := domainQuery.Validate("task_stream_query"); err != nil {
		return taskpkg.StreamQuery{}, err
	}
	return domainQuery, nil
}

func (h *BaseHandlers) taskDashboardDomainQuery(
	ctx context.Context,
	query contract.TaskDashboardQuery,
) (observe.TaskDashboardQuery, error) {
	domainQuery := observe.TaskDashboardQuery{
		Scope:          query.Scope.Normalize(),
		OwnerKind:      query.OwnerKind.Normalize(),
		OwnerRef:       strings.TrimSpace(query.OwnerRef),
		NetworkChannel: strings.TrimSpace(query.NetworkChannel),
		OriginKind:     query.OriginKind.Normalize(),
	}

	if workspaceRef := strings.TrimSpace(query.Workspace); workspaceRef != "" {
		if err := taskpkg.ValidateScopeBinding(
			domainQuery.Scope,
			workspaceRef,
			"task_dashboard_query",
			"workspace",
		); err != nil {
			return observe.TaskDashboardQuery{}, err
		}
		if domainQuery.Scope.Normalize() == taskpkg.ScopeWorkspace {
			workspaceID, err := h.lookupWorkspaceID(ctx, workspaceRef)
			if err != nil {
				return observe.TaskDashboardQuery{}, err
			}
			domainQuery.WorkspaceID = workspaceID
		}
	}

	if err := validateTaskChannel("task_dashboard_query.network_channel", domainQuery.NetworkChannel); err != nil {
		return observe.TaskDashboardQuery{}, err
	}
	if err := domainQuery.Validate(); err != nil {
		return observe.TaskDashboardQuery{}, err
	}
	return domainQuery, nil
}

func (h *BaseHandlers) taskInboxDomainQuery(
	ctx context.Context,
	query contract.TaskInboxQuery,
) (observe.TaskInboxQuery, error) {
	domainQuery := observe.TaskInboxQuery{
		Scope:     query.Scope.Normalize(),
		OwnerKind: query.OwnerKind.Normalize(),
		OwnerRef:  strings.TrimSpace(query.OwnerRef),
		Lane:      observe.TaskInboxLane(query.Lane).Normalize(),
		Unread:    query.Unread,
		Search:    strings.TrimSpace(query.Query),
		Limit:     query.Limit,
	}

	if workspaceRef := strings.TrimSpace(query.Workspace); workspaceRef != "" {
		if err := taskpkg.ValidateScopeBinding(
			domainQuery.Scope,
			workspaceRef,
			"task_inbox_query",
			"workspace",
		); err != nil {
			return observe.TaskInboxQuery{}, err
		}
		if domainQuery.Scope.Normalize() == taskpkg.ScopeWorkspace {
			workspaceID, err := h.lookupWorkspaceID(ctx, workspaceRef)
			if err != nil {
				return observe.TaskInboxQuery{}, err
			}
			domainQuery.WorkspaceID = workspaceID
		}
	}

	if err := domainQuery.Validate(); err != nil {
		return observe.TaskInboxQuery{}, err
	}
	return domainQuery, nil
}

// ParseOptionalTime parses an optional RFC3339 or RFC3339Nano timestamp.
func ParseOptionalTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}

// ParseOptionalInt parses an optional integer query value.
func ParseOptionalInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

// ParseOptionalInt64 parses an optional 64-bit integer query value.
func ParseOptionalInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

// ParseOptionalBool parses an optional boolean query value.
func ParseOptionalBool(raw string) (bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q: %w", value, err)
	}
	return parsed, nil
}
