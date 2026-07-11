import { AlertCircle, ListChecks, Search } from "lucide-react";
import { useMemo } from "react";

import { Button, Empty, ListingPage, ListingToolbar, Skeleton, Spinner } from "@agh/ui";

import { groupTasksForList, taskStatusFacetTotal } from "../lib/task-grouping";
import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskFilterOwnerOption } from "../lib/tasks-list-filters";
import type { TaskListItem, TaskListSortKey, TaskPriority, TaskStatus } from "../types";
import { TaskCard } from "./task-card";
import { TaskGroup } from "./task-group";
import { TasksListFilters } from "./tasks-list-filters";
import { TasksListSort } from "./tasks-list-sort";

const TASK_LIST_SKELETON_IDS = [
  "task-list-skeleton-1",
  "task-list-skeleton-2",
  "task-list-skeleton-3",
  "task-list-skeleton-4",
  "task-list-skeleton-5",
];

export interface TasksListSurfaceProps {
  tasks: TaskListItem[];
  totalCount?: number;
  statusCounts: Record<TaskStatus, number>;
  isLoading?: boolean;
  errorMessage?: string | null;
  workspaceName?: string | null;
  listUpdatedAt?: number;
  statusFilter: TaskStatus | null;
  ownerFilter: TaskFilterOwnerOption | null;
  priorityFilter: TaskPriority | null;
  ownerOptions: TaskFilterOwnerOption[];
  sortBy: TaskListSortKey;
  searchQuery: string;
  onStatusChange: (next: TaskStatus | null) => void;
  onOwnerChange: (next: TaskFilterOwnerOption | null) => void;
  onPriorityChange: (next: TaskPriority | null) => void;
  onSortChange: (next: TaskListSortKey) => void;
  onSearchQueryChange: (next: string) => void;
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void;
  onRetryLoad?: () => void;
}

