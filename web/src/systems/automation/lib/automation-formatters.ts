import type {
  AutomationFireLimit,
  AutomationJob,
  AutomationRetry,
  AutomationRun,
  AutomationRunStatus,
  AutomationSchedule,
  AutomationTrigger,
} from "../types";

export function formatRelativeTime(dateStr?: string | null): string {
  if (!dateStr) {
    return "Not scheduled";
  }

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }

  const diffMs = date.getTime() - Date.now();
  const diffMinutes = Math.round(diffMs / (1000 * 60));
  const absMinutes = Math.abs(diffMinutes);

  if (absMinutes < 1) {
    return diffMinutes >= 0 ? "Now" : "Just now";
  }

  if (absMinutes < 60) {
    return diffMinutes >= 0 ? `In ${absMinutes}m` : `${absMinutes}m ago`;
  }

  const absHours = Math.round(absMinutes / 60);
  if (absHours < 24) {
    return diffMinutes >= 0 ? `In ${absHours}h` : `${absHours}h ago`;
  }

  const absDays = Math.round(absHours / 24);
  return diffMinutes >= 0 ? `In ${absDays}d` : `${absDays}d ago`;
}

export function formatDateTime(dateStr?: string | null): string {
  if (!dateStr) {
    return "Unavailable";
  }

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }

  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function describeSchedule(schedule?: AutomationSchedule | null): string {
  if (!schedule) {
    return "Manual";
  }

  switch (schedule.mode) {
    case "cron":
      return schedule.expr ? `Cron ${schedule.expr}` : "Cron";
    case "every":
      return schedule.interval ? `Every ${schedule.interval}` : "Every interval";
    case "at":
      return schedule.time ? `At ${formatDateTime(schedule.time)}` : "One-shot";
    default:
      return "Manual";
  }
}

export function describeTrigger(trigger: AutomationTrigger): string {
  if (trigger.event !== "webhook") {
    return trigger.event;
  }

  if (trigger.endpoint_slug) {
    return `webhook:${trigger.endpoint_slug}`;
  }

  if (trigger.webhook_id) {
    return `webhook:${trigger.webhook_id}`;
  }

  return "webhook";
}

export function describeRetry(retry: AutomationRetry): string {
  if (retry.strategy === "none") {
    return "No retries";
  }

  return `${retry.max_retries} retries from ${retry.base_delay}`;
}

export function describeFireLimit(limit: AutomationFireLimit): string {
  return `${limit.max} fires / ${limit.window}`;
}

export function formatRunTitle(run: AutomationRun): string {
  return `${run.status.toUpperCase()} · attempt ${run.attempt}`;
}

export function automationStatusTone(
  status: AutomationRunStatus | "enabled" | "disabled"
): "accent" | "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "running":
      return "accent";
    case "completed":
    case "enabled":
      return "success";
    case "scheduled":
      return "warning";
    case "failed":
      return "danger";
    case "cancelled":
    case "disabled":
    default:
      return "neutral";
  }
}

export function automationSourceLabel(source: AutomationJob["source"]): string {
  return source === "config" ? "CONFIG" : "DYNAMIC";
}
