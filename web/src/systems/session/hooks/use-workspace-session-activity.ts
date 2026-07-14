import { useQueries } from "@tanstack/react-query";

import type { WorkspacePayload } from "@/systems/workspace";

import { fetchSessions } from "../adapters/session-api";
import { getSessionDisplayTitle } from "../lib/session-display-title";
import { sessionKeys } from "../lib/query-keys";
import type { SessionsResponse } from "../types";

export interface WorkspaceSessionReturnTarget {
  sessionId: string;
  agentName: string;
  title: string;
}

export interface WorkspaceSessionActivity {
  count: number;
  returnTarget?: WorkspaceSessionReturnTarget;
}

export type WorkspaceSessionActivityMap = Record<string, WorkspaceSessionActivity>;

interface WorkspaceSessionActivityResult {
  data: SessionsResponse | undefined;
}

export function workspaceSessionActivityFromResults(
  workspaceIds: readonly string[],
  results: readonly WorkspaceSessionActivityResult[]
): WorkspaceSessionActivityMap {
  const activity: WorkspaceSessionActivityMap = {};
  workspaceIds.forEach((workspaceId, index) => {
    const data = results[index]?.data;
    const count = data?.page.total ?? 0;
    if (count <= 0) return;

    const latest = data?.sessions[0];
    activity[workspaceId] = {
      count,
      ...(latest
        ? {
            returnTarget: {
              sessionId: latest.id,
              agentName: latest.agent_name,
              title: getSessionDisplayTitle(latest),
            },
          }
        : {}),
    };
  });
  return activity;
}

export function useWorkspaceSessionActivity(
  workspaces: readonly WorkspacePayload[],
  enabled = true
): WorkspaceSessionActivityMap {
  const workspaceIds = workspaces.flatMap(workspace => {
    const workspaceId = workspace.id.trim();
    return workspaceId === "" ? [] : [workspaceId];
  });
  const results = useQueries({
    queries: workspaceIds.map(workspace => ({
      queryKey: sessionKeys.workspaceActivity(workspace),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        fetchSessions(
          {
            workspace,
            state: "active",
            type: "user",
            sort: "last_activity",
            limit: 1,
          },
          signal
        ),
      enabled: enabled && workspace !== "",
      staleTime: 2_000,
    })),
  });

  return workspaceSessionActivityFromResults(workspaceIds, results);
}
