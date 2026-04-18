import type { TaskListItem, TaskStatus } from "../types";

export type TaskKanbanColumnId =
  | "pending"
  | "ready"
  | "in_progress"
  | "blocked"
  | "completed"
  | "failed";

export interface TaskKanbanColumn {
  id: TaskKanbanColumnId;
  label: string;
  statuses: TaskStatus[];
}

const KANBAN_COLUMNS: TaskKanbanColumn[] = [
  { id: "pending", label: "Pending", statuses: ["draft", "pending"] },
  { id: "ready", label: "Ready", statuses: ["ready"] },
  { id: "in_progress", label: "In Progress", statuses: ["in_progress"] },
  { id: "blocked", label: "Blocked", statuses: ["blocked"] },
  { id: "completed", label: "Completed", statuses: ["completed"] },
  { id: "failed", label: "Failed", statuses: ["failed", "canceled"] },
];

export interface KanbanColumnGroup {
  column: TaskKanbanColumn;
  tasks: TaskListItem[];
}

export function getKanbanColumns(): TaskKanbanColumn[] {
  return KANBAN_COLUMNS;
}

export function groupTasksForKanban(tasks: TaskListItem[]): KanbanColumnGroup[] {
  const buckets = new Map<TaskKanbanColumnId, TaskListItem[]>();
  for (const column of KANBAN_COLUMNS) {
    buckets.set(column.id, []);
  }

  for (const task of tasks) {
    const columnId = resolveKanbanColumnId(task.status);
    if (!columnId) {
      continue;
    }

    buckets.get(columnId)?.push(task);
  }

  return KANBAN_COLUMNS.map(column => ({
    column,
    tasks: buckets.get(column.id) ?? [],
  }));
}

export function resolveKanbanColumnId(status: TaskStatus): TaskKanbanColumnId | null {
  for (const column of KANBAN_COLUMNS) {
    if (column.statuses.includes(status)) {
      return column.id;
    }
  }

  return null;
}
