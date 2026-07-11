import { useSessions } from "@/systems/session";
import type { SessionPayload } from "@/systems/session";

interface UseAgentSessionsOptions {
  enabled?: boolean;
}

interface UseAgentSessionsResult {
  sessions: SessionPayload[];
  total: number;
  activeTotal: number;
  resumableTotal: number;
  lastActivityAt: string | null;
  hasMore: boolean;
  isLoadingMore: boolean;
  loadMore: () => void;
  isLoading: boolean;
  isError: boolean;
}

export function useAgentSessions(
  workspaceId: string | null,
  agentName: string | undefined,
  options?: UseAgentSessionsOptions
): UseAgentSessionsResult {
  const enabled = (options?.enabled ?? true) && Boolean(workspaceId) && Boolean(agentName);
  const sessionsQuery = useSessions(workspaceId, {
    enabled,
    filters: { agent: agentName, sort: "last_activity" },
  });
  const activeQuery = useSessions(workspaceId, {
    enabled,
    filters: { agent: agentName, state: "active", limit: 1 },
  });
  const resumableQuery = useSessions(workspaceId, {
    enabled,
    filters: { agent: agentName, resumable: true, limit: 1 },
  });
  const latestSession = sessionsQuery.data?.[0];

  return {
    sessions: sessionsQuery.data ?? [],
    total: sessionsQuery.total,
    activeTotal: activeQuery.total,
    resumableTotal: resumableQuery.total,
    lastActivityAt: latestSession?.activity?.last_activity_at ?? latestSession?.updated_at ?? null,
    hasMore: sessionsQuery.hasNextPage,
    isLoadingMore: sessionsQuery.isFetchingNextPage,
    loadMore: () => {
      void sessionsQuery.fetchNextPage();
    },
    isLoading: sessionsQuery.isLoading || activeQuery.isLoading || resumableQuery.isLoading,
    isError: sessionsQuery.isError || activeQuery.isError || resumableQuery.isError,
  };
}