export function TasksListSurface({
  tasks,
  totalCount,
  statusCounts,
  isLoading = false,
  errorMessage = null,
  workspaceName,
  listUpdatedAt,
  statusFilter,
  ownerFilter,
  priorityFilter,
  ownerOptions,
  sortBy,
  searchQuery,
  onStatusChange,
  onOwnerChange,
  onPriorityChange,
  onSortChange,
  onSearchQueryChange,
  hasMore = false,
  isLoadingMore = false,
  onLoadMore,
  onRetryLoad,
}: TasksListSurfaceProps) {
  const buckets = useMemo(
    () =>
      groupTasksForList(tasks).filter(
        bucket =>
          bucket.tasks.length > 0 || taskStatusFacetTotal(bucket.group.statuses, statusCounts) > 0
      ),
    [statusCounts, tasks]
  );

  const visibleCount = tasks.length;
  const hasFilters =
    Boolean(statusFilter) ||
    Boolean(ownerFilter) ||
    Boolean(priorityFilter) ||
    searchQuery.trim() !== "";
  const countLabel =
    totalCount === undefined
      ? undefined
      : visibleCount === totalCount
        ? `${totalCount}`
        : `${visibleCount} of ${totalCount}`;
  let syncedText: string | null = null;
  if (listUpdatedAt) {
    const syncedLabel = formatRelativeTime(new Date(listUpdatedAt).toISOString());
    syncedText = syncedLabel === "now" ? "synced just now" : `synced ${syncedLabel} ago`;
  }

  return (
    <ListingPage data-testid="tasks-list-surface">
      <ListingPage.Head
        count={countLabel}
        countTestId="tasks-list-page-count"
        data-testid="tasks-list-page-head"
        meta={
          workspaceName || syncedText ? (
            <>
              {workspaceName ? (
                <span data-testid="tasks-list-page-workspace">workspace {workspaceName}</span>
              ) : null}
              {workspaceName && syncedText ? <ListingPage.MetaDot /> : null}
              {syncedText ? <span data-testid="tasks-list-page-synced">{syncedText}</span> : null}
            </>
          ) : undefined
        }
        title={<span data-testid="tasks-list-page-title">Tasks</span>}
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search tasks"
            data-testid="tasks-list-search-input"
            onChange={onSearchQueryChange}
            placeholder="Search tasks"
            value={searchQuery}
          />
          <ListingToolbar.Filters>
            <TasksListFilters
              onOwnerChange={onOwnerChange}
              onPriorityChange={onPriorityChange}
              onStatusChange={onStatusChange}
              ownerFilter={ownerFilter}
              ownerOptions={ownerOptions}
              priorityFilter={priorityFilter}
              statusFilter={statusFilter}
            />
          </ListingToolbar.Filters>
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <TasksListSort onSortChange={onSortChange} sortBy={sortBy} />
        </ListingToolbar.Trailing>
      </ListingToolbar>

      <div className="flex flex-col gap-5" data-testid="tasks-list-surface-body">
        {isLoading && visibleCount === 0 ? (
          <div
            className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
            data-testid="tasks-list-surface-loading"
          >
            {TASK_LIST_SKELETON_IDS.map(id => (
              <div
                className="flex items-center gap-3.5 border-b border-line-soft px-4 py-3 last:border-b-0"
                key={id}
              >
                <Skeleton className="size-[34px] shrink-0 rounded-md" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <Skeleton className="h-3 w-3/5 rounded-xs" />
                  <Skeleton className="h-2.5 w-2/5 rounded-xs" />
                </div>
              </div>
            ))}
          </div>
        ) : errorMessage && visibleCount === 0 ? (
          <div className="flex flex-col items-center gap-3">
            <Empty
              data-testid="tasks-list-surface-error"
              description={errorMessage}
              icon={AlertCircle}
              title="Unable to load tasks"
            />
            {onRetryLoad ? (
              <Button onClick={onRetryLoad} size="sm" type="button" variant="ghost">
                Retry loading tasks
              </Button>
            ) : null}
          </div>
        ) : visibleCount === 0 ? (
          <Empty
            data-testid="tasks-list-surface-empty"
            description={
              hasFilters
                ? "Clear filters to see other tasks in this workspace."
                : "Open a new task contract from the topbar to populate this list."
            }
            icon={hasFilters ? Search : ListChecks}
            title={hasFilters ? "No tasks match the current filters" : "No tasks yet"}
          />
        ) : (
          buckets.map(bucket => (
            <TaskGroup
              count={bucket.tasks.length}
              id={bucket.group.id}
              key={bucket.group.id}
              label={bucket.group.label}
              totalCount={taskStatusFacetTotal(bucket.group.statuses, statusCounts)}
            >
              {bucket.tasks.map(task => (
                <TaskCard key={task.id} task={task} />
              ))}
            </TaskGroup>
          ))
        )}
        {errorMessage && visibleCount > 0 ? (
          <div
            className="flex items-center justify-between gap-3 border-t border-line-soft pt-3 text-caption text-danger"
            data-testid="tasks-list-surface-pagination-error"
            role="alert"
          >
            <span>{errorMessage}</span>
            {onRetryLoad ? (
              <Button onClick={onRetryLoad} size="sm" type="button" variant="ghost">
                Retry loading tasks
              </Button>
            ) : null}
          </div>
        ) : null}
        {hasMore && onLoadMore && !errorMessage ? (
          <div className="flex items-center justify-center border-t border-line-soft pt-3">
            <Button
              aria-busy={isLoadingMore}
              aria-label={isLoadingMore ? "Loading more tasks" : "Load more tasks"}
              data-testid="tasks-list-load-more"
              disabled={isLoadingMore}
              onClick={onLoadMore}
              size="sm"
              type="button"
              variant="ghost"
            >
              {isLoadingMore ? <Spinner aria-hidden="true" className="size-3" /> : null}
              {isLoadingMore ? "Loading more" : "Load more"}
            </Button>
          </div>
        ) : null}
      </div>
    </ListingPage>
  );
}
