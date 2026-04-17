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

// ParseObserveEventQuery parses the shared observe query parameters.
func ParseObserveEventQuery(c *gin.Context) (store.EventSummaryQuery, error) {
	since, err := ParseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventSummaryQuery{}, err
	}

	return store.EventSummaryQuery{
		SessionID: strings.TrimSpace(c.Query("session_id")),
		AgentName: strings.TrimSpace(c.Query("agent_name")),
		Type:      strings.TrimSpace(c.Query("type")),
		Since:     since,
		Limit:     limit,
	}, nil
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

// ParseObserveCursor parses a Last-Event-ID cursor for observe streaming.
func ParseObserveCursor(raw string) (ObserveCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ObserveCursor{}, nil
	}

	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return ObserveCursor{}, fmt.Errorf("invalid Last-Event-ID %q", value)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ObserveCursor{}, fmt.Errorf("invalid Last-Event-ID timestamp %q: %w", parts[0], err)
	}

	cursor := ObserveCursor{
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

	return contract.TaskListQuery{
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
	}, nil
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
	return contract.TaskDashboardQuery{
		Scope:          taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Workspace:      strings.TrimSpace(c.Query("workspace")),
		OwnerKind:      taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:       strings.TrimSpace(c.Query("owner_ref")),
		NetworkChannel: strings.TrimSpace(c.Query("network_channel")),
		OriginKind:     taskpkg.OriginKind(strings.TrimSpace(c.Query("origin_kind"))).Normalize(),
	}, nil
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

	return contract.TaskInboxQuery{
		Scope:     taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Workspace: strings.TrimSpace(c.Query("workspace")),
		OwnerKind: taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:  strings.TrimSpace(c.Query("owner_ref")),
		Lane:      contract.TaskInboxLane(strings.TrimSpace(strings.ToLower(c.Query("lane")))),
		Unread:    unread,
		Query:     strings.TrimSpace(c.Query("query")),
		Limit:     limit,
	}, nil
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

	afterSequence, err := parseLastEventID(c.GetHeader("Last-Event-ID"), h.transportName())
	if err != nil {
		return taskpkg.StreamQuery{}, NewTaskValidationError(err)
	}
	if afterSequence > 0 {
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
