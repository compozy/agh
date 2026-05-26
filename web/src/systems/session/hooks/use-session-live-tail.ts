import { useEffect, useMemo, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { buildSessionStreamUrl } from "../adapters/session-api";
import { sessionKeys } from "../lib/query-keys";
import { sessionTranscriptOptions } from "../lib/query-options";
import { toReadonlyThreadMessages } from "../lib/session-thread-repository";
import type { SessionEventPayload } from "../types";

interface SessionStreamEventSource {
  addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
  removeEventListener?: (type: string, listener: EventListenerOrEventListenerObject) => void;
  close: () => void;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
}

type SessionStreamEventSourceFactory = (url: string) => SessionStreamEventSource;

interface UseSessionLiveTailOptions {
  workspaceId: string;
  sessionId: string;
  eventSourceFactory?: SessionStreamEventSourceFactory;
}

const SESSION_STREAM_EVENT_TYPES = [
  "user_message",
  "synthetic_reentry",
  "agent_message",
  "thought",
  "tool_call",
  "tool_result",
  "plan",
  "permission",
  "usage",
  "system",
  "runtime_progress",
  "runtime_warning",
  "done",
  "error",
  "session_stopped",
  "transcript_marker.created",
  "transcript_marker.redacted",
] as const;

function defaultEventSourceFactory(url: string): SessionStreamEventSource {
  return new EventSource(url);
}

function numberFromEventID(value: string): number | null {
  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return null;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) ? parsed : null;
}

function parseSessionStreamPayload(event: MessageEvent): SessionEventPayload | null {
  if (typeof event.data !== "string" || event.data.trim().length === 0) {
    return null;
  }
  try {
    return JSON.parse(event.data) as SessionEventPayload;
  } catch {
    return null;
  }
}

export function useSessionLiveTail({
  workspaceId,
  sessionId,
  eventSourceFactory,
}: UseSessionLiveTailOptions) {
  const queryClient = useQueryClient();
  const cursorRef = useRef(0);
  const reloadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sourceFactory = eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(eventSourceFactory);
  const transcriptQuery = useQuery({
    ...sessionTranscriptOptions(workspaceId, sessionId),
    staleTime: 0,
    refetchOnMount: "always",
  });
  const transcriptMessages = transcriptQuery.data;
  const transcriptReady = transcriptQuery.isSuccess;
  const refetchTranscript = transcriptQuery.refetch;
  const readonlyMessages = useMemo(() => {
    return transcriptMessages ? toReadonlyThreadMessages(transcriptMessages) : [];
  }, [transcriptMessages]);

  useEffect(() => {
    return () => {
      if (reloadTimerRef.current) {
        clearTimeout(reloadTimerRef.current);
        reloadTimerRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (
      !transcriptReady ||
      workspaceId.trim() === "" ||
      sessionId.trim() === "" ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const reloadTranscript = () => {
      void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(workspaceId, sessionId) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.history(workspaceId, sessionId) });
      void queryClient.invalidateQueries({
        queryKey: sessionKeys.transcript(workspaceId, sessionId),
      });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
      void refetchTranscript();
    };

    const scheduleReload = () => {
      if (reloadTimerRef.current) {
        clearTimeout(reloadTimerRef.current);
      }
      reloadTimerRef.current = setTimeout(() => {
        reloadTimerRef.current = null;
        reloadTranscript();
      }, 120);
    };

    const source = sourceFactory(buildSessionStreamUrl(workspaceId, sessionId, cursorRef.current));

    const handleMessage = (event: MessageEvent) => {
      const payload = parseSessionStreamPayload(event);
      const eventID = numberFromEventID(event.lastEventId);
      const sequence = payload?.sequence ?? eventID ?? 0;
      const hasGap = sequence > 0 && cursorRef.current > 0 && sequence > cursorRef.current + 1;
      if (sequence > cursorRef.current) {
        cursorRef.current = sequence;
      }
      scheduleReload();
      if (hasGap) {
        reloadTranscript();
      }
    };

    const handleError = () => {
      scheduleReload();
    };

    source.onmessage = handleMessage;
    source.onerror = handleError;

    const namedListener = handleMessage as EventListener;
    for (const type of SESSION_STREAM_EVENT_TYPES) {
      source.addEventListener(type, namedListener);
    }

    return () => {
      if (source.removeEventListener) {
        for (const type of SESSION_STREAM_EVENT_TYPES) {
          source.removeEventListener(type, namedListener);
        }
      }
      source.onmessage = null;
      source.onerror = null;
      source.close();
    };
  }, [
    hasCustomFactory,
    queryClient,
    refetchTranscript,
    sessionId,
    sourceFactory,
    transcriptReady,
    workspaceId,
  ]);

  return { messages: readonlyMessages };
}

export type { SessionStreamEventSource, SessionStreamEventSourceFactory };
