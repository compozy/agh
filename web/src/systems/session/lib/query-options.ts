import { queryOptions } from "@tanstack/react-query";

import {
  fetchSession,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessionLedger,
  fetchSessionRecap,
  fetchSessionTranscript,
  fetchSessions,
  SessionLedgerUnavailableError,
} from "../adapters/session-api";
import type { FetchSessionEventsParams } from "../adapters/session-api";
import type { SessionState } from "../types";
import { sessionKeys } from "./query-keys";

const SESSION_LIVE_REFETCH_INTERVAL_MS = 5_000;
const SESSION_DETAIL_STALE_TIME_MS = 2_000;
const SESSION_TRANSCRIPT_STALE_TIME_MS = 10_000;
const SESSION_WARM_CACHE_GC_TIME_MS = 30 * 60 * 1_000;

/**
 * Session detail + transcript are the hot return path for `/agents/:name/sessions/:id`.
 * Keep them inactive for 30 minutes so tab restores and cross-route returns render from
 * cache immediately, while their short staleTime/background polling keeps live sessions fresh.
 */
const SESSION_WARM_CACHE_POLICY = {
  gcTime: SESSION_WARM_CACHE_GC_TIME_MS,
} as const;

/**
 * Live session states worth polling: while a session is `active|starting|stopping`
 * its transcript can still grow, so the read paths run a bounded self-heal refetch
 * (mirrors `sessionDetailOptions`). A `stopped` session is terminal; no polling.
 */
export function isLiveSessionState(state: SessionState | null | undefined): boolean {
  return state === "active" || state === "starting" || state === "stopping";
}

export function sessionsListOptions(workspace: string | null = null) {
  return queryOptions({
    queryKey: sessionKeys.list(workspace),
    queryFn: ({ signal }) => fetchSessions(workspace ?? undefined, signal),
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function sessionDetailOptions(workspace: string, id: string) {
  return queryOptions({
    queryKey: sessionKeys.detail(workspace, id),
    queryFn: ({ signal }) => fetchSession(workspace, id, signal),
    refetchInterval: query =>
      isLiveSessionState(query.state.data?.state) ? SESSION_LIVE_REFETCH_INTERVAL_MS : false,
    staleTime: SESSION_DETAIL_STALE_TIME_MS,
    ...SESSION_WARM_CACHE_POLICY,
    enabled: !!workspace && !!id,
  });
}

export function sessionEventsOptions(
  workspace: string,
  id: string,
  params?: FetchSessionEventsParams
) {
  return queryOptions({
    queryKey: sessionKeys.events(workspace, id),
    queryFn: ({ signal }) => fetchSessionEvents(workspace, id, params, signal),
    staleTime: 5_000,
    enabled: !!workspace && !!id,
  });
}

export function sessionHistoryOptions(workspace: string, id: string) {
  return queryOptions({
    queryKey: sessionKeys.history(workspace, id),
    queryFn: ({ signal }) => fetchSessionHistory(workspace, id, signal),
    staleTime: 10_000,
    enabled: !!workspace && !!id,
  });
}

export function sessionTranscriptOptions(
  workspace: string,
  id: string,
  sessionState?: SessionState | null
) {
  return queryOptions({
    queryKey: sessionKeys.transcript(workspace, id),
    queryFn: ({ signal }) => fetchSessionTranscript(workspace, id, signal),
    // Bounded self-heal: a transient transcript fetch failure (5xx, recorder churn,
    // daemon restart) recovers without navigation while the session is still live.
    refetchInterval: isLiveSessionState(sessionState) ? SESSION_LIVE_REFETCH_INTERVAL_MS : false,
    staleTime: SESSION_TRANSCRIPT_STALE_TIME_MS,
    ...SESSION_WARM_CACHE_POLICY,
    enabled: !!workspace && !!id,
  });
}

export function sessionRecapOptions(workspace: string, id: string, limit?: number) {
  return queryOptions({
    queryKey: sessionKeys.recap(workspace, id, limit),
    queryFn: ({ signal }) => fetchSessionRecap(workspace, id, limit, signal),
    staleTime: 10_000,
    enabled: !!workspace && !!id,
  });
}

export interface SessionLedgerQueryOptions {
  enabled?: boolean;
}

export function sessionLedgerOptions(
  workspace: string,
  id: string,
  options?: SessionLedgerQueryOptions
) {
  const enabled = (options?.enabled ?? true) && !!workspace && !!id;
  return queryOptions({
    queryKey: sessionKeys.ledger(workspace, id),
    queryFn: ({ signal }) => fetchSessionLedger(workspace, id, signal),
    staleTime: 10_000,
    enabled,
    retry: (failureCount, error) => {
      if (error instanceof SessionLedgerUnavailableError) {
        return false;
      }
      return failureCount < 1;
    },
  });
}
