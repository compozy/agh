import { useEffect, useMemo, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { buildSessionStreamUrl } from "../adapters/session-api";
import { normalizeTranscriptMessages } from "../lib/message-schemas";
import { sessionKeys } from "../lib/query-keys";
import {
  isLiveSessionState,
  sessionDetailOptions,
  sessionTranscriptOptions,
} from "../lib/query-options";
import {
  computeStableThreadMessages,
  EMPTY_STABLE_THREAD_MESSAGES,
  type StableThreadMessagesState,
} from "../lib/session-thread-repository";
import {
  formatSessionDebugError,
  recordSessionDebugEvent,
  SESSION_DEBUG_EVENTS,
} from "../lib/session-observability";
import type { SessionTranscriptThreadStatus } from "../lib/session-transcript-thread-context-value";
import type {
  SessionEventPayload,
  SessionMessage,
  SessionPayload,
  TranscriptDeltaPayload,
  TranscriptSnapshotPayload,
} from "../types";
import {
  numberFromEventID,
  parseSessionStreamPayload,
  sortMessagesByKnownSequence,
  terminalFailureMessage,
  upsertTranscriptMessage,
} from "./session-live-tail-helpers";

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

const TRANSCRIPT_SNAPSHOT_EVENT = "transcript_snapshot";
const TRANSCRIPT_DELTA_EVENT = "transcript_delta";
const SESSION_STOPPED_EVENT = "session_stopped";
const SESSION_DONE_EVENT = "done";
const STREAM_ERROR_EVENT = "error";
const RECONNECT_BASE_DELAY_MS = 250;
const RECONNECT_MAX_DELAY_MS = 4_000;
type TranscriptApplyFrame = "snapshot" | "delta";

function defaultEventSourceFactory(url: string): SessionStreamEventSource {
  return new EventSource(url);
}

export function useSessionLiveTail({
  workspaceId,
  sessionId,
  eventSourceFactory,
}: UseSessionLiveTailOptions) {
  const queryClient = useQueryClient();
  const cursorRef = useRef(0);
  const epochRef = useRef<number | null>(null);
  const messageSequenceRef = useRef(new Map<string, number>());
  const stableMessagesRef = useRef<StableThreadMessagesState>(EMPTY_STABLE_THREAD_MESSAGES);
  const reloadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastTranscriptErrorRef = useRef<unknown>(null);
  const sourceFactory = eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(eventSourceFactory);
  // Observe the live session state (shared cache with the route detail query) so the
  // transcript self-heal refetch only polls while the session is active|starting|stopping.
  const sessionState = useQuery(sessionDetailOptions(workspaceId, sessionId)).data?.state;
  const streamShouldOpen = sessionState == null || isLiveSessionState(sessionState);
  const transcriptQuery = useQuery(sessionTranscriptOptions(workspaceId, sessionId, sessionState));
  const transcriptMessages = transcriptQuery.data;
  const refetchTranscript = transcriptQuery.refetch;
  const refetchTranscriptRef = useRef(refetchTranscript);
  const transcriptStatus: SessionTranscriptThreadStatus = transcriptQuery.isPending
    ? "pending"
    : transcriptQuery.isError
      ? "error"
      : "success";
  // Preserve read-model identity across updates: reuse each unchanged message's
  // `ThreadMessage` so a single-message delta re-allocates only that row's object,
  // never the whole settled thread (task 39.3 — read-model structural sharing).
  const readonlyMessages = useMemo(() => {
    if (!transcriptMessages) {
      stableMessagesRef.current = EMPTY_STABLE_THREAD_MESSAGES;
      return EMPTY_STABLE_THREAD_MESSAGES.result;
    }
    const stable = computeStableThreadMessages(transcriptMessages, stableMessagesRef.current);
    stableMessagesRef.current = stable;
    return stable.result;
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
    refetchTranscriptRef.current = refetchTranscript;
  }, [refetchTranscript]);

  useEffect(() => {
    if (!transcriptQuery.isError) {
      lastTranscriptErrorRef.current = null;
      return;
    }
    if (lastTranscriptErrorRef.current === transcriptQuery.error) {
      return;
    }
    lastTranscriptErrorRef.current = transcriptQuery.error;
    recordSessionDebugEvent(SESSION_DEBUG_EVENTS.transcriptFetchFailed, {
      cursor: cursorRef.current,
      error: formatSessionDebugError(transcriptQuery.error),
      session_id: sessionId,
      session_state: sessionState ?? "unknown",
      workspace_id: workspaceId,
    });
  }, [sessionId, sessionState, transcriptQuery.error, transcriptQuery.isError, workspaceId]);

  useEffect(() => {
    // Decoupled from the transcript fetch outcome: the stream opens as soon as the
    // ids are known so live recovery + the SSE-driven self-heal refetch proceed even
    // when the initial REST transcript fetch fails or is slow.
    if (
      workspaceId.trim() === "" ||
      sessionId.trim() === "" ||
      !streamShouldOpen ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const invalidateSessionSurfaces = () => {
      void queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(workspaceId, sessionId),
        exact: true,
      });
      void queryClient.invalidateQueries({
        queryKey: sessionKeys.history(workspaceId, sessionId),
        exact: true,
      });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    };

    const scheduleSurfaceRefresh = () => {
      if (reloadTimerRef.current) {
        clearTimeout(reloadTimerRef.current);
      }
      reloadTimerRef.current = setTimeout(() => {
        reloadTimerRef.current = null;
        invalidateSessionSurfaces();
      }, 120);
    };

    const transcriptQueryKey = sessionKeys.transcript(workspaceId, sessionId);
    let disposed = false;
    let reconnectAttempt = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let source: SessionStreamEventSource | null = null;
    let detachSourceListeners: (() => void) | null = null;
    let transcriptApplyQueue = Promise.resolve();

    const clearReconnectTimer = () => {
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };

    const closeCurrentSource = (reason: string) => {
      detachSourceListeners?.();
      detachSourceListeners = null;
      if (!source) {
        return;
      }
      const cursor = cursorRef.current;
      source.onmessage = null;
      source.onerror = null;
      source.close();
      source = null;
      recordSessionDebugEvent(SESSION_DEBUG_EVENTS.sseClose, {
        cursor,
        reason,
        session_id: sessionId,
        workspace_id: workspaceId,
      });
    };

    const resetReconnectBackoff = () => {
      reconnectAttempt = 0;
    };

    const scheduleReconnect = () => {
      if (disposed) {
        return;
      }
      clearReconnectTimer();
      const cursor = cursorRef.current;
      const delay = Math.min(
        RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttempt,
        RECONNECT_MAX_DELAY_MS
      );
      reconnectAttempt += 1;
      closeCurrentSource("reconnect");
      recordSessionDebugEvent(SESSION_DEBUG_EVENTS.sseReconnect, {
        attempt: reconnectAttempt,
        cursor,
        delay_ms: delay,
        session_id: sessionId,
        workspace_id: workspaceId,
      });
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        openSource();
      }, delay);
    };

    const recordTranscriptApplyFailure = (
      frame: TranscriptApplyFrame,
      sequence: number,
      error: unknown
    ) => {
      recordSessionDebugEvent(SESSION_DEBUG_EVENTS.transcriptApplyFailed, {
        cursor: cursorRef.current,
        error: formatSessionDebugError(error),
        frame,
        sequence,
        session_id: sessionId,
        workspace_id: workspaceId,
      });
      void refetchTranscriptRef.current();
    };

    const enqueueTranscriptApply = (
      frame: TranscriptApplyFrame,
      sequence: number,
      apply: () => Promise<void>
    ) => {
      transcriptApplyQueue = transcriptApplyQueue.then(async () => {
        if (disposed) {
          return;
        }
        try {
          await apply();
        } catch (error) {
          recordTranscriptApplyFailure(frame, sequence, error);
        }
      });
    };

    const applySnapshot = (event: MessageEvent) => {
      const payload = parseSessionStreamPayload<TranscriptSnapshotPayload>(event);
      if (!payload) {
        return;
      }
      resetReconnectBackoff();
      const eventID = numberFromEventID(event.lastEventId);
      const previousCursor = cursorRef.current;
      const epochChanged = epochRef.current !== null && epochRef.current !== payload.epoch;
      const backwardsMax = payload.max_sequence < previousCursor;
      const sequence = Math.max(payload.max_sequence, eventID ?? 0);
      if (epochChanged || backwardsMax) {
        cursorRef.current = sequence;
      } else if (sequence > cursorRef.current) {
        cursorRef.current = sequence;
      }
      epochRef.current = payload.epoch;
      enqueueTranscriptApply("snapshot", sequence, async () => {
        const messages = await normalizeTranscriptMessages(
          payload.entries.map(entry => entry.message)
        );
        const incomingSequences = new Map<string, number>();
        for (const [index, message] of messages.entries()) {
          const entry = payload.entries[index];
          if (entry) {
            incomingSequences.set(message.id, entry.sequence);
          }
        }
        const incomingIDs = new Set(incomingSequences.keys());
        queryClient.setQueryData<SessionMessage[]>(transcriptQueryKey, existing => {
          if (epochChanged || backwardsMax) {
            messageSequenceRef.current = incomingSequences;
            return sortMessagesByKnownSequence(messages, incomingSequences);
          }
          const nextSequences = new Map<string, number>();
          const retained = (existing ?? []).filter(message => {
            if (incomingIDs.has(message.id)) {
              return false;
            }
            const existingSequence = messageSequenceRef.current.get(message.id);
            if (payload.reset_below) {
              if (existingSequence === undefined || existingSequence < payload.min_sequence) {
                return false;
              }
              nextSequences.set(message.id, existingSequence);
              return true;
            }
            if (
              existingSequence !== undefined &&
              existingSequence >= payload.min_sequence &&
              existingSequence <= payload.max_sequence
            ) {
              return false;
            }
            if (existingSequence !== undefined) {
              nextSequences.set(message.id, existingSequence);
            }
            return true;
          });
          for (const [id, incomingSequence] of incomingSequences) {
            nextSequences.set(id, incomingSequence);
          }
          messageSequenceRef.current = nextSequences;
          return sortMessagesByKnownSequence([...retained, ...messages], nextSequences);
        });
      });
      invalidateSessionSurfaces();
    };

    const applyDelta = (event: MessageEvent) => {
      const payload = parseSessionStreamPayload<TranscriptDeltaPayload>(event);
      if (!payload) {
        return;
      }
      resetReconnectBackoff();
      const eventID = numberFromEventID(event.lastEventId);
      const previousCursor = cursorRef.current;
      const sequence = Math.max(payload.sequence, payload.entry.sequence, eventID ?? 0);
      if (previousCursor > 0 && sequence > previousCursor + 1) {
        recordSessionDebugEvent(SESSION_DEBUG_EVENTS.gapRecoveryTriggered, {
          cursor: previousCursor,
          missing_count: sequence - previousCursor - 1,
          next_sequence: sequence,
          session_id: sessionId,
          workspace_id: workspaceId,
        });
        void refetchTranscriptRef.current();
      }
      if (sequence > cursorRef.current) {
        cursorRef.current = sequence;
      }
      const epochChanged = epochRef.current !== null && epochRef.current !== payload.epoch;
      epochRef.current = payload.epoch;
      enqueueTranscriptApply("delta", sequence, async () => {
        const messages = await normalizeTranscriptMessages([payload.entry.message]);
        const message = messages[0];
        if (!message) {
          return;
        }
        if (epochChanged) {
          messageSequenceRef.current = new Map([[message.id, payload.entry.sequence]]);
        } else {
          messageSequenceRef.current.set(message.id, payload.entry.sequence);
        }
        queryClient.setQueryData<SessionMessage[]>(transcriptQueryKey, existing =>
          epochChanged ? [message] : upsertTranscriptMessage(existing, message)
        );
      });
      scheduleSurfaceRefresh();
    };

    const handleTerminalEvent = (event: MessageEvent) => {
      const payload = parseSessionStreamPayload<SessionEventPayload>(event);
      const sequence = Math.max(payload?.sequence ?? 0, numberFromEventID(event.lastEventId) ?? 0);
      if (sequence > cursorRef.current) {
        cursorRef.current = sequence;
      }
      if (payload) {
        queryClient.setQueryData<SessionPayload>(
          sessionKeys.detail(workspaceId, sessionId),
          existing => {
            if (!existing) {
              return existing;
            }
            return {
              ...existing,
              state: "stopped",
              stop_reason: payload.stop_reason ?? existing.stop_reason,
              stop_detail: payload.stop_detail ?? existing.stop_detail,
              failure: payload.failure ?? existing.failure,
              updated_at: payload.timestamp || existing.updated_at,
            };
          }
        );
        const failureMessage = terminalFailureMessage(payload, sessionId);
        if (failureMessage) {
          messageSequenceRef.current.set(failureMessage.id, sequence);
          queryClient.setQueryData<SessionMessage[]>(transcriptQueryKey, existing =>
            upsertTranscriptMessage(existing, failureMessage)
          );
        }
      }
      clearReconnectTimer();
      closeCurrentSource("terminal");
      invalidateSessionSurfaces();
    };

    const handleError = () => {
      scheduleSurfaceRefresh();
      scheduleReconnect();
    };

    const snapshotListener = applySnapshot as EventListener;
    const deltaListener = applyDelta as EventListener;
    const terminalListener = handleTerminalEvent as EventListener;
    const streamErrorListener = handleError as EventListener;

    function openSource() {
      if (disposed) {
        return;
      }
      const cursor = cursorRef.current;
      const nextSource = sourceFactory(buildSessionStreamUrl(workspaceId, sessionId, cursor));
      source = nextSource;
      recordSessionDebugEvent(SESSION_DEBUG_EVENTS.sseOpen, {
        cursor,
        session_id: sessionId,
        workspace_id: workspaceId,
      });
      nextSource.onmessage = null;
      nextSource.onerror = handleError;
      nextSource.addEventListener(TRANSCRIPT_SNAPSHOT_EVENT, snapshotListener);
      nextSource.addEventListener(TRANSCRIPT_DELTA_EVENT, deltaListener);
      nextSource.addEventListener(SESSION_STOPPED_EVENT, terminalListener);
      nextSource.addEventListener(SESSION_DONE_EVENT, terminalListener);
      nextSource.addEventListener(STREAM_ERROR_EVENT, streamErrorListener);
      detachSourceListeners = () => {
        if (nextSource.removeEventListener) {
          nextSource.removeEventListener(TRANSCRIPT_SNAPSHOT_EVENT, snapshotListener);
          nextSource.removeEventListener(TRANSCRIPT_DELTA_EVENT, deltaListener);
          nextSource.removeEventListener(SESSION_STOPPED_EVENT, terminalListener);
          nextSource.removeEventListener(SESSION_DONE_EVENT, terminalListener);
          nextSource.removeEventListener(STREAM_ERROR_EVENT, streamErrorListener);
        }
      };
    }

    openSource();

    return () => {
      disposed = true;
      clearReconnectTimer();
      closeCurrentSource("cleanup");
    };
  }, [hasCustomFactory, queryClient, sessionId, sourceFactory, streamShouldOpen, workspaceId]);

  return {
    messages: readonlyMessages,
    status: transcriptStatus,
    isPending: transcriptQuery.isPending,
    isError: transcriptQuery.isError,
    error: transcriptQuery.error,
    retry: () => {
      void refetchTranscript();
    },
  };
}

export type { SessionStreamEventSource, SessionStreamEventSourceFactory };
