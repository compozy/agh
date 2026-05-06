import { useQuery } from "@tanstack/react-query";

import {
  sessionDetailOptions,
  sessionLedgerOptions,
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

export function useSession(id: string) {
  return useQuery(sessionDetailOptions(id));
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
export function useSessionLedger(id: string, options?: UseSessionLedgerOptions) {
  return useQuery(sessionLedgerOptions(id, options));
}
