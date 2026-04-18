import type {
  TaskDashboardFilter,
  TaskInboxFilter,
  TaskListFilter,
  TaskRunsFilter,
  TaskTimelineFilter,
} from "../types";

function normalizeText(value?: string | null): string {
  return typeof value === "string" ? value : "";
}

function normalizeFlag(value?: boolean): string {
  return value === undefined ? "" : value ? "1" : "0";
}

function normalizeNumber(value?: number): string {
  return value === undefined ? "" : String(value);
}

export const tasksKeys = {
  all: ["tasks"] as const,

  lists: () => [...tasksKeys.all, "list"] as const,
  list: (filters: TaskListFilter = {}) =>
    [
      ...tasksKeys.lists(),
      normalizeText(filters.scope),
      normalizeText(filters.workspace),
      normalizeText(filters.status),
      normalizeText(filters.priority),
      normalizeFlag(filters.include_drafts),
      normalizeText(filters.approval_state),
      normalizeText(filters.owner_kind),
      normalizeText(filters.owner_ref),
      normalizeText(filters.parent_task_id),
      normalizeText(filters.network_channel),
      normalizeText(filters.query),
      normalizeNumber(filters.limit),
    ] as const,

  details: () => [...tasksKeys.all, "detail"] as const,
  detail: (id: string) => [...tasksKeys.details(), id] as const,

  runsRoot: () => [...tasksKeys.all, "runs"] as const,
  runs: (id: string, filters: TaskRunsFilter = {}) =>
    [
      ...tasksKeys.runsRoot(),
      id,
      normalizeText(filters.status),
      normalizeText(filters.session_id),
      normalizeNumber(filters.limit),
    ] as const,

  timelineRoot: () => [...tasksKeys.all, "timeline"] as const,
  timeline: (id: string, filters: TaskTimelineFilter = {}) =>
    [
      ...tasksKeys.timelineRoot(),
      id,
      normalizeNumber(filters.after_sequence),
      normalizeNumber(filters.limit),
    ] as const,

  treeRoot: () => [...tasksKeys.all, "tree"] as const,
  tree: (id: string) => [...tasksKeys.treeRoot(), id] as const,

  runDetails: () => [...tasksKeys.all, "run-detail"] as const,
  runDetail: (runId: string) => [...tasksKeys.runDetails(), runId] as const,

  dashboard: (filters: TaskDashboardFilter = {}) =>
    [
      ...tasksKeys.all,
      "dashboard",
      normalizeText(filters.scope),
      normalizeText(filters.workspace),
      normalizeText(filters.owner_kind),
      normalizeText(filters.owner_ref),
      normalizeText(filters.network_channel),
      normalizeText(filters.origin_kind),
    ] as const,

  inbox: (filters: TaskInboxFilter = {}) =>
    [
      ...tasksKeys.all,
      "inbox",
      normalizeText(filters.scope),
      normalizeText(filters.workspace),
      normalizeText(filters.owner_kind),
      normalizeText(filters.owner_ref),
      normalizeText(filters.lane),
      normalizeFlag(filters.unread),
      normalizeText(filters.query),
      normalizeNumber(filters.limit),
    ] as const,

  triageRoot: () => [...tasksKeys.all, "triage"] as const,
};
