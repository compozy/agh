import type { TaskListFilter } from "@/systems/tasks";

import type { ActiveTaskScopeFilter } from "./workspace-scope-filter";

export const DEFAULT_TASK_LIST_LIMIT = 50;

export function defaultTaskCatalogFilter(scope: ActiveTaskScopeFilter): TaskListFilter {
  return {
    scope: scope.scope,
    workspace: scope.workspace,
    include_drafts: true,
    limit: DEFAULT_TASK_LIST_LIMIT,
    sort: "recent",
  };
}
