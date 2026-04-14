import type {
  AutomationKind,
  AutomationFireLimit,
  AutomationJob,
  AutomationRetry,
  AutomationRun,
  AutomationRunStatus,
  AutomationSchedule,
  AutomationScope,
  AutomationScopeFilter,
  AutomationTrigger,
} from "../types";

export type AutomationSemanticTone = "amber" | "danger" | "green" | "neutral" | "violet";

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

export function formatDate(dateStr?: string | null): string {
  if (!dateStr) {
    return "Unavailable";
  }

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }

  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
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

export function formatRunDuration(run: AutomationRun): string {
  const startedAt = run.started_at ? new Date(run.started_at) : null;
  if (!startedAt || Number.isNaN(startedAt.getTime())) {
    return "Queued";
  }

  if (!run.ended_at) {
    return "Running";
  }

  const endedAt = new Date(run.ended_at);
  if (Number.isNaN(endedAt.getTime())) {
    return "Unavailable";
  }

  const diffMs = Math.max(0, endedAt.getTime() - startedAt.getTime());
  const totalSeconds = Math.round(diffMs / 1000);

  if (totalSeconds < 60) {
    return `${totalSeconds}s`;
  }

  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }

  return `${minutes}m ${seconds}s`;
}

export function formatPromptPreview(prompt: string, maxLength = 72): string {
  const normalized = prompt.replaceAll(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) {
    return normalized;
  }

  return `${normalized.slice(0, maxLength - 1).trimEnd()}…`;
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

export function automationSemanticTone(
  status: AutomationRunStatus | "enabled" | "disabled"
): AutomationSemanticTone {
  switch (automationStatusTone(status)) {
    case "accent":
    case "warning":
      return "amber";
    case "success":
      return "green";
    case "danger":
      return "danger";
    case "neutral":
    default:
      return "neutral";
  }
}

export function automationSourceLabel(source: AutomationJob["source"]): string {
  return source === "config" ? "CONFIG" : "DYNAMIC";
}

export function automationScopeLabel(scope: AutomationScope): string {
  return scope === "workspace" ? "WORKSPACE" : "GLOBAL";
}

export function automationSourceTone(source: AutomationJob["source"]): AutomationSemanticTone {
  return source === "dynamic" ? "amber" : "neutral";
}

export function automationScopeTone(scope: AutomationScope): AutomationSemanticTone {
  return scope === "workspace" ? "violet" : "neutral";
}

function searchableParts(parts: Array<string | null | undefined>): string {
  return parts
    .filter((value): value is string => typeof value === "string" && value.trim() !== "")
    .join(" ")
    .toLowerCase();
}

export function filterAutomationJobs(jobs: AutomationJob[], query: string): AutomationJob[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery === "") {
    return jobs;
  }

  return jobs.filter(job =>
    searchableParts([
      job.name,
      job.agent_name,
      job.prompt,
      job.scope,
      job.source,
      job.schedule?.mode,
      job.schedule?.expr,
      job.schedule?.interval,
      job.schedule?.time,
    ]).includes(normalizedQuery)
  );
}

export function filterAutomationTriggers(
  triggers: AutomationTrigger[],
  query: string
): AutomationTrigger[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery === "") {
    return triggers;
  }

  return triggers.filter(trigger =>
    searchableParts([
      trigger.name,
      trigger.agent_name,
      trigger.prompt,
      trigger.scope,
      trigger.source,
      trigger.event,
      trigger.endpoint_slug,
      trigger.webhook_id,
      ...Object.entries(trigger.filter ?? {}).flat(),
    ]).includes(normalizedQuery)
  );
}

function sortBySourceAndName<T extends { name: string; source: AutomationJob["source"] }>(
  items: T[]
): T[] {
  const sourceOrder = {
    config: 0,
    dynamic: 1,
  } as const;

  return [...items].sort((left, right) => {
    const sourceDelta = sourceOrder[left.source] - sourceOrder[right.source];
    if (sourceDelta !== 0) {
      return sourceDelta;
    }

    return left.name.localeCompare(right.name);
  });
}

export function sortAutomationJobs(jobs: AutomationJob[]): AutomationJob[] {
  return sortBySourceAndName(jobs);
}

export function sortAutomationTriggers(triggers: AutomationTrigger[]): AutomationTrigger[] {
  return sortBySourceAndName(triggers);
}

export function formatAutomationListSummary({
  activeWorkspaceName,
  kind,
  scopeFilter,
  searchQuery,
  totalCount,
  visibleCount,
}: {
  activeWorkspaceName?: string;
  kind: AutomationKind;
  scopeFilter: AutomationScopeFilter;
  searchQuery: string;
  totalCount: number;
  visibleCount: number;
}): string {
  const noun =
    kind === "jobs"
      ? visibleCount === 1
        ? "job"
        : "jobs"
      : visibleCount === 1
        ? "trigger"
        : "triggers";
  const totalNoun =
    kind === "jobs"
      ? totalCount === 1
        ? "job"
        : "jobs"
      : totalCount === 1
        ? "trigger"
        : "triggers";
  const trimmedQuery = searchQuery.trim();

  if (trimmedQuery !== "") {
    return `${visibleCount} ${noun} matching current search`;
  }

  if (totalCount === 0) {
    return `0 ${kind} found`;
  }

  if (scopeFilter === "all") {
    return `${totalCount} ${totalNoun} across all scopes`;
  }

  if (scopeFilter === "global") {
    return `${visibleCount} ${noun} in global scope`;
  }

  if (activeWorkspaceName) {
    return `${visibleCount} ${noun} in ${activeWorkspaceName}`;
  }

  return `${visibleCount} ${noun} in workspace scope`;
}
