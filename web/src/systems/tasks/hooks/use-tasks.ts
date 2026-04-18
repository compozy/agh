import { useQuery } from "@tanstack/react-query";

import { taskDetailOptions, taskRunsOptions, tasksListOptions } from "../lib/query-options";
import type { TaskListFilter, TaskRunsFilter } from "../types";

interface QueryHookOptions {
  enabled?: boolean;
}

export function useTasks(filters: TaskListFilter = {}, options: QueryHookOptions = {}) {
  return useQuery(tasksListOptions(filters, options.enabled ?? true));
}

export function useTask(id: string, options: QueryHookOptions = {}) {
  return useQuery(taskDetailOptions(id, options.enabled ?? true));
}

export function useTaskRuns(
  id: string,
  filters: TaskRunsFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery(taskRunsOptions(id, filters, options.enabled ?? true));
}
