import { AlertCircle, ListFilter, Search } from "lucide-react";

import { BlockLoading, Button, Empty, Eyebrow, SearchInput, StatusDot, Switch } from "@agh/ui";
import { Filters } from "@agh/ui/components/reui/filters";

import {
  INBOX_GROUPS,
  type InboxGroupDefinition,
  type InboxLaneFilterId,
} from "../lib/inbox-grouping";
import { useTasksInboxView } from "../hooks/use-tasks-inbox-view";
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
  const {
    archivedTotal,
    filterChips,
    filterFields,
    groups,
    handleFiltersChange,
    hasItems,
    totalCount,
    unreadTotal,
    visibleCount,
  } = useTasksInboxView({
    inbox,
    laneFilter,
    onLaneChange,
    statusFilter,
    onStatusChange,
    priorityFilter,
    onPriorityChange,
    unreadOnly,
  });

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
    <div
      className="flex min-h-0 flex-1 flex-col overflow-y-auto bg-canvas"
      data-testid="tasks-inbox-view"
    >
      <div className="mx-auto w-full max-w-content-max px-9 pt-7 pb-20">
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
                <ListFilter aria-hidden="true" className="size-3" />
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
          className="font-mono text-badge tabular-nums text-faint"
          data-testid={`tasks-inbox-group-count-${group.id}`}
        >
          {items.length}
        </span>
      </header>
      <div className="flex flex-col">
        {items.map(item => (
          <TasksInboxItem key={item.task.id} {...itemActionProps} group={group.id} item={item} />
        ))}
      </div>
    </section>
  );
}
