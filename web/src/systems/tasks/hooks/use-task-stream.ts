import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { buildTaskStreamUrl } from "../adapters/tasks-api";
import { tasksKeys } from "../lib/query-keys";
import type { TaskStreamFilter, TaskStreamPayload } from "../types";

interface TaskStreamEventSource {
  addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
  removeEventListener?: (type: string, listener: EventListenerOrEventListenerObject) => void;
  close: () => void;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
}

type TaskStreamEventSourceFactory = (url: string) => TaskStreamEventSource;

interface UseTaskStreamOptions {
  enabled?: boolean;
  afterSequence?: number;
  filters?: TaskStreamFilter;
  eventSourceFactory?: TaskStreamEventSourceFactory;
  onEvent?: (payload: TaskStreamPayload) => void;
  onError?: (error: unknown) => void;
}

// AGH task SSE emits named events via `event: <type>` from internal/api/core/sse.go
// (WriteTaskStreamEvent sets Name = event.Type). EventSource routes named SSE events
// to listeners registered with addEventListener("<type>", ...); they never reach
// onmessage, which only handles unnamed `message` frames. Keep this list aligned with
// the canonical task event types emitted by internal/task/manager.go and the review /
// notification surfaces in internal/api.
const TASK_STREAM_EVENT_TYPES = [
  "task.created",
  "task.updated",
  "task.published",
  "task.approved",
  "task.rejected",
  "task.canceled",
  "task.child_created",
  "task.dependency_added",
  "task.dependency_removed",
  "task.run_enqueued",
  "task.run_claimed",
  "task.run_starting",
  "task.run_session_bound",
  "task.run_started",
  "task.run_completed",
  "task.run_failed",
  "task.run_canceled",
  "task.run_force_stopped",
  "task.run_recovered",
  "task.run_rejected",
  "task.run_lease_extended",
  "task.run_lease_expired",
  "task.run_released",
  "task.execution_profile_updated",
  "task.execution_profile_deleted",
  "task.run_review_requested",
  "task.run_review_bound",
  "task.run_review_recorded",
  "task.run_review_approved",
  "task.run_review_rejected",
  "task.run_review_blocked",
  "task.run_review_error",
  "task.run_review_timeout",
  "task.run_review_invalid_output",
  "task.run_review_retry_enqueued",
  "task.run_review_circuit_opened",
  "task.run_review_canceled",
  "task.notification_delivered",
] as const;

function defaultEventSourceFactory(url: string): TaskStreamEventSource {
  return new EventSource(url);
}

function resolveFilters(options: UseTaskStreamOptions): TaskStreamFilter {
  if (options.filters !== undefined) {
    return options.filters;
  }
  if (options.afterSequence !== undefined) {
    return { after_sequence: options.afterSequence };
  }
  return {};
}

type QueryClient = ReturnType<typeof useQueryClient>;

function invalidateTaskStreamQueries(queryClient: QueryClient, taskId: string) {
  void queryClient.invalidateQueries({ queryKey: tasksKeys.detail(taskId) });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.timelineRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.runsRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.runDetails() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.lists() });
  void queryClient.invalidateQueries({ queryKey: [...tasksKeys.all, "dashboard"] });
  void queryClient.invalidateQueries({ queryKey: [...tasksKeys.all, "inbox"] });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.contextBundle() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.agentContext() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.reviewsRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.bridgeNotificationsRoot() });
}

export function useTaskStream(taskId: string, options: UseTaskStreamOptions = {}) {
  const enabled = options.enabled ?? true;
  const eventSourceFactory = options.eventSourceFactory ?? defaultEventSourceFactory;
  const hasCustomFactory = Boolean(options.eventSourceFactory);
  const filters = resolveFilters(options);
  const afterSequence = filters.after_sequence;
  const onEvent = options.onEvent;
  const onError = options.onError;
  const queryClient = useQueryClient();
  const trimmedId = taskId.trim();

  useEffect(() => {
    if (
      !enabled ||
      trimmedId === "" ||
      typeof window === "undefined" ||
      (!hasCustomFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const url = buildTaskStreamUrl(trimmedId, { after_sequence: afterSequence });
    const source = eventSourceFactory(url);

    const handleMessage = (event: MessageEvent) => {
      if (typeof event.data !== "string") {
        return;
      }
      try {
        const payload = JSON.parse(event.data) as TaskStreamPayload;
        invalidateTaskStreamQueries(queryClient, trimmedId);
        if (onEvent) {
          onEvent(payload);
        }
      } catch (error) {
        if (onError) {
          onError(error);
        } else {
          console.error("Failed to parse task stream payload", error);
        }
      }
    };

    const handleError = (event: Event) => {
      if (onError) {
        onError(event);
      } else {
        console.error("Task stream failed", event);
      }
    };

    source.onmessage = handleMessage;
    source.onerror = handleError;

    const namedListener = handleMessage as EventListener;
    for (const type of TASK_STREAM_EVENT_TYPES) {
      source.addEventListener(type, namedListener);
    }

    return () => {
      if (source.removeEventListener) {
        for (const type of TASK_STREAM_EVENT_TYPES) {
          source.removeEventListener(type, namedListener);
        }
      }
      source.onmessage = null;
      source.onerror = null;
      source.close();
    };
  }, [
    enabled,
    trimmedId,
    afterSequence,
    eventSourceFactory,
    hasCustomFactory,
    onEvent,
    onError,
    queryClient,
  ]);
}

export type { TaskStreamEventSource, TaskStreamEventSourceFactory, UseTaskStreamOptions };
