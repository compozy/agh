import { Activity, AlertCircle } from "lucide-react";
import { useMemo, useState } from "react";

import {
  BlockLoading,
  Button,
  Empty,
  Eyebrow,
  Pill,
  PillGroup,
  Section,
  Time,
  Timeline,
  TimelineEvent,
} from "@agh/ui";
import type { PillGroupItem } from "@agh/ui";

import { cn } from "@/lib/utils";
import {
  describeEvent,
  isFailureEvent,
  isSuccessEvent,
  resolveEventTone,
  visualFor,
} from "../lib/timeline-visuals";
import { taskRunStatusLabel } from "../lib/task-formatters";
import type { TaskTimelineItem } from "../types";

export type TasksTimelineViewMode = "by_agent" | "interleaved" | "by_event_type";

export interface TasksTimelinePanelProps {
  items: TaskTimelineItem[];
  isLoading?: boolean;
  errorMessage?: string | null;
  isLive?: boolean;
  onLoadMore?: () => void;
  canLoadMore?: boolean;
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

function LiveIndicator() {
  return (
    <span
      className="inline-flex items-center gap-1.5 text-eyebrow"
      data-testid="tasks-timeline-live"
    >
      <Pill.Dot tone="info" pulse />
      <span className="eyebrow text-info">Live</span>
    </span>
  );
}

/**
 * Task events panel — interleaved by default, with a compact view-mode toggle
 * that switches to per-agent or per-event-type groupings of the same data.
 * Rows render through `<Timeline>` + `<TimelineEvent>` so the overview timeline
 * shares its rail / icon-well grammar with the run-detail panel. Live state
 * pulses through the `Live` chip in the section right slot — never paints
 * accent — so the page-level CTA keeps the single accent target.
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
      count={items.length}
      data-testid="tasks-timeline-panel"
      icon={Activity}
      label="Events"
      right={isLive ? <LiveIndicator /> : undefined}
    >
      <div className="flex flex-col gap-5">
        <PillGroup<TasksTimelineViewMode>
          aria-label="Timeline view mode"
          data-testid="tasks-timeline-view-mode"
          items={VIEW_MODE_ITEMS}
          onChange={setViewMode}
          size="sm"
          value={viewMode}
        />

        {viewMode === "interleaved" ? (
          <InterleavedTimeline isLive={isLive} items={items} />
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
                  <span className="font-mono text-mono-id tabular-nums text-faint">
                    {group.items.length}
                  </span>
                </header>
                <InterleavedTimeline isLive={isLive} items={group.items} />
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

interface InterleavedTimelineProps {
  items: TaskTimelineItem[];
  isLive: boolean;
}

function InterleavedTimeline({ items, isLive }: InterleavedTimelineProps) {
  return (
    <Timeline data-testid="tasks-timeline-interleaved">
      {items.map(item => {
        const visual = visualFor(item.event_type);
        const isFailure = isFailureEvent(item.event_type);
        const isSuccess = isSuccessEvent(item.event_type);
        const tone = resolveEventTone(item.event_type, isLive);
        const titleClass = cn(
          "font-mono tabular-nums",
          isFailure ? "text-danger" : isSuccess ? "text-success" : "text-fg-strong"
        );

        return (
          <TimelineEvent
            data-testid={`tasks-timeline-item-${item.event_id}`}
            description={
              <span data-testid={`tasks-timeline-message-${item.event_id}`}>
                {describeEvent(item)}
              </span>
            }
            icon={visual.icon}
            key={item.event_id}
            meta={
              <>
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
                    <span>{taskRunStatusLabel(item.run.status)}</span>
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
              </>
            }
            time={
              item.timestamp ? (
                <Time
                  data-testid={`tasks-timeline-timestamp-${item.event_id}`}
                  iso={item.timestamp}
                  mode="relative"
                />
              ) : undefined
            }
            title={
              <span
                className={titleClass}
                data-testid={`tasks-timeline-event-type-${item.event_id}`}
              >
                {item.event_type}
              </span>
            }
            tone={tone}
          />
        );
      })}
    </Timeline>
  );
}
