import { AlertCircle, Loader2, Plus } from "lucide-react";

import { cn } from "@/lib/utils";

import type { KanbanColumnGroup } from "../lib/task-grouping";
import {
  formatAttemptLabel,
  formatRelativeTime,
  taskOwnerLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskListItem } from "../types";

const COLUMN_TONE_DOT: Record<string, string> = {
  pending: "bg-[color:var(--color-text-tertiary)]",
  ready: "bg-[color:var(--color-info)]",
  in_progress: "bg-[color:var(--color-accent)]",
  blocked: "bg-[color:var(--color-warning)]",
  completed: "bg-[color:var(--color-success)]",
  failed: "bg-[color:var(--color-danger)]",
};

export interface TasksKanbanBoardProps {
  columns: KanbanColumnGroup[];
  selectedTaskId: string | null;
  onSelectTask: (taskId: string) => void;
  onCreateInColumn?: (columnId: string) => void;
  onRetryTask?: (taskId: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
}

export function TasksKanbanBoard({
  columns,
  selectedTaskId,
  onSelectTask,
  onCreateInColumn,
  onRetryTask,
  isLoading = false,
  errorMessage = null,
}: TasksKanbanBoardProps) {
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="tasks-kanban-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="tasks-kanban-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">{errorMessage}</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex min-h-0 flex-1 gap-4 overflow-x-auto px-4 py-4"
      data-testid="tasks-kanban-board"
    >
      {columns.map(group => (
        <KanbanColumn
          column={group.column}
          key={group.column.id}
          onCreate={onCreateInColumn ? () => onCreateInColumn(group.column.id) : undefined}
          onRetryTask={onRetryTask}
          onSelectTask={onSelectTask}
          selectedTaskId={selectedTaskId}
          tasks={group.tasks}
        />
      ))}
    </div>
  );
}

interface KanbanColumnProps {
  column: KanbanColumnGroup["column"];
  tasks: TaskListItem[];
  selectedTaskId: string | null;
  onSelectTask: (taskId: string) => void;
  onCreate?: () => void;
  onRetryTask?: (taskId: string) => void;
}

function KanbanColumn({
  column,
  tasks,
  selectedTaskId,
  onSelectTask,
  onCreate,
  onRetryTask,
}: KanbanColumnProps) {
  return (
    <section
      className="flex w-[280px] shrink-0 flex-col"
      data-testid={`tasks-kanban-column-${column.id}`}
    >
      <header className="flex items-center justify-between px-2 py-3">
        <div className="flex items-center gap-2">
          <span className={cn("inline-block size-2 rounded-full", COLUMN_TONE_DOT[column.id])} />
          <h3 className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            {column.label}
          </h3>
          <span
            className="font-mono text-[0.66rem] tracking-[0.14em] text-[color:var(--color-text-secondary)]"
            data-testid={`tasks-kanban-column-count-${column.id}`}
          >
            {tasks.length}
          </span>
        </div>
        {onCreate ? (
          <button
            aria-label={`Add task to ${column.label}`}
            className="rounded-full border border-transparent p-1 text-[color:var(--color-text-tertiary)] transition-colors hover:border-[color:var(--color-divider)] hover:text-[color:var(--color-text-primary)]"
            data-testid={`tasks-kanban-column-add-${column.id}`}
            onClick={onCreate}
            type="button"
          >
            <Plus className="size-3.5" />
          </button>
        ) : null}
      </header>

      <div className="flex flex-1 flex-col gap-2 overflow-y-auto pb-4">
        {tasks.length === 0 ? (
          <div
            className="flex flex-1 items-center justify-center rounded-2xl border border-dashed border-[color:rgba(58,58,60,0.4)] px-3 py-8 text-center text-xs text-[color:var(--color-text-tertiary)]"
            data-testid={`tasks-kanban-column-empty-${column.id}`}
          >
            No tasks
          </div>
        ) : (
          tasks.map(task => (
            <KanbanCard
              isSelected={task.id === selectedTaskId}
              key={task.id}
              onRetry={onRetryTask ? () => onRetryTask(task.id) : undefined}
              onSelect={() => onSelectTask(task.id)}
              task={task}
            />
          ))
        )}
      </div>
    </section>
  );
}

interface KanbanCardProps {
  task: TaskListItem;
  isSelected: boolean;
  onSelect: () => void;
  onRetry?: () => void;
}

function KanbanCard({ task, isSelected, onSelect, onRetry }: KanbanCardProps) {
  const tone = taskStatusTone(task.status);
  const activeRun = task.active_run ?? null;
  const isLive = task.status === "in_progress" && activeRun !== null;
  const isBlocked = task.status === "blocked";
  const failedRunError =
    task.status === "failed" && task.active_run?.error ? task.active_run.error : null;

  return (
    <button
      aria-pressed={isSelected}
      className={cn(
        "group flex flex-col gap-2 rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-3.5 py-3 text-left transition-colors",
        "hover:border-[color:var(--color-text-label)]",
        isSelected && "border-[color:var(--color-accent)]"
      )}
      data-testid={`tasks-kanban-card-${task.id}`}
      onClick={onSelect}
      type="button"
    >
      <div className="flex items-start justify-between gap-2 text-xs text-[color:var(--color-text-tertiary)]">
        {task.identifier ? (
          <span className="font-mono uppercase tracking-[0.12em]">{task.identifier}</span>
        ) : (
          <span />
        )}
        <span className="shrink-0">
          {formatRelativeTime(task.last_activity_at ?? task.updated_at)}
        </span>
      </div>

      <p className="text-sm font-medium leading-snug text-[color:var(--color-text-primary)]">
        {task.title}
      </p>

      {isLive && activeRun ? (
        <p
          className="font-mono text-[0.65rem] uppercase tracking-[0.12em] text-[color:var(--color-accent)]"
          data-testid={`tasks-kanban-card-live-${task.id}`}
        >
          ● LIVE · {formatAttemptLabel(activeRun.attempt, activeRun.max_attempts) ?? "running"}
        </p>
      ) : null}

      {isBlocked ? (
        <p
          className="font-mono text-[0.65rem] uppercase tracking-[0.12em] text-[color:var(--color-warning)]"
          data-testid={`tasks-kanban-card-blocked-${task.id}`}
        >
          ● Blocked
        </p>
      ) : null}

      {failedRunError ? (
        <p
          className="font-mono text-[0.65rem] uppercase tracking-[0.12em] text-[color:var(--color-danger)]"
          data-testid={`tasks-kanban-card-error-${task.id}`}
        >
          {failedRunError}
        </p>
      ) : null}

      <div className="flex items-center justify-between gap-2 text-xs text-[color:var(--color-text-secondary)]">
        <span data-testid={`tasks-kanban-card-owner-${task.id}`}>{taskOwnerLabel(task.owner)}</span>
        {task.status === "failed" && onRetry ? (
          <button
            aria-label={`Retry ${task.title}`}
            className="rounded-full border border-[color:var(--color-accent)] px-2 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-accent)] transition-colors hover:bg-[color:var(--color-accent-tint)]"
            data-testid={`tasks-kanban-card-retry-${task.id}`}
            onClick={event => {
              event.stopPropagation();
              onRetry();
            }}
            type="button"
          >
            Retry
          </button>
        ) : (
          <span
            className="text-[color:var(--color-text-tertiary)]"
            data-testid={`tasks-kanban-card-tone-${task.id}`}
          >
            {tone}
          </span>
        )}
      </div>
    </button>
  );
}
