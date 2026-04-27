import { useMemo } from "react";

import { useSessions } from "@/systems/session";
import type { SessionPayload } from "@/systems/session";

interface UseAgentSessionsOptions {
  enabled?: boolean;
}

interface UseAgentSessionsResult {
  sessions: SessionPayload[];
  isLoading: boolean;
  isError: boolean;
}

function sessionRecencyTimestamp(session: SessionPayload): number {
  const candidates = [session.activity?.last_activity_at, session.updated_at, session.created_at];
  for (const value of candidates) {
    if (typeof value !== "string" || value.length === 0) continue;
    const ts = new Date(value).getTime();
    if (Number.isFinite(ts)) return ts;
  }
  return 0;
}

function compareByRecencyDesc(left: SessionPayload, right: SessionPayload): number {
  return sessionRecencyTimestamp(right) - sessionRecencyTimestamp(left);
}

export function useAgentSessions(
  workspaceId: string | null,
  agentName: string | undefined,
  options?: UseAgentSessionsOptions
): UseAgentSessionsResult {
  const enabled = options?.enabled ?? workspaceId !== null;
  const query = useSessions(workspaceId, { enabled });

  const sessions = useMemo<SessionPayload[]>(() => {
    if (!query.data || !agentName) return [];
    return query.data
      .filter(session => session.agent_name === agentName)
      .slice()
      .sort(compareByRecencyDesc);
  }, [query.data, agentName]);

  return {
    sessions,
    isLoading: query.isLoading,
    isError: query.isError,
  };
}
