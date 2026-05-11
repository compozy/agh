import { AlertCircle, ListFilter, Search } from "lucide-react";
import { useCallback, useMemo } from "react";

import { BlockLoading, Button, Empty, Eyebrow, SearchInput, StatusDot, Switch } from "@agh/ui";
import { Filters } from "@agh/ui/components/reui/filters";

import {
  INBOX_GROUPS,
  INBOX_UI_LANES,
  type InboxGroupDefinition,
  type InboxGroupId,
  type InboxLaneFilterId,
  type InboxUiLane,
  backendLaneToUiLane,
  resolveInboxGroupId,
} from "../lib/inbox-grouping";
import {
  applyInboxFilterChips,
  buildInboxFilterFields,
  type InboxLaneCount,
  inboxFiltersToChips,
} from "../lib/inbox-filters";
import type { TaskInboxItem, TaskInboxView, TaskPriority, TaskStatus } from "../types";
import { TasksInboxItem, type TasksInboxItemProps } from "./tasks-inbox-item";
import { TasksInboxPageHead } from "./tasks-inbox-page-head";

export interface TasksInboxViewProps {
  inbox: TaskInboxView | null;
  laneFilter: InboxLaneFilterId;
  onLaneChange: (lane: InboxLaneFilterId) => void;
  statusFilter: TaskStatus | null;
  onStatusChange: (next: TaskStatus | null) => void;
  priorityFilter: TaskPriority | null;
  onPriorityChange: (next: TaskPriority | null) => void;
  unreadOnly: boolean;
  onToggleUnread: (next: boolean) => void;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  workspaceName?: string | null;
  /** Epoch ms of the last successful inbox fetch (TanStack `dataUpdatedAt`). */
  inboxUpdatedAt?: number;
  isLoading?: boolean;
  errorMessage?: string | null;
  onApprove?: TasksInboxItemProps["onApprove"];
  onReject?: TasksInboxItemProps["onReject"];
  onRetry?: TasksInboxItemProps["onRetry"];
  onArchive?: TasksInboxItemProps["onArchive"];
  onDismiss?: TasksInboxItemProps["onDismiss"];
  onMarkRead?: TasksInboxItemProps["onMarkRead"];
  onOpen?: TasksInboxItemProps["onOpen"];
  pendingApproveId?: string | null;
  pendingRejectId?: string | null;
  pendingRetryId?: string | null;
  pendingArchiveId?: string | null;
  pendingDismissId?: string | null;
  pendingMarkReadId?: string | null;
}

type LaneCount = InboxLaneCount;

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

