import { queryOptions } from "@tanstack/react-query";

import {
  fetchSession,
  fetchSessionEvents,
  fetchSessionHistory,
  fetchSessionTranscript,
  fetchSessions,
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
    staleTime: 5_000,
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
