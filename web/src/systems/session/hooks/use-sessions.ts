import { useQuery } from "@tanstack/react-query";

import { useActiveWorkspace } from "@/systems/workspace";

import {
  sessionDetailOptions,
  sessionLedgerOptions,
  sessionRecapOptions,
  sessionWorkspaceResolutionOptions,
  sessionsListOptions,
} from "../lib/query-options";

interface UseSessionsOptions {
  enabled?: boolean;
}

export function useSessions(workspace: string | null = null, options?: UseSessionsOptions) {
  return useQuery({
    ...sessionsListOptions(workspace),
    enabled: options?.enabled ?? true,
  });
}

export function useSession(id: string, workspace?: string | null) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = workspace ?? activeWorkspaceId ?? "";
  return useQuery(sessionDetailOptions(workspaceId, id));
}

export function useSessionById(id: string) {
  const workspaceQuery = useQuery(sessionWorkspaceResolutionOptions(id));
  const workspaceId = workspaceQuery.data ?? "";
  const detailQuery = useQuery(sessionDetailOptions(workspaceId, id));
  const error = workspaceQuery.error ?? detailQuery.error ?? null;

  return {
    ...detailQuery,
    error,
    isError: workspaceQuery.isError || detailQuery.isError,
    isLoading: workspaceQuery.isLoading || (workspaceId !== "" && detailQuery.isLoading),
    isPending: workspaceQuery.isPending || (workspaceId !== "" && detailQuery.isPending),
  };
}

export interface UseSessionLedgerOptions {
  enabled?: boolean;
}

/**
 * The forensic ledger only materializes after `OnSessionEnd`, so the caller
 * must gate this query on `session.state === "stopped"`. Calling it earlier
 * causes a 404 path that lingers as the cached state and prevents the query
 * from naturally fetching when the session later transitions to stopped.
 */
export function useSessionLedger(
  id: string,
  workspace?: string | null,
  options?: UseSessionLedgerOptions
) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = workspace ?? activeWorkspaceId ?? "";
  return useQuery(sessionLedgerOptions(workspaceId, id, options));
}

export function useSessionRecap(id: string, workspace?: string | null, limit?: number) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = workspace ?? activeWorkspaceId ?? "";
  return useQuery(sessionRecapOptions(workspaceId, id, limit));
}
