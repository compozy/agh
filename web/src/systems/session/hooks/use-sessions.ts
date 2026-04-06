import { useQuery } from "@tanstack/react-query";

import { sessionDetailOptions, sessionsListOptions } from "../lib/query-options";

interface UseSessionsOptions {
  enabled?: boolean;
}

export function useSessions(workspace: string | null = null, options?: UseSessionsOptions) {
  return useQuery({
    ...sessionsListOptions(workspace),
    enabled: options?.enabled ?? true,
  });
}

export function useSession(id: string) {
  return useQuery(sessionDetailOptions(id));
}
