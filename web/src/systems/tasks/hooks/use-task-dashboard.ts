import { useQuery } from "@tanstack/react-query";

import { taskDashboardOptions } from "../lib/query-options";
import type { TaskDashboardFilter } from "../types";

interface QueryHookOptions {
  enabled?: boolean;
}

export function useTaskDashboard(
  filters: TaskDashboardFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery(taskDashboardOptions(filters, options.enabled ?? true));
}
