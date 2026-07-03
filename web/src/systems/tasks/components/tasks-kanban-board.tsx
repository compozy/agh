import { AlertCircle } from "lucide-react";

import { Empty, Skeleton } from "@agh/ui";

import { TaskKanbanCard } from "./task-kanban-card";
import { TaskKanbanColumn } from "./task-kanban-column";
import type { KanbanColumnGroup, TaskKanbanColumnId } from "../lib/task-grouping";

import type { PillTone } from "@agh/ui";

/**
 * Column header tone — `In progress` reads as `info` (live work without an
 * accent recolor), `Blocked` reads as `danger`, `Needs attention` reads as
 * `warning` (distinct escalation, no coercion), terminal `Done` and `Pending`
 * stay neutral.
 */
const COLUMN_HEADER_TONE: Record<TaskKanbanColumnId, PillTone> = {
  pending: "neutral",
  in_progress: "info",
  blocked: "danger",
  needs_attention: "warning",
  done: "neutral",
};

const KANBAN_SKELETON_KEYS = ["a", "b", "c"] as const;

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
  if (errorMessage) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-kanban-error"
        role="alert"
      >
        <Empty description={errorMessage} icon={AlertCircle} title="Unable to load kanban" />
      </div>
    );
  }

  // Derive the grid track count from the canonical column set so it can never
  // drift from `KANBAN_COLUMNS`: a new status column widens the grid instead of
  // wrapping the last column onto a broken second row.
  return (
    <div
      className="grid min-h-0 flex-1 gap-3 overflow-y-auto px-4 pt-4 pb-15"
      data-testid="tasks-kanban-board"
      role="list"
      style={{
        gridTemplateColumns: `repeat(${Math.max(columns.length, 1)}, minmax(0, 1fr))`,
      }}
    >
      {isLoading ? (
        <span aria-live="polite" className="sr-only" data-testid="tasks-kanban-loading">
          Loading kanban board
        </span>
      ) : null}
      {columns.map(group => (
        <TaskKanbanColumn
          column={group.column}
          count={group.tasks.length}
          key={group.column.id}
          onAdd={onCreateInColumn ? () => onCreateInColumn(group.column.id) : undefined}
          tone={COLUMN_HEADER_TONE[group.column.id]}
        >
          {isLoading
            ? KANBAN_SKELETON_KEYS.map(slot => (
                <KanbanCardSkeleton key={`${group.column.id}-skeleton-${slot}`} />
              ))
            : group.tasks.map(task => (
                <TaskKanbanCard
                  key={task.id}
                  onRetry={onRetryTask}
                  onSelect={onSelectTask}
                  selected={task.id === selectedTaskId}
                  task={task}
                />
              ))}
        </TaskKanbanColumn>
      ))}
    </div>
  );
}

function KanbanCardSkeleton() {
  return (
    <div
      aria-hidden="true"
      className="flex w-full min-w-0 flex-col gap-2 rounded-md bg-canvas-tint p-3"
      data-testid="tasks-kanban-card-skeleton"
    >
      <Skeleton className="h-3 w-4/5 rounded-xs" />
      <Skeleton className="h-2.5 w-12 rounded-xs" />
      <div className="flex items-center justify-between gap-2">
        <Skeleton className="h-2.5 w-24 rounded-xs" />
        <Skeleton className="h-2.5 w-8 rounded-xs" />
      </div>
    </div>
  );
}
