import { useCallback, useMemo } from "react";

import {
  type InboxLaneCount,
  applyInboxFilterChips,
  buildInboxFilterFields,
  inboxFiltersToChips,
} from "../lib/inbox-filters";
import {
  INBOX_GROUPS,
  INBOX_UI_LANES,
  type InboxGroupId,
  type InboxLaneFilterId,
  type InboxUiLane,
  backendLaneToUiLane,
  resolveInboxGroupId,
} from "../lib/inbox-grouping";
import type { TaskInboxItem, TaskInboxView, TaskPriority, TaskStatus } from "../types";

type LaneCount = InboxLaneCount;

interface UseTasksInboxViewArgs {
  inbox: TaskInboxView | null;
  laneFilter: InboxLaneFilterId;
  onLaneChange: (lane: InboxLaneFilterId) => void;
  statusFilter: TaskStatus | null;
  onStatusChange: (next: TaskStatus | null) => void;
  priorityFilter: TaskPriority | null;
  onPriorityChange: (next: TaskPriority | null) => void;
  unreadOnly: boolean;
}

function flattenItems(inbox: TaskInboxView | null): TaskInboxItem[] {
  if (!inbox?.groups) {
    return [];
  }
  return inbox.groups.flatMap(group => group.items ?? []);
}

function computeLaneCounts(items: TaskInboxItem[]): Map<InboxUiLane, LaneCount> {
  const counts = new Map<InboxUiLane, LaneCount>();
  for (const lane of INBOX_UI_LANES) {
    counts.set(lane.id, { count: 0, unread: 0 });
  }
  for (const item of items) {
    const laneId = backendLaneToUiLane(item.lane);
    const entry = counts.get(laneId);
    if (!entry) continue;
    entry.count += 1;
    if (!item.triage.read && !item.triage.dismissed) {
      entry.unread += 1;
    }
  }
  return counts;
}

function partitionByGroup(items: TaskInboxItem[]): Map<InboxGroupId, TaskInboxItem[]> {
  const buckets = new Map<InboxGroupId, TaskInboxItem[]>();
  for (const group of INBOX_GROUPS) {
    buckets.set(group.id, []);
  }
  for (const item of items) {
    const groupId = resolveInboxGroupId(item);
    buckets.get(groupId)?.push(item);
  }
  return buckets;
}

export function useTasksInboxView({
  inbox,
  laneFilter,
  onLaneChange,
  statusFilter,
  onStatusChange,
  priorityFilter,
  onPriorityChange,
  unreadOnly,
}: UseTasksInboxViewArgs) {
  const allItems = useMemo(() => flattenItems(inbox), [inbox]);
  const laneCounts = useMemo(() => computeLaneCounts(allItems), [allItems]);
  const filterFields = useMemo(() => buildInboxFilterFields(laneCounts), [laneCounts]);
  const filterChips = useMemo(
    () => inboxFiltersToChips({ laneFilter, statusFilter, priorityFilter }),
    [laneFilter, priorityFilter, statusFilter]
  );
  const handleFiltersChange = useCallback(
    (chips: Parameters<typeof applyInboxFilterChips>[0]) => {
      applyInboxFilterChips(chips, {
        onLaneChange,
        onStatusChange,
        onPriorityChange,
      });
    },
    [onLaneChange, onPriorityChange, onStatusChange]
  );
  const filteredItems = useMemo(() => {
    const lanedItems =
      laneFilter === "all"
        ? allItems
        : allItems.filter(item => backendLaneToUiLane(item.lane) === laneFilter);
    const statusedItems = statusFilter
      ? lanedItems.filter(item => item.task.status === statusFilter)
      : lanedItems;
    const prioritizedItems = priorityFilter
      ? statusedItems.filter(item => item.task.priority === priorityFilter)
      : statusedItems;
    return unreadOnly
      ? prioritizedItems.filter(item => !item.triage.read && !item.triage.dismissed)
      : prioritizedItems;
  }, [allItems, laneFilter, priorityFilter, statusFilter, unreadOnly]);
  const groups = useMemo(() => partitionByGroup(filteredItems), [filteredItems]);

  return {
    archivedTotal: inbox?.archived_total ?? 0,
    filterChips,
    filterFields,
    groups,
    handleFiltersChange,
    hasItems: filteredItems.length > 0,
    totalCount: inbox?.total ?? allItems.length,
    unreadTotal: inbox?.unread_total ?? 0,
    visibleCount: filteredItems.length,
  };
}
