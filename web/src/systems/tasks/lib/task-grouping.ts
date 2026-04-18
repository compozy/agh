import type { TaskListItem, TaskStatus } from "../types";

export type TaskKanbanColumnId = "pending" | "running" | "done" | "failed";

export interface TaskKanbanColumn {
  id: TaskKanbanColumnId;
  label: string;
  statuses: TaskStatus[];
}

const KANBAN_COLUMNS: TaskKanbanColumn[] = [
  { id: "pending", label: "Pending", statuses: ["draft", "pending", "ready", "blocked"] },
  { id: "running", label: "Running", statuses: ["in_progress"] },
  { id: "done", label: "Done", statuses: ["completed"] },
  { id: "failed", label: "Failed", statuses: ["failed", "canceled"] },
];

const MOCK_STATUS_ALIASES: Record<string, TaskKanbanColumnId> = {
  running: "running",
  done: "done",
};

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

export function resolveKanbanColumnId(status: TaskStatus | string): TaskKanbanColumnId | null {
  for (const column of KANBAN_COLUMNS) {
    if ((column.statuses as readonly string[]).includes(status)) {
      return column.id;
    }
  }

  return MOCK_STATUS_ALIASES[status] ?? null;
}
