"use client";

import { useMemo } from "react";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  CircleDot,
  GitBranch,
  Hourglass,
  Inbox as InboxIcon,
  Pause,
  PlayCircle,
  Plus,
  Sparkles,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import {
  BlockLoading,
  Empty,
  Pill,
  RunCard,
  type PillTone,
  type RunCardStatus,
  Time,
  Timeline,
  TimelineEvent,
} from "@agh/ui";

import { cn } from "@/lib/utils";
import { taskRunStatusLabel } from "../lib/task-formatters";
import type { TaskRunDetailView, TaskTimelineItem } from "../types";

export interface TaskRunTimelinePanelProps {
  run: TaskRunDetailView;
  items: TaskTimelineItem[];
  isLoading?: boolean;
  isLive?: boolean;
}

interface EventVisualMeta {
  tone: PillTone;
  icon: LucideIcon;
}

const FAILURE_EVENT_TYPES = new Set([
  "task.run_failed",
  "task.failed",
  "task.run_canceled",
  "task.canceled",
]);

const LIVE_EVENT_TYPES = new Set(["task.run_progress", "task.run_started", "task.run_claimed"]);

const SUCCESS_EVENT_TYPES = new Set(["task.run_completed", "task.completed"]);

const EVENT_VISUALS: Record<string, EventVisualMeta> = {
  "task.created": { tone: "neutral", icon: Plus },
  "task.run_enqueued": { tone: "neutral", icon: InboxIcon },
  "task.run_claimed": { tone: "info", icon: CircleDot },
  "task.run_started": { tone: "info", icon: PlayCircle },
  "task.run_progress": { tone: "info", icon: Hourglass },
  "task.run_completed": { tone: "success", icon: CheckCircle2 },
  "task.completed": { tone: "success", icon: CheckCircle2 },
  "task.run_failed": { tone: "danger", icon: XCircle },
  "task.failed": { tone: "danger", icon: XCircle },
  "task.run_canceled": { tone: "warning", icon: Pause },
  "task.canceled": { tone: "warning", icon: Pause },
  "task.run_blocked": { tone: "warning", icon: AlertTriangle },
  "task.dependency_added": { tone: "neutral", icon: GitBranch },
  "task.dependency_resolved": { tone: "success", icon: GitBranch },
};

function visualFor(eventType: string): EventVisualMeta {
  return EVENT_VISUALS[eventType] ?? { tone: "neutral", icon: Sparkles };
}

function describeEvent(item: TaskTimelineItem): string {
  const payload = item.payload as Record<string, unknown> | undefined;
  const message = payload && typeof payload === "object" ? (payload.message as string) : undefined;
  if (typeof message === "string" && message.trim().length > 0) return message;
  switch (item.event_type) {
    case "task.run_enqueued":
      return "Run queued";
    case "task.run_claimed":
      return "Run claimed";
    case "task.run_started":
      return "Run started";
    case "task.run_progress":
      return "Run in progress";
    case "task.run_completed":
      return "Run completed";
    case "task.run_failed":
      return item.run?.error ?? "Run failed";
    case "task.run_canceled":
      return "Run canceled";
    default:
      return item.event_type;
  }
}

const RUN_STATUS_MAP: Record<string, RunCardStatus> = {
  queued: "pending",
  claimed: "in_progress",
  starting: "in_progress",
  running: "in_progress",
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

/**
 * Run detail timeline panel — single `<RunCard>` header + `<TimelineEvent>`
 * rows scoped to the run (per ADR-007 §9 + ADR-008 §3). Replaces the old
 * `MetadataList` Identity panel which duplicated `<RunCard>` data.
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

  const warning =
    record.error && (record.status === "failed" || record.status === "canceled")
      ? {
          tone: "danger" as const,
          message: record.error,
        }
      : undefined;

  return (
    <section
      aria-label="Run timeline"
      className="flex flex-col gap-4"
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
        const isLiveEvent = isLive && LIVE_EVENT_TYPES.has(item.event_type);
        const tone: PillTone = isLiveEvent ? "accent" : visual.tone;
        const isFailure = FAILURE_EVENT_TYPES.has(item.event_type);
        const isSuccess = SUCCESS_EVENT_TYPES.has(item.event_type);
        const titleClass = cn(
          "font-mono tracking-[-0.005em]",
          isFailure ? "text-(--danger)" : isSuccess ? "text-(--success)" : "text-(--fg-strong)"
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
                  <Pill mono>seq {item.sequence}</Pill>
                  <span aria-hidden>·</span>
                  <span className="tabular-nums">attempt {item.run.attempt}</span>
                  <span aria-hidden>·</span>
                  <span>{taskRunStatusLabel(item.run.status)}</span>
                </>
              ) : (
                <Pill mono>seq {item.sequence}</Pill>
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
