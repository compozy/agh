import { AlertCircle, ListChecks, Search } from "lucide-react";
import { useMemo } from "react";

import { Empty, ListingPage, ListingToolbar, Skeleton } from "@agh/ui";

import { groupTasksForList } from "../lib/task-grouping";
import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskFilterOwnerOption } from "../lib/tasks-list-filters";
import type { TaskListItem, TaskPriority, TaskStatus } from "../types";
import { TaskCard } from "./task-card";
import { TaskGroup } from "./task-group";
import { TasksListFilters } from "./tasks-list-filters";
import { TasksListSort } from "./tasks-list-sort";
import type { TaskListSortKey } from "@/hooks/routes/use-tasks-page";

const TASK_LIST_SKELETON_IDS = [
  "task-list-skeleton-1",
  "task-list-skeleton-2",
  "task-list-skeleton-3",
  "task-list-skeleton-4",
  "task-list-skeleton-5",
];

export interface TasksListSurfaceProps {
  tasks: TaskListItem[];
  totalCount: number;
  isLoading?: boolean;
  errorMessage?: string | null;
  workspaceName?: string | null;
  listUpdatedAt?: number;
  statusFilter: TaskStatus | null;
  ownerFilter: string | null;
  priorityFilter: TaskPriority | null;
  ownerOptions: TaskFilterOwnerOption[];
  sortBy: TaskListSortKey;
  searchQuery: string;
  onStatusChange: (next: TaskStatus | null) => void;
  onOwnerChange: (next: string | null) => void;
  onPriorityChange: (next: TaskPriority | null) => void;
  onSortChange: (next: TaskListSortKey) => void;
  onSearchQueryChange: (next: string) => void;
}

/**
 * Full-page `/tasks` list surface. Composes the shared `ListingPage` shell
 * (width-capped container + page head) and `ListingToolbar` (search · filters ·
 * sort), then renders the status-grouped sections from `groupTasksForList` —
 * each group's rows sit in the same bordered list card as the Loops catalog.
 */
export function TasksListSurface({
  tasks,
  totalCount,
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
}: TasksListSurfaceProps) {
  const buckets = useMemo(
    () => groupTasksForList(tasks).filter(bucket => bucket.tasks.length > 0),
    [tasks]
  );

  const visibleCount = tasks.length;
  const hasFilters = Boolean(statusFilter) || Boolean(ownerFilter) || Boolean(priorityFilter);
  const countLabel =
    visibleCount === totalCount ? `${totalCount}` : `${visibleCount} of ${totalCount}`;
  const syncedLabel = listUpdatedAt
    ? formatRelativeTime(new Date(listUpdatedAt).toISOString())
    : null;
  const syncedText = syncedLabel
    ? syncedLabel === "now"
      ? "synced just now"
      : `synced ${syncedLabel} ago`
    : null;

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
          <Empty
            data-testid="tasks-list-surface-error"
            description={errorMessage}
            icon={AlertCircle}
            title="Unable to load tasks"
          />
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
            >
              {bucket.tasks.map(task => (
                <TaskCard key={task.id} task={task} />
              ))}
            </TaskGroup>
          ))
        )}
      </div>
    </ListingPage>
  );
}
