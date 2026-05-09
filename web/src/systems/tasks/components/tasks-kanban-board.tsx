import { AlertCircle, Plus } from "lucide-react";

import { BlockLoading, Button, Section, Pill } from "@agh/ui";

import type { KanbanColumnGroup, TaskKanbanColumnId } from "../lib/task-grouping";
import { formatAttemptLabel, taskOwnerLabel } from "../lib/task-formatters";
import type { TaskListItem } from "../types";
import { TasksListRow } from "./tasks-list-row";

import type { PillTone } from "@agh/ui";

const COLUMN_HEADER_TONE: Record<TaskKanbanColumnId, PillTone> = {
  pending: "neutral",
  running: "accent",
  done: "success",
  failed: "danger",
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
      <BlockLoading
        className="flex-1"
        label="Loading kanban board"
        size="md"
        surface="bare"
        data-testid="tasks-kanban-loading"
      />
    );
  }

  if (errorMessage) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="tasks-kanban-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">{errorMessage}</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex min-h-0 flex-1 gap-4 overflow-x-auto px-4 py-4"
      data-testid="tasks-kanban-board"
      role="list"
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
  const headerTone = COLUMN_HEADER_TONE[column.id];

  return (
    <Section
      data-testid={`tasks-kanban-column-${column.id}`}
      role="listitem"
      className="min-w-[260px] flex-1"
      label={
        <span className="inline-flex items-center gap-2">
          <Pill.Dot tone={headerTone} />
          <span>{column.label}</span>
          <span
            className="font-mono text-badge font-medium tracking-badge text-(--color-text-tertiary)"
            data-testid={`tasks-kanban-column-count-${column.id}`}
          >
            {tasks.length}
          </span>
        </span>
      }
      right={
        onCreate ? (
          <Button
            aria-label={`Add task to ${column.label}`}
            data-testid={`tasks-kanban-column-add-${column.id}`}
            onClick={onCreate}
            size="icon-xs"
            type="button"
            variant="ghost"
          >
            <Plus />
          </Button>
        ) : undefined
      }
    >
      <div className="flex min-h-0 flex-1 flex-col gap-2 pt-2 pb-4">
        {tasks.length === 0 ? (
          <div
            className="flex flex-1 items-center justify-center rounded-(--radius-diagram) border border-dashed border-(--color-divider) px-3 py-8 text-center text-xs text-(--color-text-tertiary)"
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
              onSelect={onSelectTask}
              task={task}
            />
          ))
        )}
      </div>
    </Section>
  );
}

interface KanbanCardProps {
  task: TaskListItem;
  isSelected: boolean;
  onSelect: (taskId: string) => void;
  onRetry?: () => void;
}

function KanbanCard({ task, isSelected, onSelect, onRetry }: KanbanCardProps) {
  const activeRun = task.active_run ?? null;
  const isLive = task.status === "in_progress" && activeRun !== null;
  const isBlocked = task.status === "blocked";
  const failedRunError =
    task.status === "failed" && task.active_run?.error ? task.active_run.error : null;
  const canRetry = task.status === "failed" && Boolean(onRetry);

  const footer = (
    <div className="flex min-w-0 flex-col gap-1 text-eyebrow">
      {isLive && activeRun ? (
        <span className="text-badge text-accent" data-testid={`tasks-kanban-card-live-${task.id}`}>
          ● LIVE · {formatAttemptLabel(activeRun.attempt, activeRun.max_attempts) ?? "running"}
        </span>
      ) : null}
      {isBlocked ? (
        <span
          className="text-badge text-(--color-warning)"
          data-testid={`tasks-kanban-card-blocked-${task.id}`}
        >
          ● Blocked
        </span>
      ) : null}
      {failedRunError ? (
        <span
          className="text-badge text-(--color-danger)"
          data-testid={`tasks-kanban-card-error-${task.id}`}
        >
          {failedRunError}
        </span>
      ) : null}
      <div className="flex items-center justify-between gap-2 text-(--color-text-secondary)">
        <span data-testid={`tasks-kanban-card-owner-${task.id}`}>{taskOwnerLabel(task.owner)}</span>
        {canRetry ? (
          <Button
            aria-label={`Retry ${task.title}`}
            data-testid={`tasks-kanban-card-retry-${task.id}`}
            onClick={event => {
              event.stopPropagation();
              onRetry?.();
            }}
            size="xs"
            type="button"
            variant="outline"
          >
            Retry
          </Button>
        ) : null}
      </div>
    </div>
  );

  return (
    <TasksListRow
      className="rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) px-3.5 py-3"
      footer={footer}
      onSelect={onSelect}
      selected={isSelected}
      task={task}
      testId={`tasks-kanban-card-${task.id}`}
    />
  );
}
