import { useQueries } from "@tanstack/react-query";

import { workspaceDetailOptions } from "@/systems/workspace";

export interface OsSpaceDetailModel {
  status: "loading" | "ready" | "error";
  agentNames: string[];
  agentCount: number;
  sessionCount: number;
}

export interface OsSpaceDetailsResult {
  byWorkspaceId: Map<string, OsSpaceDetailModel>;
  /** Total agents across all loaded spaces; null until every query resolved. */
  totalAgents: number | null;
}

/**
 * Per-workspace agent/session rollups for the Spaces overview cards, read
 * from the existing workspace detail endpoint (no invented metrics).
 */
export function useSpaceDetails(
  workspaceIds: readonly string[],
  { enabled }: { enabled: boolean }
): OsSpaceDetailsResult {
  const queries = useQueries({
    queries: workspaceIds.map(id => ({
      ...workspaceDetailOptions(id),
      enabled,
    })),
  });

  const byWorkspaceId = new Map<string, OsSpaceDetailModel>();
  let totalAgents: number | null = 0;
  workspaceIds.forEach((id, index) => {
    const query = queries[index];
    if (!query || query.isPending) {
      byWorkspaceId.set(id, { status: "loading", agentNames: [], agentCount: 0, sessionCount: 0 });
      totalAgents = null;
      return;
    }
    if (query.isError || !query.data) {
      byWorkspaceId.set(id, { status: "error", agentNames: [], agentCount: 0, sessionCount: 0 });
      totalAgents = null;
      return;
    }
    const agents = query.data.agents ?? [];
    const sessions = query.data.sessions ?? [];
    const agentNames = agents.flatMap(agent => (agent.name ? [agent.name] : []));
    byWorkspaceId.set(id, {
      status: "ready",
      agentNames,
      agentCount: agents.length,
      sessionCount: sessions.length,
    });
    if (totalAgents !== null) totalAgents += agents.length;
  });

  return { byWorkspaceId, totalAgents };
}
