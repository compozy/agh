import { queryOptions } from "@tanstack/react-query";

import {
  fetchSession,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessionLedger,
  fetchSessionTranscript,
  fetchSessions,
  SessionLedgerUnavailableError,
} from "../adapters/session-api";
import type { FetchSessionEventsParams } from "../adapters/session-api";
import { sessionKeys } from "./query-keys";

export function sessionsListOptions(workspace: string | null = null) {
  return queryOptions({
    queryKey: sessionKeys.list(workspace),
    queryFn: ({ signal }) => fetchSessions(workspace ?? undefined, signal),
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function sessionDetailOptions(id: string) {
  return queryOptions({
    queryKey: sessionKeys.detail(id),
    queryFn: ({ signal }) => fetchSession(id, signal),
    refetchInterval: query => {
      const state = query.state.data?.state;
      return state === "active" || state === "starting" || state === "stopping" ? 5_000 : false;
    },
    staleTime: 2_000,
    enabled: !!id,
  });
}

export function sessionEventsOptions(id: string, params?: FetchSessionEventsParams) {
  return queryOptions({
    queryKey: sessionKeys.events(id),
    queryFn: ({ signal }) => fetchSessionEvents(id, params, signal),
    staleTime: 5_000,
    enabled: !!id,
  });
}

export function sessionHistoryOptions(id: string) {
  return queryOptions({
    queryKey: sessionKeys.history(id),
    queryFn: ({ signal }) => fetchSessionHistory(id, signal),
    staleTime: 10_000,
    enabled: !!id,
  });
}

export function sessionTranscriptOptions(id: string) {
  return queryOptions({
    queryKey: sessionKeys.transcript(id),
    queryFn: ({ signal }) => fetchSessionTranscript(id, signal),
    staleTime: 10_000,
    enabled: !!id,
  });
}

export interface SessionLedgerQueryOptions {
  enabled?: boolean;
}

export function sessionLedgerOptions(id: string, options?: SessionLedgerQueryOptions) {
  const enabled = (options?.enabled ?? true) && !!id;
  return queryOptions({
    queryKey: sessionKeys.ledger(id),
    queryFn: ({ signal }) => fetchSessionLedger(id, signal),
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
