import type { TaskListItem, TaskStatus } from "../types";

/**
 * Kanban column set (master rollup §2.6): four columns in
 * declared order — `Pending · In progress · Blocked · Done`. The previous
 * `Pending / Running / Done / Failed` set is gone (greenfield, no aliases).
 *
 * Terminal statuses (`completed`, `failed`, `canceled`) collapse into the
 * `done` column — the proposal's kanban surface treats every terminal state
 * as "off the board" and routes failure detail to the row card itself.
 */
export type TaskKanbanColumnId = "pending" | "in_progress" | "blocked" | "done";

export interface TaskKanbanColumn {
  id: TaskKanbanColumnId;
  label: string;
  statuses: TaskStatus[];
}

const KANBAN_COLUMNS: TaskKanbanColumn[] = [
  { id: "pending", label: "Pending", statuses: ["draft", "pending", "ready"] },
  { id: "in_progress", label: "In progress", statuses: ["in_progress"] },
  { id: "blocked", label: "Blocked", statuses: ["blocked"] },
  { id: "done", label: "Done", statuses: ["completed", "failed", "canceled"] },
];

/**
 * List-view group buckets — six groups in proposal order:
 * Active · Blocked · Stuck · Queued · Done · Failed.
 *
 * `stuck` carries no status mapping today (techspec MVP excludes the
 * `task.is_stuck` flag); the bucket is preserved so consumers can render it
 * the moment a backing signal lands without re-baselining the grouping API.
 */
export type TaskListGroupId = "active" | "blocked" | "stuck" | "queued" | "done" | "failed";

export interface TaskListGroupDefinition {
  id: TaskListGroupId;
  label: string;
  statuses: TaskStatus[];
}

const LIST_GROUPS: TaskListGroupDefinition[] = [
  { id: "active", label: "Active", statuses: ["in_progress"] },
  { id: "blocked", label: "Blocked", statuses: ["blocked"] },
  { id: "stuck", label: "Stuck", statuses: [] },
  { id: "queued", label: "Queued", statuses: ["ready", "pending", "draft"] },
  { id: "done", label: "Done", statuses: ["completed"] },
  { id: "failed", label: "Failed", statuses: ["failed", "canceled"] },
];

export interface TaskListGroupBucket {
  group: TaskListGroupDefinition;
  tasks: TaskListItem[];
}

export function getTaskListGroups(): TaskListGroupDefinition[] {
  return LIST_GROUPS;
}

export function resolveTaskListGroupId(status: TaskStatus | string): TaskListGroupId | null {
  for (const group of LIST_GROUPS) {
    if ((group.statuses as readonly string[]).includes(status)) {
      return group.id;
    }
  }
  return null;
}

/**
 * Partition the list into the six ordered group buckets. Mirrors the proposal
 * pattern (`docs/design/new-proposal/agh-refined-7.html:1147`): groups always
 * emit in canonical order, and callers decide how to render empty buckets.
 */
export function groupTasksForList(tasks: TaskListItem[]): TaskListGroupBucket[] {
  const buckets = new Map<TaskListGroupId, TaskListItem[]>();
  for (const group of LIST_GROUPS) {
    buckets.set(group.id, []);
  }

  for (const task of tasks) {
    const groupId = resolveTaskListGroupId(task.status);
    if (!groupId) {
      continue;
    }
    buckets.get(groupId)?.push(task);
  }

  return LIST_GROUPS.map(group => ({
    group,
    tasks: buckets.get(group.id) ?? [],
  }));
}

const MOCK_STATUS_ALIASES: Record<string, TaskKanbanColumnId> = {
  running: "in_progress",
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
