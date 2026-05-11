import { Activity, AlertCircle } from "lucide-react";
import { useMemo, useState } from "react";

import { BlockLoading, Button, Empty, Eyebrow, Pill, PillGroup, Section, Time } from "@agh/ui";
import type { PillGroupItem } from "@agh/ui";

import { cn } from "@/lib/utils";
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
 * Task events panel — interleaved by default, with a compact view-mode toggle
 * that switches to per-agent or per-event-type groupings of the same data.
 * Each row carries a `Pill.Dot` tone that follows the event kind (failure →
 * danger, live → info, otherwise neutral). Live state pulses the dot — never
 * paints accent — so the page-level CTA keeps the single accent target.
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
      <div className="flex flex-col gap-5">
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
              className="inline-flex items-center gap-1.5 text-[11px] text-info"
              data-testid="tasks-timeline-live"
            >
              <Pill.Dot tone="info" pulse />
              <span className="eyebrow text-info">Live</span>
            </span>
          ) : null}
        </div>

        {viewMode === "interleaved" ? (
          <InterleavedEventList isLive={isLive} items={items} />
        ) : (
          <div className="flex flex-col gap-6" data-testid="tasks-timeline-groups">
            {groups.map(group => (
              <section
                className="flex flex-col gap-3"
                data-testid={`tasks-timeline-group-${group.key}`}
                key={group.key}
              >
                <header className="flex items-baseline justify-between gap-2">
                  <Eyebrow className="truncate text-muted">{group.label}</Eyebrow>
                  <span className="font-mono text-[10.5px] tabular-nums text-faint">
                    {group.items.length}
                  </span>
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
              variant="neutral"
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
    <ol className="flex flex-col gap-3" data-testid="tasks-timeline-interleaved">
      {items.map(item => {
        const isFailure = isFailureEvent(item.event_type);
        const isRunning = isLive && isLiveEvent(item.event_type);
        const signalTone = isFailure
          ? "danger"
          : item.run
            ? taskStatusSignal(item.run.status).tone
            : "neutral";

        return (
          <li
            className="grid grid-cols-[auto_1fr_auto] items-start gap-3"
            data-testid={`tasks-timeline-item-${item.event_id}`}
            key={item.event_id}
          >
            <span className="mt-1.5 inline-flex shrink-0">
              <Pill.Dot pulse={isRunning} tone={signalTone} />
            </span>
            <div className="min-w-0 flex-1 flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2 text-[11.5px] text-muted">
                <span
                  className={cn(
                    "font-mono tabular-nums",
                    isFailure ? "text-danger" : "text-fg-strong"
                  )}
                  data-testid={`tasks-timeline-event-type-${item.event_id}`}
                >
                  {item.event_type}
                </span>
                <Pill mono size="xs">
                  seq {item.sequence}
                </Pill>
                {item.run ? (
                  <>
                    <span aria-hidden className="text-faint">
                      ·
                    </span>
                    <span className="tabular-nums">attempt {item.run.attempt}</span>
                    <span aria-hidden className="text-faint">
                      ·
                    </span>
                    <span>{runStatusPill(item.run.status)}</span>
                  </>
                ) : null}
                {item.origin?.ref ? (
                  <>
                    <span aria-hidden className="text-faint">
                      ·
                    </span>
                    <span>{item.origin.ref}</span>
                  </>
                ) : null}
              </div>
              <p
                className={cn("mt-1 text-[12.5px]", isFailure ? "text-danger" : "text-fg")}
                data-testid={`tasks-timeline-message-${item.event_id}`}
              >
                {describeEvent(item)}
              </p>
            </div>
            {item.timestamp ? (
              <Time
                className="mt-1 shrink-0 font-mono text-[10.5px] tabular-nums text-subtle"
                data-testid={`tasks-timeline-timestamp-${item.event_id}`}
                iso={item.timestamp}
                mode="relative"
              />
            ) : (
              <span aria-hidden />
            )}
          </li>
        );
      })}
    </ol>
  );
}