export function TasksInboxView({
  inbox,
  laneFilter,
  onLaneChange,
  statusFilter,
  onStatusChange,
  priorityFilter,
  onPriorityChange,
  unreadOnly,
  onToggleUnread,
  searchQuery,
  onSearchChange,
  workspaceName,
  inboxUpdatedAt,
  isLoading = false,
  errorMessage = null,
  onApprove,
  onReject,
  onRetry,
  onArchive,
  onDismiss,
  onMarkRead,
  onOpen,
  pendingApproveId,
  pendingRejectId,
  pendingRetryId,
  pendingArchiveId,
  pendingDismissId,
  pendingMarkReadId,
}: TasksInboxViewProps) {
  const allItems = useMemo(() => flattenItems(inbox), [inbox]);
  const laneCounts = useMemo(() => computeLaneCounts(allItems), [allItems]);

  const filterFields = useMemo(() => buildInboxFilterFields(laneCounts), [laneCounts]);
  const filterChips = useMemo(
    () => inboxFiltersToChips({ laneFilter, statusFilter, priorityFilter }),
    [laneFilter, statusFilter, priorityFilter]
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
  const hasItems = filteredItems.length > 0;

  const itemActionProps: Omit<TasksInboxItemProps, "item" | "group"> = {
    onApprove,
    onReject,
    onRetry,
    onArchive,
    onDismiss,
    onMarkRead,
    onOpen,
    pendingApproveId,
    pendingRejectId,
    pendingRetryId,
    pendingArchiveId,
    pendingDismissId,
    pendingMarkReadId,
  };

  const totalCount = inbox?.total ?? allItems.length;
  const visibleCount = filteredItems.length;
  const unreadTotal = inbox?.unread_total ?? 0;
  const archivedTotal = inbox?.archived_total ?? 0;

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-y-auto bg-canvas"
      data-testid="tasks-inbox-view"
    >
      <div className="mx-auto w-full max-w-[1320px] px-9 pt-7 pb-20">
        <TasksInboxPageHead
          archivedCount={archivedTotal}
          inboxUpdatedAt={inboxUpdatedAt}
          totalCount={totalCount}
          unreadCount={unreadTotal}
          visibleCount={visibleCount}
          workspaceName={workspaceName}
        />

        <div
          className="flex flex-wrap items-center gap-2 border-b border-line-soft pb-3"
          data-testid="tasks-inbox-toolbar"
        >
          <SearchInput
            className="h-8 w-64 max-w-full"
            data-testid="tasks-inbox-search"
            onChange={next => onSearchChange(next)}
            placeholder="Search inbox..."
            value={searchQuery}
          />
          <Filters<string>
            allowMultiple={false}
            fields={filterFields}
            filters={filterChips}
            onChange={handleFiltersChange}
            size="sm"
            trigger={
              <Button
                aria-label="Filter inbox"
                data-testid="tasks-inbox-filter-trigger"
                size="sm"
                type="button"
                variant="ghost"
              >
                <ListFilter aria-hidden="true" className="size-3.5" />
                Filter
              </Button>
            }
          />
          <label
            className="ml-auto inline-flex items-center gap-2"
            data-testid="tasks-inbox-unread-toggle"
            htmlFor="tasks-inbox-unread-only"
          >
            <Switch
              checked={unreadOnly}
              id="tasks-inbox-unread-only"
              onCheckedChange={onToggleUnread}
            />
            <Eyebrow className="text-muted">Unread only</Eyebrow>
          </label>
        </div>

        <div className="mt-4 flex flex-col gap-6" data-testid="tasks-inbox-body">
          {isLoading && !inbox ? (
            <BlockLoading
              label="Loading inbox"
              size="md"
              surface="bare"
              data-testid="tasks-inbox-loading"
            />
          ) : errorMessage && !inbox ? (
            <Empty
              data-testid="tasks-inbox-error"
              description={errorMessage}
              icon={AlertCircle}
              title="Unable to load inbox"
            />
          ) : !hasItems ? (
            <Empty
              className="mx-auto max-w-xl"
              data-testid="tasks-inbox-empty"
              description="Approval requests, failed runs, blockers, and archived items will appear here as work progresses."
              icon={Search}
              title="Nothing is waiting in the inbox"
            />
          ) : (
            <div className="flex flex-col gap-6" data-testid="tasks-inbox-groups">
              {INBOX_GROUPS.map(group => {
                const bucket = groups.get(group.id) ?? [];
                if (bucket.length === 0) {
                  return null;
                }
                return (
                  <GroupSection
                    group={group}
                    items={bucket}
                    itemActionProps={itemActionProps}
                    key={group.id}
                  />
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface GroupSectionProps {
  group: InboxGroupDefinition;
  items: TaskInboxItem[];
  itemActionProps: Omit<TasksInboxItemProps, "item" | "group">;
}

function GroupSection({ group, items, itemActionProps }: GroupSectionProps) {
  return (
    <section className="flex flex-col gap-2" data-testid={`tasks-inbox-group-${group.id}`}>
      <header className="flex items-center gap-2">
        <StatusDot
          data-testid={`tasks-inbox-group-dot-${group.id}`}
          label={group.label}
          tone={group.dotTone}
          variant={group.dotVariant}
        />
        <Eyebrow>{group.label}</Eyebrow>
        <span
          className="font-mono text-[10px] tabular-nums text-faint"
          data-testid={`tasks-inbox-group-count-${group.id}`}
        >
          {items.length}
        </span>
      </header>
      <div className="flex flex-col">
        {items.map(item => (
          <TasksInboxItem {...itemActionProps} group={group.id} item={item} key={item.task.id} />
        ))}
      </div>
    </section>
  );
}
