import { AlertCircle, Loader2 } from "lucide-react";

import { Button } from "@agh/ui";

import type { TaskRunStatus, TaskTimelineItem } from "../types";

export interface TasksTimelinePanelProps {
  items: TaskTimelineItem[];
  isLoading?: boolean;
  errorMessage?: string | null;
  isLive?: boolean;
  onLoadMore?: () => void;
  canLoadMore?: boolean;
}

const FAILURE_EVENT_TYPES = new Set([
  "task.run_failed",
  "task.failed",
  "task.run_canceled",
  "task.canceled",
]);

const LIVE_EVENT_TYPES = new Set(["task.run_progress", "task.run_started", "task.run_claimed"]);

function isFailureEvent(eventType: string): boolean {
  return FAILURE_EVENT_TYPES.has(eventType);
}

function isLiveEvent(eventType: string): boolean {
  return LIVE_EVENT_TYPES.has(eventType);
}

function formatTime(value?: string | null): string {
  if (!value) {
    return "";
  }

  const ts = Date.parse(value);
  if (Number.isNaN(ts)) {
    return "";
  }

  const date = new Date(ts);
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}

function describeEvent(item: TaskTimelineItem): string {
  const payload = item.payload as Record<string, unknown> | undefined;
  const message = payload && typeof payload === "object" ? (payload.message as string) : undefined;
  if (typeof message === "string" && message.trim().length > 0) {
    return message;
  }

  switch (item.event_type) {
    case "task.created":
      return `Task ${item.task.identifier ?? item.task.id} created`;
    case "task.run_enqueued":
      return item.run ? `Run ${item.run.id} queued` : "Run queued";
    case "task.run_claimed":
      return item.run ? `Run ${item.run.id} claimed` : "Run claimed";
    case "task.run_started":
      return item.run ? `Run ${item.run.id} started` : "Run started";
    case "task.run_progress":
      return item.run ? `Run ${item.run.id} in progress` : "Run in progress";
    case "task.run_completed":
      return item.run ? `Run ${item.run.id} completed` : "Run completed";
    case "task.run_failed":
      return item.run?.error
        ? item.run.error
        : item.run
          ? `Run ${item.run.id} failed`
          : "Run failed";
    case "task.run_canceled":
      return item.run ? `Run ${item.run.id} canceled` : "Run canceled";
    case "task.dependency_added":
      return "Dependency added";
    case "task.dependency_resolved":
      return "Dependency resolved";
    default:
      return item.event_type;
  }
}

function runStatusPill(status: TaskRunStatus): string {
  switch (status) {
    case "running":
      return "Running";
    case "completed":
      return "Completed";
    case "failed":
      return "Failed";
    case "canceled":
      return "Canceled";
    case "queued":
      return "Queued";
    case "claimed":
      return "Claimed";
    case "starting":
      return "Starting";
    default:
      return status;
  }
}

export function TasksTimelinePanel({
  items,
  isLoading = false,
  errorMessage = null,
  isLive = false,
  onLoadMore,
  canLoadMore = false,
}: TasksTimelinePanelProps) {
  if (isLoading && items.length === 0) {
    return (
      <div
        className="flex min-h-[240px] items-center justify-center"
        data-testid="tasks-timeline-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage && items.length === 0) {
    return (
      <div
        className="flex min-h-[240px] flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-timeline-error"
      >
        <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
        <p className="text-sm text-[color:var(--color-text-secondary)]">{errorMessage}</p>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div
        className="flex min-h-[240px] flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-timeline-empty"
      >
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          No events recorded yet for this task.
        </p>
      </div>
    );
  }

  return (
    <section
      aria-label="Task timeline"
      className="flex min-h-0 flex-1 flex-col"
      data-testid="tasks-timeline-panel"
    >
      <ol className="flex flex-col gap-4 px-6 py-5">
        {items.map(item => {
          const isFailure = isFailureEvent(item.event_type);
          const isRunning = isLive && isLiveEvent(item.event_type);
          const timestamp = formatTime(item.timestamp);

          return (
            <li
              className="relative flex gap-3"
              data-testid={`tasks-timeline-item-${item.event_id}`}
              key={item.event_id}
            >
              <span
                aria-hidden="true"
                className={`mt-1 block size-2.5 shrink-0 rounded-full ${
                  isFailure
                    ? "bg-[color:var(--color-danger)]"
                    : isRunning
                      ? "bg-[color:var(--color-accent)]"
                      : "border border-[color:var(--color-divider)] bg-transparent"
                }`}
              />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                  <span
                    className={
                      isFailure
                        ? "text-[color:var(--color-danger)]"
                        : "text-[color:var(--color-text-primary)]"
                    }
                    data-testid={`tasks-timeline-event-type-${item.event_id}`}
                  >
                    {item.event_type}
                  </span>
                  <span>seq {item.sequence}</span>
                  {item.run ? (
                    <>
                      <span>· attempt {item.run.attempt}</span>
                      <span>· {runStatusPill(item.run.status)}</span>
                    </>
                  ) : null}
                  {item.origin?.ref ? <span>· {item.origin.ref}</span> : null}
                  {isRunning ? (
                    <span
                      className="inline-flex h-4 items-center rounded-sm bg-[color:var(--color-accent-tint)] px-1.5 text-[color:var(--color-accent)]"
                      data-testid={`tasks-timeline-live-${item.event_id}`}
                    >
                      Live
                    </span>
                  ) : null}
                </div>
                <p
                  className={`mt-1 text-sm ${
                    isFailure
                      ? "text-[color:var(--color-danger)]"
                      : "text-[color:var(--color-text-primary)]"
                  }`}
                  data-testid={`tasks-timeline-message-${item.event_id}`}
                >
                  {describeEvent(item)}
                </p>
              </div>
              {timestamp ? (
                <span
                  className="mt-1 shrink-0 font-mono text-[0.66rem] text-[color:var(--color-text-tertiary)]"
                  data-testid={`tasks-timeline-timestamp-${item.event_id}`}
                >
                  {timestamp}
                </span>
              ) : null}
            </li>
          );
        })}
      </ol>

      {canLoadMore && onLoadMore ? (
        <div className="flex items-center justify-center border-t border-[color:var(--color-divider)] px-6 py-4">
          <Button
            data-testid="tasks-timeline-load-more"
            onClick={onLoadMore}
            size="sm"
            type="button"
            variant="outline"
          >
            Load more
          </Button>
        </div>
      ) : null}
    </section>
  );
}
