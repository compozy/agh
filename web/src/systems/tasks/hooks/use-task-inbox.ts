import { useQuery } from "@tanstack/react-query";

import { taskInboxOptions } from "../lib/query-options";
import type { TaskInboxFilter } from "../types";

interface QueryHookOptions {
  enabled?: boolean;
}

export function useTaskInbox(filters: TaskInboxFilter = {}, options: QueryHookOptions = {}) {
  return useQuery(taskInboxOptions(filters, options.enabled ?? true));
}
