import { AlertCircle, Plus, Search } from "lucide-react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Input } from "@agh/ui";
import { Button } from "@agh/ui";

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
  onCreateTask?: () => void;
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
  onCreateTask,
}: TasksListPanelProps) {
  const isEmpty = tasks.length === 0;

  return (
    <aside
      className="flex w-[360px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]"
      data-testid="tasks-list-panel"
    >
      <div className="space-y-3 border-b border-[color:var(--color-divider)] px-4 py-4">
        <div className="space-y-1">
          <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            Task rail
          </p>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Browse the current queue, select a task, or open a new contract in the main pane.
          </p>
        </div>

        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] pl-9"
            data-testid="tasks-list-search-input"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search tasks..."
            value={searchQuery}
          />
        </div>

        {onCreateTask ? (
          <Button
            className="w-full justify-center"
            data-testid="tasks-list-create"
            onClick={onCreateTask}
            size="lg"
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            New task
          </Button>
        ) : null}
      </div>

      <div className="flex items-center justify-between border-b border-[color:var(--color-divider)] px-4 py-2.5 text-[0.66rem] font-mono uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        <span data-testid="tasks-list-headline">
          {getStatusHeadline(statusFilter)}
          {tasks.length > 0 ? <span className="ml-2">{tasks.length}</span> : null}
        </span>
        <span data-testid="tasks-list-total">{totalCount} total</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div className="space-y-3 px-4 py-4" data-testid="tasks-list-loading">
            {Array.from({ length: 5 }, (_, index) => (
              <div
                className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
                key={index}
              >
                <div className="h-2.5 w-20 rounded-full bg-[color:var(--color-surface-elevated)]" />
                <div className="mt-3 h-3.5 w-3/4 rounded-full bg-[color:var(--color-surface-elevated)]" />
                <div className="mt-2 h-2.5 w-1/2 rounded-full bg-[color:var(--color-surface-elevated)]" />
              </div>
            ))}
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
            className="flex min-h-full items-center justify-center px-4 py-8"
            data-testid="tasks-list-empty"
          >
            <Empty className="border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-6 py-8">
              <EmptyHeader className="max-w-xs">
                <EmptyMedia className="flex size-10 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
                  <Search className="size-4" />
                </EmptyMedia>
                <EmptyTitle className="text-base font-semibold text-[color:var(--color-text-primary)]">
                  Nothing matches the current filters
                </EmptyTitle>
                <EmptyDescription className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
                  Adjust the search or open a new task contract from the rail.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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
