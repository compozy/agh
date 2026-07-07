import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { LOOP_RUN_EVENT_KINDS, LOOP_RUN_LIFECYCLE_EVENT_KINDS } from "@/generated/loop-enums";

import { buildLoopStreamUrl } from "../adapters/loops-api";
import { loopsKeys } from "../lib/query-keys";
import type { LoopRunEventFrame, LoopRunEventKind } from "../types";

interface LoopStreamEventSource {
  addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
  removeEventListener?: (type: string, listener: EventListenerOrEventListenerObject) => void;
  close: () => void;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
}

type LoopStreamEventSourceFactory = (url: string) => LoopStreamEventSource;

interface UseLoopStreamOptions {
  enabled?: boolean;
  afterSequence?: number;
  eventSourceFactory?: LoopStreamEventSourceFactory;
  onEvent?: (payload: LoopRunEventFrame) => void;
  onError?: (error: unknown) => void;
}

// AGH Loop SSE emits named events via `event: <kind>` from the run-events writer
// (internal/daemon). EventSource routes named SSE frames to listeners registered
// with addEventListener("<kind>", ...); they never reach onmessage, which only
// handles unnamed `message` frames. Keep this list aligned with the enumerated
// LoopRunEventKind contract (techspec §observability, L-017 named-listener rule):
// an unenumerated kind silently never renders.
const LOOP_STREAM_EVENT_TYPES = LOOP_RUN_EVENT_KINDS satisfies readonly LoopRunEventKind[];

// Lifecycle kinds mutate durable run state, so each one invalidates the run detail +
// runs list (daemon truth wins). The high-frequency display frames `token_tick` and
// `channel_msg` are applied locally via `onEvent` (the run-page meter/timeline store,
// task 20) and never invalidate — otherwise every tick would refetch the workspace
// runs list. The catalog's 30d aggregates refresh on their own interval, not per frame.
const LOOP_LIFECYCLE_EVENT_KINDS = new Set<LoopRunEventKind>(
  LOOP_RUN_LIFECYCLE_EVENT_KINDS satisfies readonly LoopRunEventKind[]
);

function isLifecycleKind(kind: string): boolean {
  return LOOP_LIFECYCLE_EVENT_KINDS.has(kind as LoopRunEventKind);
}

function defaultEventSourceFactory(url: string): LoopStreamEventSource {
  return new EventSource(url);
}

type QueryClient = ReturnType<typeof useQueryClient>;

function invalidateLoopRunQueries(queryClient: QueryClient, workspaceId: string, runId: string) {
  void queryClient.invalidateQueries({ queryKey: loopsKeys.runDetail(workspaceId, runId) });
  void queryClient.invalidateQueries({ queryKey: loopsKeys.runsByWorkspace(workspaceId) });
}

/**
 * Subscribes to a Loop run's SSE event stream, mirroring `useTaskStream`: named
 * listeners for every enumerated kind, `onEvent` on each frame, `afterSequence`
 * resume, and query invalidation on the lifecycle kinds (`token_tick`/`channel_msg`
 * are display-only, applied via `onEvent`, never invalidating). `onEvent`/`onError`
 * are stabilized through refs so an inline (non-memoized) callback never tears down
 * and reopens the EventSource; only the workspace, run, resume seed, or factory
 * identity reopen the stream.
 */
export function useLoopStream(
  workspaceId: string,
  runId: string,
  options: UseLoopStreamOptions = {}
) {
  const enabled = options.enabled ?? true;
  const eventSourceFactory = options.eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(options.eventSourceFactory);
  const afterSequence = options.afterSequence;
  const queryClient = useQueryClient();
  const trimmedWorkspace = workspaceId.trim();
  const trimmedRun = runId.trim();

  const onEventRef = useRef(options.onEvent);
  const onErrorRef = useRef(options.onError);
  onEventRef.current = options.onEvent;
  onErrorRef.current = options.onError;

  useEffect(() => {
    if (
      !enabled ||
      trimmedWorkspace === "" ||
      trimmedRun === "" ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const url = buildLoopStreamUrl(trimmedWorkspace, trimmedRun, {
      after_sequence: afterSequence === undefined ? undefined : String(afterSequence),
    });
    const source = eventSourceFactory(url);

    const handleFrame = (event: MessageEvent) => {
      if (typeof event.data !== "string") {
        return;
      }
      // Only genuine parse failures reach onError; a throw inside the consumer's
      // onEvent sink must propagate, not be misreported as a stream-parse error.
      let payload: LoopRunEventFrame;
      try {
        payload = JSON.parse(event.data) as LoopRunEventFrame;
      } catch (error) {
        if (onErrorRef.current) {
          onErrorRef.current(error);
        } else {
          console.error("Failed to parse loop stream payload", error);
        }
        return;
      }
      // Named frames carry the kind as event.type; the defensive onmessage frame
      // ("message") falls back to the parsed payload kind.
      const kind = event.type !== "message" ? event.type : (payload.kind ?? "");
      if (isLifecycleKind(kind)) {
        invalidateLoopRunQueries(queryClient, trimmedWorkspace, trimmedRun);
      }
      onEventRef.current?.(payload);
    };

    const handleError = (event: Event) => {
      if (onErrorRef.current) {
        onErrorRef.current(event);
      } else {
        console.error("Loop stream failed", event);
      }
    };

    source.onmessage = handleFrame;
    source.onerror = handleError;

    const namedListener = handleFrame as EventListener;
    for (const type of LOOP_STREAM_EVENT_TYPES) {
      source.addEventListener(type, namedListener);
    }

    return () => {
      if (source.removeEventListener) {
        for (const type of LOOP_STREAM_EVENT_TYPES) {
          source.removeEventListener(type, namedListener);
        }
      }
      source.onmessage = null;
      source.onerror = null;
      source.close();
    };
  }, [
    enabled,
    trimmedWorkspace,
    trimmedRun,
    afterSequence,
    eventSourceFactory,
    hasCustomFactory,
    queryClient,
  ]);
}

export type { LoopStreamEventSource, LoopStreamEventSourceFactory, UseLoopStreamOptions };
