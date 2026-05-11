import { Activity, AlertCircle } from "lucide-react";
import { useMemo, useState } from "react";

import { BlockLoading, Button, Empty, Pill, PillGroup, Section } from "@agh/ui";
import type { PillGroupItem } from "@agh/ui";

import { taskStatusSignal } from "../lib/task-formatters";
import type { TaskRunStatus, TaskTimelineItem } from "../types";

export type TasksTimelineViewMode = "by_agent" | "interleaved" | "by_event_type";

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
  if (!value) return "";
  const ts = Date.parse(value);
  if (Number.isNaN(ts)) return "";
  const date = new Date(ts);
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}

function describeEvent(item: TaskTimelineItem): string {
  const payload = item.payload as Record<string, unknown> | undefined;
  const message = payload && typeof payload === "object" ? (payload.message as string) : undefined;
  if (typeof message === "string" && message.trim().length > 0) return message;

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

const VIEW_MODE_ITEMS: ReadonlyArray<PillGroupItem<TasksTimelineViewMode>> = [
  { value: "interleaved", label: "Interleaved", testId: "tasks-timeline-view-interleaved" },
  { value: "by_agent", label: "By agent", testId: "tasks-timeline-view-by-agent" },
  { value: "by_event_type", label: "By event type", testId: "tasks-timeline-view-by-event-type" },
];

interface TimelineGroup {
  key: string;
  label: string;
  items: TaskTimelineItem[];
}

function groupByAgent(items: TaskTimelineItem[]): TimelineGroup[] {
  const buckets = new Map<string, { label: string; items: TaskTimelineItem[] }>();
  for (const item of items) {
    const id = item.task?.id ?? "--";
    const label = item.task?.identifier ?? item.task?.title ?? id;
    const bucket = buckets.get(id);
    if (bucket) {
      bucket.items.push(item);
    } else {
      buckets.set(id, { label, items: [item] });
    }
  }
  return Array.from(buckets.entries()).map(([key, value]) => ({
    key,
    label: value.label,
    items: value.items,
  }));
}

function groupByEventType(items: TaskTimelineItem[]): TimelineGroup[] {
  const buckets = new Map<string, TaskTimelineItem[]>();
  for (const item of items) {
    const key = item.event_type;
    const bucket = buckets.get(key);
    if (bucket) {
      bucket.push(item);
    } else {
      buckets.set(key, [item]);
    }
  }
  return Array.from(buckets.entries()).map(([key, groupItems]) => ({
    key,
    label: key,
    items: groupItems,
  }));
}

/**
 * Events panel -- interleaved by default, with a compact view-mode toggle that
 * switches to per-agent or per-event-type groupings of the same data. Each row
 * carries a `StatusDot` whose tone follows the event kind (failure → danger,
 * live → accent, otherwise neutral) and the load-more button paginates the
 * underlying cursor.
 */
export function TasksTimelinePanel({
  items,
  isLoading = false,
  errorMessage = null,
  isLive = false,
  onLoadMore,
  canLoadMore = false,
}: TasksTimelinePanelProps) {
  const [viewMode, setViewMode] = useState<TasksTimelineViewMode>("interleaved");

  const groups = useMemo<TimelineGroup[]>(() => {
    if (viewMode === "by_agent") return groupByAgent(items);
    if (viewMode === "by_event_type") return groupByEventType(items);
    return [];
  }, [items, viewMode]);

  if (isLoading && items.length === 0) {
    return (
      <BlockLoading
        label="Loading task events"
        size="md"
        surface="bare"
        data-testid="tasks-timeline-loading"
      />
    );
  }

  if (errorMessage && items.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Unable to load events"
        description={errorMessage}
        data-testid="tasks-timeline-error"
      />
    );
  }

  if (items.length === 0) {
    return (
      <Empty
        icon={Activity}
        title="No events recorded yet"
        description="Events will appear here as this task is executed."
        data-testid="tasks-timeline-empty"
      />
    );
  }

  return (
    <Section
      aria-label="Task events"
      className="w-full gap-6 px-6 py-5"
      data-testid="tasks-timeline-panel"
    >
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between gap-3">
          <PillGroup<TasksTimelineViewMode>
            aria-label="Timeline view mode"
            data-testid="tasks-timeline-view-mode"
            items={VIEW_MODE_ITEMS}
            onChange={setViewMode}
            size="sm"
            value={viewMode}
          />
          {isLive ? (
            <span
              className="inline-flex items-center gap-1 text-badge text-accent"
              data-testid="tasks-timeline-live"
            >
              <Pill.Dot tone="accent" pulse />
              Live
            </span>
          ) : null}
        </div>

        {viewMode === "interleaved" ? (
          <InterleavedEventList isLive={isLive} items={items} />
        ) : (
          <div className="flex flex-col gap-5" data-testid="tasks-timeline-groups">
            {groups.map(group => (
              <section
                className="flex flex-col gap-3"
                data-testid={`tasks-timeline-group-${group.key}`}
                key={group.key}
              >
                <header className="flex items-baseline justify-between gap-2">
                  <h3 className="text-eyebrow font-medium text-muted">{group.label}</h3>
                  <span className="font-mono text-eyebrow text-subtle">({group.items.length})</span>
                </header>
                <InterleavedEventList isLive={isLive} items={group.items} />
              </section>
            ))}
          </div>
        )}

        {canLoadMore && onLoadMore ? (
          <div className="flex items-center justify-center border-t border-line pt-4">
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
      </div>
    </Section>
  );
}

interface InterleavedEventListProps {
  items: TaskTimelineItem[];
  isLive: boolean;
}

function InterleavedEventList({ items, isLive }: InterleavedEventListProps) {
  return (
    <ol className="flex flex-col gap-4">
      {items.map(item => {
        const isFailure = isFailureEvent(item.event_type);
        const isRunning = isLive && isLiveEvent(item.event_type);
        const tone = isFailure ? "danger" : isRunning ? "accent" : "neutral";
        const signalTone = isFailure
          ? "danger"
          : item.run
            ? taskStatusSignal(item.run.status).tone
            : tone;
        const pulse = isRunning;
        const timestamp = formatTime(item.timestamp);

        return (
          <div
            className="relative flex gap-3"
            data-testid={`tasks-timeline-item-${item.event_id}`}
            key={item.event_id}
          >
            <div className="mt-1">
              <Pill.Dot pulse={pulse} tone={signalTone} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2 text-badge text-muted">
                <span
                  className={isFailure ? "text-danger" : "text-fg"}
                  data-testid={`tasks-timeline-event-type-${item.event_id}`}
                >
                  {item.event_type}
                </span>
                <Pill mono>seq {item.sequence}</Pill>
                {item.run ? (
                  <>
                    <span>· attempt {item.run.attempt}</span>
                    <span>· {runStatusPill(item.run.status)}</span>
                  </>
                ) : null}
                {item.origin?.ref ? <span>· {item.origin.ref}</span> : null}
              </div>
              <p
                className={`mt-1 text-small-body ${isFailure ? "text-danger" : "text-fg"}`}
                data-testid={`tasks-timeline-message-${item.event_id}`}
              >
                {describeEvent(item)}
              </p>
            </div>
            {timestamp ? (
              <span
                className="mt-1 shrink-0 font-mono text-eyebrow text-subtle"
                data-testid={`tasks-timeline-timestamp-${item.event_id}`}
              >
                {timestamp}
              </span>
            ) : null}
          </div>
        );
      })}
    </ol>
  );
}
