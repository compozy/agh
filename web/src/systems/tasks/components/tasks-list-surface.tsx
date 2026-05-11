import { AlertCircle, ListChecks, Search } from "lucide-react";
import { useMemo } from "react";

import { Empty, Skeleton } from "@agh/ui";

import { groupTasksForList } from "../lib/task-grouping";
import type { TaskFilterOwnerOption, TaskScopeFilter } from "../lib/tasks-list-filters";
import type { TaskListItem, TaskPriority, TaskStatus } from "../types";
import { TaskCard } from "./task-card";
import { TaskGroup } from "./task-group";
import { TasksListFilters } from "./tasks-list-filters";
import { TasksListPageHead } from "./tasks-list-page-head";
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
  onSelectTask: (taskId: string) => void;
  workspaceName?: string | null;
  listUpdatedAt?: number;
  statusFilter: TaskStatus | null;
  ownerFilter: string | null;
  priorityFilter: TaskPriority | null;
  scopeFilter: TaskScopeFilter;
  ownerOptions: TaskFilterOwnerOption[];
  sortBy: TaskListSortKey;
  onStatusChange: (next: TaskStatus | null) => void;
  onOwnerChange: (next: string | null) => void;
  onPriorityChange: (next: TaskPriority | null) => void;
  onScopeChange: (next: TaskScopeFilter) => void;
  onSortChange: (next: TaskListSortKey) => void;
}

/**
 * Full-page `/tasks` list surface. Renders the page header (title + count +
 * meta), the chip filter bar, and the six status-grouped sections from
 * `groupTasksForList`. Replaces the deleted sidebar+detail `SplitPane` layout
 * — clicking a row navigates to `/tasks/$id` instead of opening an inline
 * preview.
 */
export function TasksListSurface({
  tasks,
  totalCount,
  isLoading = false,
  errorMessage = null,
  onSelectTask,
  workspaceName,
  listUpdatedAt,
  statusFilter,
  ownerFilter,
  priorityFilter,
  scopeFilter,
  ownerOptions,
  sortBy,
  onStatusChange,
  onOwnerChange,
  onPriorityChange,
  onScopeChange,
  onSortChange,
}: TasksListSurfaceProps) {
  const buckets = useMemo(
    () => groupTasksForList(tasks).filter(bucket => bucket.tasks.length > 0),
    [tasks]
  );

  const visibleCount = tasks.length;
  const hasFilters =
    Boolean(statusFilter) ||
    Boolean(ownerFilter) ||
    Boolean(priorityFilter) ||
    scopeFilter !== "all";

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-y-auto bg-canvas"
      data-testid="tasks-list-surface"
    >
      <div className="mx-auto w-full max-w-[1320px] px-9 pt-7 pb-20">
        <TasksListPageHead
          listUpdatedAt={listUpdatedAt}
          totalCount={totalCount}
          visibleCount={visibleCount}
          workspaceName={workspaceName}
        />
        <TasksListFilters
          onOwnerChange={onOwnerChange}
          onPriorityChange={onPriorityChange}
          onScopeChange={onScopeChange}
          onSortChange={onSortChange}
          onStatusChange={onStatusChange}
          ownerFilter={ownerFilter}
          ownerOptions={ownerOptions}
          priorityFilter={priorityFilter}
          scopeFilter={scopeFilter}
          sortBy={sortBy}
          statusFilter={statusFilter}
        />

        <div className="mt-4 flex flex-col gap-2" data-testid="tasks-list-surface-body">
          {isLoading && visibleCount === 0 ? (
            <div className="flex flex-col gap-2" data-testid="tasks-list-surface-loading">
              {TASK_LIST_SKELETON_IDS.map(id => (
                <div
                  className="flex items-center gap-3 border-b border-line-soft py-3 pr-3 pl-3.5"
                  key={id}
                >
                  <Skeleton className="size-1.5 rounded-full" />
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
                  <TaskCard key={task.id} onSelect={() => onSelectTask(task.id)} task={task} />
                ))}
              </TaskGroup>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
