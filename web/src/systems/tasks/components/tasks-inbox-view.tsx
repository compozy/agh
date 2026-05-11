import { AlertCircle, Search } from "lucide-react";
import { useMemo } from "react";

import {
  BlockLoading,
  Empty,
  Eyebrow,
  SearchInput,
  StatusDot,
  Switch,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@agh/ui";

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
import type { TaskInboxItem, TaskInboxView } from "../types";
import { TasksInboxItem, type TasksInboxItemProps } from "./tasks-inbox-item";

export interface TasksInboxViewProps {
  inbox: TaskInboxView | null;
  laneFilter: InboxLaneFilterId;
  onLaneChange: (lane: InboxLaneFilterId) => void;
  unreadOnly: boolean;
  onToggleUnread: (next: boolean) => void;
  searchQuery: string;
  onSearchChange: (value: string) => void;
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

interface LaneCount {
  count: number;
  unread: number;
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

export function TasksInboxView({
  inbox,
  laneFilter,
  onLaneChange,
  unreadOnly,
  onToggleUnread,
  searchQuery,
  onSearchChange,
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

  const filteredItems = useMemo(() => {
    const lanedItems =
      laneFilter === "all"
        ? allItems
        : allItems.filter(item => backendLaneToUiLane(item.lane) === laneFilter);
    return unreadOnly
      ? lanedItems.filter(item => !item.triage.read && !item.triage.dismissed)
      : lanedItems;
  }, [allItems, laneFilter, unreadOnly]);

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

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-inbox-view">
      <div className="border-b border-line px-4 py-2.5" data-testid="tasks-inbox-lane-tabs">
        <Tabs
          onValueChange={next => onLaneChange(next as InboxLaneFilterId)}
          orientation="horizontal"
          value={laneFilter}
        >
          <TabsList className="h-8 overflow-x-auto" variant="line">
            <TabsTrigger
              className="flex-none gap-1.5"
              count={allItems.length}
              data-testid="tasks-inbox-lane-all"
              value="all"
            >
              All
            </TabsTrigger>
            {INBOX_UI_LANES.map(lane => {
              const counts = laneCounts.get(lane.id);
              return (
                <TabsTrigger
                  className="flex-none gap-1.5"
                  count={counts?.count ?? 0}
                  data-testid={`tasks-inbox-lane-${lane.id}`}
                  key={lane.id}
                  value={lane.id}
                >
                  {lane.label}
                </TabsTrigger>
              );
            })}
          </TabsList>
        </Tabs>
      </div>

      <div className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3">
        <SearchInput
          className="h-9 min-w-[220px] flex-1"
          data-testid="tasks-inbox-search"
          onChange={next => onSearchChange(next)}
          placeholder="Search inbox..."
          value={searchQuery}
        />
        <label
          className="inline-flex items-center gap-2"
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
        <Eyebrow data-testid="tasks-inbox-totals">
          {inbox?.unread_total ?? 0} unread · {inbox?.archived_total ?? 0} archived
        </Eyebrow>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
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
      <header className="flex items-baseline gap-2">
        <StatusDot
          data-testid={`tasks-inbox-group-dot-${group.id}`}
          label={group.label}
          tone={group.dotTone}
          variant={group.dotVariant}
        />
        <Eyebrow>{group.label}</Eyebrow>
        <Eyebrow data-testid={`tasks-inbox-group-count-${group.id}`}>({items.length})</Eyebrow>
      </header>
      <div className="flex flex-col">
        {items.map(item => (
          <TasksInboxItem {...itemActionProps} group={group.id} item={item} key={item.task.id} />
        ))}
      </div>
    </section>
  );
}
