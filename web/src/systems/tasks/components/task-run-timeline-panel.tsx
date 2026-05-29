"use client";

import { useMemo } from "react";
import { Activity } from "lucide-react";

import {
  BlockLoading,
  Empty,
  Pill,
  RunCard,
  type RunCardStatus,
  Time,
  Timeline,
  TimelineEvent,
} from "@agh/ui";

import { cn } from "@/lib/utils";
import { taskRunStatusLabel } from "../lib/task-formatters";
import {
  describeEvent,
  isFailureEvent,
  isSuccessEvent,
  resolveEventTone,
  visualFor,
} from "../lib/timeline-visuals";
import type { TaskRunDetailView, TaskTimelineItem } from "../types";

export interface TaskRunTimelinePanelProps {
  run: TaskRunDetailView;
  items: TaskTimelineItem[];
  isLoading?: boolean;
  isLive?: boolean;
}

const RUN_STATUS_MAP: Record<string, RunCardStatus> = {
  queued: "pending",
  claimed: "in_progress",
  starting: "in_progress",
  running: "in_progress",
  needs_attention: "needs_attention",
  completed: "completed",
  failed: "failed",
  canceled: "canceled",
};

function computeElapsed(startedAt?: string | null, endedAt?: string | null): string | undefined {
  if (!startedAt) {
    return undefined;
  }
  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) {
    return undefined;
  }
  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) {
    return undefined;
  }
  const delta = Math.max(0, end - start);
  const totalSeconds = Math.floor(delta / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
}

function runCardWarning(record: TaskRunDetailView["run"]) {
  if (!record.error) {
    return undefined;
  }
  if (record.status === "needs_attention") {
    return {
      tone: "warning" as const,
      message: record.error,
    };
  }
  if (record.status === "failed" || record.status === "canceled") {
    return {
      tone: "danger" as const,
      message: record.error,
    };
  }
  return undefined;
}

/**
 * Run detail timeline panel — single `<RunCard>` header + `<TimelineEvent>`
 * rows scoped to the run. Live state pulses through the `<RunCard>` status
 * pill; the timeline itself stays free of accent paint so the run-header CTA
 * remains the single accent target per viewport.
 */
export function TaskRunTimelinePanel({
  run,
  items,
  isLoading = false,
  isLive = false,
}: TaskRunTimelinePanelProps) {
  const record = run.run;
  const session = run.session;
  const status = RUN_STATUS_MAP[record.status] ?? "pending";
  const channel = record.origin?.kind?.toUpperCase() ?? undefined;
  const elapsed = computeElapsed(record.started_at, record.ended_at);
  const sessionInfo =
    session?.agent_name && session?.session_id
      ? `session ${session.session_id} · agent ${session.agent_name}`
      : record.session_id
        ? `session ${record.session_id}`
        : undefined;

  const runEvents = useMemo(
    () => items.filter(item => item.run?.id === record.id),
    [items, record.id]
  );

  const warning = runCardWarning(record);

  return (
    <section
      aria-label="Run timeline"
      className="flex flex-col gap-5"
      data-testid="tasks-run-detail-timeline"
    >
      <RunCard
        attempt={record.attempt}
        channel={channel}
        data-testid="tasks-run-detail-card"
        elapsed={elapsed}
        queuedAt={record.queued_at ?? undefined}
        runId={record.id}
        sessionInfo={sessionInfo}
        startedAt={record.started_at ?? undefined}
        status={status}
        warning={warning}
      />

      {isLoading && runEvents.length === 0 ? (
        <BlockLoading
          data-testid="tasks-run-detail-timeline-loading"
          label="Loading run events"
          size="md"
          surface="bare"
        />
      ) : runEvents.length === 0 ? (
        <Empty
          data-testid="tasks-run-detail-timeline-empty"
          description="Events for this run will appear here as it executes."
          icon={Activity}
          title="No events yet"
        />
      ) : (
        <TimelineWithMarkers events={runEvents} isLive={isLive} />
      )}
    </section>
  );
}

interface TimelineWithMarkersProps {
  events: TaskTimelineItem[];
  isLive: boolean;
}

function TimelineWithMarkers({ events, isLive }: TimelineWithMarkersProps) {
  return (
    <Timeline data-testid="tasks-run-detail-timeline-list">
      {events.map(item => {
        const visual = visualFor(item.event_type);
        const isFailure = isFailureEvent(item.event_type);
        const isSuccess = isSuccessEvent(item.event_type);
        const tone = resolveEventTone(item.event_type, isLive);
        const titleClass = cn(
          isFailure ? "text-danger" : isSuccess ? "text-success" : "text-fg-strong"
        );
        return (
          <TimelineEvent
            data-testid={`tasks-run-detail-timeline-item-${item.event_id}`}
            description={describeEvent(item)}
            icon={visual.icon}
            key={item.event_id}
            meta={
              item.run ? (
                <>
                  <Pill mono size="xs">
                    seq {item.sequence}
                  </Pill>
                  <span aria-hidden className="text-faint">
                    ·
                  </span>
                  <span className="tabular-nums">attempt {item.run.attempt}</span>
                  <span aria-hidden className="text-faint">
                    ·
                  </span>
                  <span>{taskRunStatusLabel(item.run.status)}</span>
                </>
              ) : (
                <Pill mono size="xs">
                  seq {item.sequence}
                </Pill>
              )
            }
            time={item.timestamp ? <Time iso={item.timestamp} mode="relative" /> : undefined}
            title={<span className={titleClass}>{item.event_type}</span>}
            tone={tone}
          />
        );
      })}
    </Timeline>
  );
}
