export const TASK_DETAIL_TABS = ["overview", "runs", "activity"] as const;
export type TaskDetailTab = (typeof TASK_DETAIL_TABS)[number];

/** Optional URL params — defaults applied via `resolveTaskDetailSearch`. */
export interface TaskDetailSearch {
  tab?: TaskDetailTab;
}

export interface ResolvedTaskDetailSearch {
  tab: TaskDetailTab;
}

function parseEnum<T extends string>(value: unknown, allowed: readonly T[], fallback: T): T {
  return typeof value === "string" && (allowed as readonly string[]).includes(value)
    ? (value as T)
    : fallback;
}

export function validateTaskDetailSearch(search: Record<string, unknown>): TaskDetailSearch {
  return {
    tab: parseEnum(search.tab, TASK_DETAIL_TABS, "overview"),
  };
}

export function resolveTaskDetailSearch(search: TaskDetailSearch): ResolvedTaskDetailSearch {
  return {
    tab: search.tab ?? "overview",
  };
}
