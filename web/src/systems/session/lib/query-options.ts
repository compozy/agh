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

export function sessionDetailOptions(workspace: string, id: string) {
  return queryOptions({
    queryKey: sessionKeys.detail(workspace, id),
    queryFn: ({ signal }) => fetchSession(workspace, id, signal),
    refetchInterval: query => {
      const state = query.state.data?.state;
      return state === "active" || state === "starting" || state === "stopping" ? 5_000 : false;
    },
    staleTime: 2_000,
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

export function sessionTranscriptOptions(workspace: string, id: string) {
  return queryOptions({
    queryKey: sessionKeys.transcript(workspace, id),
    queryFn: ({ signal }) => fetchSessionTranscript(workspace, id, signal),
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
