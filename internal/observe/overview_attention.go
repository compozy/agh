package observe

import (
	"context"
	"fmt"
	"sort"
	"strings"

	taskpkg "github.com/compozy/agh/internal/task"
)

func (o *Observer) overviewAttention(ctx context.Context, query OverviewQuery) (OverviewAttention, error) {
	page, err := o.QueryTaskInbox(ctx, TaskInboxQuery{
		Scope:       query.TaskScope,
		WorkspaceID: query.WorkspaceID,
		Limit:       overviewAttentionScan,
	}, query.Actor)
	if err != nil {
		return OverviewAttention{}, fmt.Errorf("observe: query attention inbox: %w", err)
	}

	needsAttention, err := o.registry.ListTasks(ctx, taskpkg.Query{
		Scope:       taskScopeFromCatalogScope(query.TaskScope),
		WorkspaceID: query.WorkspaceID,
		Status:      taskpkg.TaskStatusNeedsAttention,
		Limit:       overviewAttentionScan,
	})
	if err != nil {
		return OverviewAttention{}, fmt.Errorf("observe: list needs-attention tasks: %w", err)
	}

	attention := OverviewAttention{ByKind: map[string]int{}}
	seen := make(map[string]struct{})
	items := make([]OverviewAttentionItem, 0)

	for _, group := range page.Groups {
		switch group.Lane {
		case TaskInboxLaneApprovals:
			attention.ByKind[OverviewAttentionKindApproval] = group.Count
			for _, item := range group.Items {
				items = appendAttentionItem(items, seen, approvalAttentionItem(item))
			}
		case TaskInboxLaneFailedRuns:
			attention.ByKind[OverviewAttentionKindFailure] = group.Count
			for _, item := range group.Items {
				items = appendAttentionItem(items, seen, failureAttentionItem(item))
			}
		default:
		}
	}

	needsInputCount := 0
	for index := range needsAttention {
		item, ok := needsInputAttentionItem(&needsAttention[index])
		if !ok {
			continue
		}
		needsInputCount++
		items = appendAttentionItem(items, seen, item)
	}
	attention.ByKind[OverviewAttentionKindNeedsInput] = needsInputCount

	sort.SliceStable(items, func(left, right int) bool {
		return items[left].OccurredAt.After(items[right].OccurredAt)
	})
	if len(items) > overviewAttentionItemCap {
		items = items[:overviewAttentionItemCap]
	}

	attention.Items = items
	for _, count := range attention.ByKind {
		attention.Total += count
	}
	return attention, nil
}

func taskScopeFromCatalogScope(scope taskpkg.CatalogScope) taskpkg.Scope {
	switch scope {
	case taskpkg.CatalogScopeGlobal:
		return taskpkg.ScopeGlobal
	case taskpkg.CatalogScopeWorkspace:
		return taskpkg.ScopeWorkspace
	default:
		return ""
	}
}

func appendAttentionItem(
	items []OverviewAttentionItem,
	seen map[string]struct{},
	item OverviewAttentionItem,
) []OverviewAttentionItem {
	key := item.TaskID
	if key == "" {
		key = item.Kind + ":" + item.Title
	}
	if _, exists := seen[key]; exists {
		return items
	}
	seen[key] = struct{}{}
	return append(items, item)
}

func approvalAttentionItem(item taskpkg.InboxItem) OverviewAttentionItem {
	return OverviewAttentionItem{
		Kind:       OverviewAttentionKindApproval,
		Title:      item.Task.Title,
		TaskID:     item.Task.ID,
		OccurredAt: item.LatestActivityAt,
		Actions:    []string{OverviewActionApprove, OverviewActionReject, OverviewActionOpen},
	}
}

func failureAttentionItem(item taskpkg.InboxItem) OverviewAttentionItem {
	attention := OverviewAttentionItem{
		Kind:       OverviewAttentionKindFailure,
		Title:      item.Task.Title,
		TaskID:     item.Task.ID,
		OccurredAt: item.LatestActivityAt,
		Actions:    []string{OverviewActionOpen},
	}
	if item.Run != nil {
		attention.RunID = item.Run.ID
		attention.SessionID = item.Run.SessionID
		attention.Detail = strings.TrimSpace(item.Run.Error)
		if item.Run.ID != "" {
			attention.Actions = []string{OverviewActionRetry, OverviewActionOpen}
		}
	}
	return attention
}

func needsInputAttentionItem(summary *taskpkg.Summary) (OverviewAttentionItem, bool) {
	if summary == nil || summary.Status != taskpkg.TaskStatusNeedsAttention {
		return OverviewAttentionItem{}, false
	}
	item := OverviewAttentionItem{
		Kind:       OverviewAttentionKindNeedsInput,
		Title:      summary.Title,
		TaskID:     summary.ID,
		OccurredAt: summary.LastActivityAt,
		Actions:    []string{OverviewActionOpen},
	}
	if item.OccurredAt.IsZero() {
		item.OccurredAt = summary.UpdatedAt
	}
	if summary.NeedsAttention != nil {
		item.Detail = strings.TrimSpace(summary.NeedsAttention.Reason)
	}
	if summary.ActiveRun != nil {
		item.RunID = summary.ActiveRun.ID
		item.SessionID = summary.ActiveRun.SessionID
	}
	return item, true
}
