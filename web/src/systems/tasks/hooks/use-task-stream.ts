import { useEffect, useEffectEvent, useRef } from "react";
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
// every task event type persisted by the task manager, transactional task hooks,
// and the review / notification surfaces in internal/api.
const TASK_STREAM_EVENT_TYPES = [
  "task.created",
  "task.updated",
  "task.published",
  "task.approved",
  "task.rejected",
  "task.canceled",
  // Transactional hook events are appended to the durable task stream under
  // their dot names inside the same transaction as the state change.
  "task.blocked",
  "task.unblocked",
  "task.needs_attention",
  "task.recovered",
  "task.status_changed",
  "task.run.completed",
  "task.run.failed",
  "task.child_created",
  "task.dependency_added",
  "task.dependency_removed",
  "task.paused",
  "task.resumed",
  "task.block.created",
  "task.block.cleared",
  "task.block.expired",
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
  "task.run_operator_forced_fail",
  "task.run_operator_retry",
  "task.run_recovered_from_attention",
  "task.run_starved",
  "task.run_needs_attention",
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
  "task.auto_enqueue.triggered",
  "task.completion.hallucination_blocked",
  "task.completion.hallucination_suspected",
  "task.wake.delivered",
  "task.wake.suppressed",
  "task.notification_delivered",
] as const;

function defaultEventSourceFactory(url: string): TaskStreamEventSource {
  return new EventSource(url);
}

function attachTaskStreamSource(
  source: TaskStreamEventSource,
  handleMessage: (event: MessageEvent) => void,
  handleError: (event: Event) => void
): () => void {
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
}

type QueryClient = ReturnType<typeof useQueryClient>;

function invalidateLiveTaskStreamQueries(queryClient: QueryClient, taskId: string) {
  void queryClient.invalidateQueries({ queryKey: tasksKeys.detail(taskId) });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.timelineRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.runsRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.runDetails() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.contextBundle() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.agentContext() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.reviewsRoot() });
  void queryClient.invalidateQueries({ queryKey: tasksKeys.bridgeNotificationsRoot() });
}

function markTaskCatalogsStale(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: tasksKeys.lists(), refetchType: "none" });
  void queryClient.invalidateQueries({
    queryKey: tasksKeys.dashboardRoot(),
    refetchType: "none",
  });
  void queryClient.invalidateQueries({
    queryKey: tasksKeys.inboxRoot(),
    refetchType: "none",
  });
}

function reconcileActiveTaskCatalogs(queryClient: QueryClient) {
  void queryClient.refetchQueries({ queryKey: tasksKeys.lists(), type: "active" });
  void queryClient.refetchQueries({
    queryKey: tasksKeys.dashboardRoot(),
    type: "active",
  });
  void queryClient.refetchQueries({ queryKey: tasksKeys.inboxRoot(), type: "active" });
}

export function useTaskStream(
  taskId: string,
  {
    enabled = true,
    afterSequence: fallbackAfterSequence,
    filters: { after_sequence: filteredAfterSequence = fallbackAfterSequence } = {},
    eventSourceFactory: customEventSourceFactory,
    onEvent,
    onError,
  }: UseTaskStreamOptions = {}
) {
  const queryClient = useQueryClient();
  const trimmedId = taskId.trim();
  const catalogsDirty = useRef(false);
  const notifyEvent = useEffectEvent((payload: TaskStreamPayload) => {
    onEvent?.(payload);
  });
  const notifyError = useEffectEvent((error: unknown, fallback: string) => {
    if (onError) {
      onError(error);
      return;
    }
    console.error(fallback, error);
  });

  useEffect(() => {
    if (!enabled || trimmedId === "") {
      return undefined;
    }
    return () => {
      if (!catalogsDirty.current) {
        return;
      }
      catalogsDirty.current = false;
      reconcileActiveTaskCatalogs(queryClient);
    };
  }, [enabled, queryClient, trimmedId]);

  useEffect(() => {
    if (
      !enabled ||
      trimmedId === "" ||
      typeof window === "undefined" ||
      (!customEventSourceFactory && typeof EventSource === "undefined")
    ) {
      return undefined;
    }

    const url = buildTaskStreamUrl(trimmedId, { after_sequence: filteredAfterSequence });
    const source = (customEventSourceFactory ?? defaultEventSourceFactory)(url);

    const handleMessage = (event: MessageEvent) => {
      if (typeof event.data !== "string") {
        return;
      }
      try {
        const payload = JSON.parse(event.data) as TaskStreamPayload;
        invalidateLiveTaskStreamQueries(queryClient, trimmedId);
        if (!catalogsDirty.current) {
          catalogsDirty.current = true;
          markTaskCatalogsStale(queryClient);
        }
        notifyEvent(payload);
      } catch (error) {
        notifyError(error, "Failed to parse task stream payload");
      }
    };

    const handleError = (event: Event) => {
      notifyError(event, "Task stream failed");
    };

    return attachTaskStreamSource(source, handleMessage, handleError);
  }, [customEventSourceFactory, enabled, filteredAfterSequence, queryClient, trimmedId]);
}

export type { TaskStreamEventSource, TaskStreamEventSourceFactory, UseTaskStreamOptions };
