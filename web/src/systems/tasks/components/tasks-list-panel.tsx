import { AlertCircle, Loader2, Search } from "lucide-react";

import { Input } from "@agh/ui";

import type { TaskListItem, TaskStatus } from "../types";
import { TaskCard } from "./task-card";

export interface TasksListPanelProps {
  tasks: TaskListItem[];
  totalCount: number;
  selectedTaskId: string | null;
  onSelectTask: (taskId: string) => void;
  onPublishTask?: (taskId: string) => void;
  onRetryTask?: (taskId: string) => void;
  searchQuery: string;
  onSearchChange: (next: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  statusFilter?: TaskStatus | null;
  isPublishPending?: boolean;
}

const STATUS_HEADLINES: Partial<Record<TaskStatus, string>> = {
  in_progress: "In Progress",
  ready: "Ready",
  blocked: "Blocked",
  pending: "Pending",
  draft: "Draft",
  failed: "Failed",
  completed: "Completed",
  canceled: "Canceled",
};

function getStatusHeadline(filter?: TaskStatus | null): string {
  if (!filter) {
    return "All Tasks";
  }

  return STATUS_HEADLINES[filter] ?? "Tasks";
}

export function TasksListPanel({
  tasks,
  totalCount,
  selectedTaskId,
  onSelectTask,
  onPublishTask,
  onRetryTask,
  searchQuery,
  onSearchChange,
  isLoading = false,
  errorMessage = null,
  statusFilter = null,
  isPublishPending = false,
}: TasksListPanelProps) {
  const isEmpty = tasks.length === 0;

  return (
    <aside
      className="flex w-[360px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="tasks-list-panel"
    >
      <div className="border-b border-[color:var(--color-divider)] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="pl-8"
            data-testid="tasks-list-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search tasks..."
            value={searchQuery}
          />
        </div>
      </div>

      <div className="flex items-center justify-between border-b border-[color:var(--color-divider)] px-4 py-2 text-[0.66rem] font-mono uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        <span data-testid="tasks-list-headline">
          {getStatusHeadline(statusFilter)}
          {tasks.length > 0 ? <span className="ml-2">{tasks.length}</span> : null}
        </span>
        <span data-testid="tasks-list-total">{totalCount} total</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="tasks-list-loading"
          >
            <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
          </div>
        ) : errorMessage && isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10"
            data-testid="tasks-list-error"
          >
            <div className="flex max-w-xs flex-col items-center gap-2 text-center">
              <AlertCircle className="size-5 text-[color:var(--color-danger)]" />
              <p className="text-sm text-[color:var(--color-text-secondary)]">{errorMessage}</p>
            </div>
          </div>
        ) : isEmpty ? (
          <div
            className="flex min-h-full items-center justify-center px-6 py-10 text-center text-sm text-[color:var(--color-text-secondary)]"
            data-testid="tasks-list-empty"
          >
            No tasks match the current filters.
          </div>
        ) : (
          tasks.map(task => (
            <TaskCard
              isPublishPending={isPublishPending}
              key={task.id}
              onPublish={onPublishTask ? () => onPublishTask(task.id) : undefined}
              onRetry={onRetryTask ? () => onRetryTask(task.id) : undefined}
              onSelect={() => onSelectTask(task.id)}
              selected={task.id === selectedTaskId}
              task={task}
            />
          ))
        )}
      </div>
    </aside>
  );
}
