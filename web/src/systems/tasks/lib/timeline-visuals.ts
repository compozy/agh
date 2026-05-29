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
  RotateCw,
  Sparkles,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import type { PillTone } from "@agh/ui";

import type { TaskTimelineItem } from "../types";

export interface EventVisualMeta {
  tone: PillTone;
  icon: LucideIcon;
}

export const FAILURE_EVENT_TYPES: ReadonlySet<string> = new Set([
  "task.run_failed",
  "task.failed",
  "task.run_canceled",
  "task.canceled",
]);

export const LIVE_EVENT_TYPES: ReadonlySet<string> = new Set([
  "task.run_progress",
  "task.run_started",
  "task.run_claimed",
]);

export const SUCCESS_EVENT_TYPES: ReadonlySet<string> = new Set([
  "task.run_completed",
  "task.completed",
]);

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
  "task.run_operator_forced_fail": { tone: "danger", icon: XCircle },
  "task.run_operator_retry": { tone: "info", icon: RotateCw },
  "task.run_recovered_from_attention": { tone: "info", icon: RotateCw },
  "task.run_starved": { tone: "warning", icon: Hourglass },
  "task.run_needs_attention": { tone: "warning", icon: AlertTriangle },
  "task.dependency_added": { tone: "neutral", icon: GitBranch },
  "task.dependency_resolved": { tone: "success", icon: GitBranch },
};

export function visualFor(eventType: string): EventVisualMeta {
  return EVENT_VISUALS[eventType] ?? { tone: "neutral", icon: Sparkles };
}

export function isFailureEvent(eventType: string): boolean {
  return FAILURE_EVENT_TYPES.has(eventType);
}

export function isLiveEvent(eventType: string): boolean {
  return LIVE_EVENT_TYPES.has(eventType);
}

export function isSuccessEvent(eventType: string): boolean {
  return SUCCESS_EVENT_TYPES.has(eventType);
}

export function describeEvent(item: TaskTimelineItem): string {
  const payload = item.payload as Record<string, unknown> | undefined;
  const message = payload && typeof payload === "object" ? (payload.message as string) : undefined;
  if (typeof message === "string" && message.trim().length > 0) return message;
  const diagnostic =
    payload && typeof payload === "object" ? (payload.diagnostic as string) : undefined;
  if (typeof diagnostic === "string" && diagnostic.trim().length > 0) return diagnostic;
  const reason = payload && typeof payload === "object" ? (payload.reason as string) : undefined;
  if (typeof reason === "string" && reason.trim().length > 0) return reason;

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
    case "task.run_operator_forced_fail":
      return item.run ? `Run ${item.run.id} force failed` : "Run force failed";
    case "task.run_operator_retry":
      return item.run ? `Run ${item.run.id} retry queued` : "Run retry queued";
    case "task.run_recovered_from_attention":
      return item.run ? `Run ${item.run.id} recovered` : "Run recovered";
    case "task.run_starved":
      return item.run ? `Run ${item.run.id} is waiting for a claim` : "Run waiting for claim";
    case "task.run_needs_attention":
      return item.run?.error
        ? item.run.error
        : item.run
          ? `Run ${item.run.id} needs attention`
          : "Run needs attention";
    case "task.dependency_added":
      return "Dependency added";
    case "task.dependency_resolved":
      return "Dependency resolved";
    default:
      return item.event_type;
  }
}

/**
 * Resolves the `<TimelineEvent>` tone for an event, factoring in the
 * top-level kind (failure / success) and the live state of the surface.
 * Both `tasks-timeline-panel` and `task-run-timeline-panel` consume this.
 */
export function resolveEventTone(eventType: string, isLive: boolean): PillTone {
  if (isFailureEvent(eventType)) return "danger";
  if (isSuccessEvent(eventType)) return "success";
  if (isLive && isLiveEvent(eventType)) return "info";
  return visualFor(eventType).tone;
}

// Re-export Activity icon for callers that need a generic event surface icon
// (overview panel "Events" section header).
export { Activity };
