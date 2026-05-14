import { Activity, AlertCircle, AlertTriangle, Clock, Wrench } from "lucide-react";

import { Alert, AlertDescription, AlertMeta, AlertTitle, Pill, cn } from "@agh/ui";

import type { AgentEventPayload, RuntimeActivityPayload } from "../types";

const RUNTIME_EVENT_TYPES = new Set(["runtime_progress", "runtime_warning"]);

export function isRuntimeActivityEvent(event: AgentEventPayload): boolean {
  return RUNTIME_EVENT_TYPES.has(event.type) && event.runtime !== undefined;
}

export function isSessionErrorEvent(event: AgentEventPayload): boolean {
  return event.type === "error" && (hasText(event.error) || hasText(event.failure?.summary));
}

function hasText(value: string | undefined): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function formatDuration(seconds: number | undefined): string | null {
  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds < 0) {
    return null;
  }

  const wholeSeconds = Math.floor(seconds);
  if (wholeSeconds < 60) {
    return `${wholeSeconds}s`;
  }

  const wholeMinutes = Math.floor(wholeSeconds / 60);
  if (wholeMinutes < 60) {
    return `${wholeMinutes}m`;
  }

  const hours = Math.floor(wholeMinutes / 60);
  const minutes = wholeMinutes % 60;
  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`;
}

function humanizeKind(kind: string | undefined): string | null {
  const normalized = kind?.trim();
  if (!normalized) {
    return null;
  }
  return normalized.replaceAll("_", " ");
}

function describeActivity(activity: RuntimeActivityPayload | undefined): string {
  if (!activity) {
    return "Waiting for runtime activity";
  }

  if (activity.current_tool?.trim()) {
    return `Using ${activity.current_tool.trim()}`;
  }

  if (activity.last_activity_detail?.trim()) {
    return activity.last_activity_detail.trim();
  }

  return humanizeKind(activity.last_activity_kind) ?? "Runtime activity";
}

function activityMeta(activity: RuntimeActivityPayload | undefined): string | null {
  const elapsed = formatDuration(activity?.elapsed_seconds);
  const idle = formatDuration(activity?.idle_seconds);
  if (elapsed && idle) {
    return `${elapsed} elapsed, ${idle} idle`;
  }
  if (elapsed) {
    return `${elapsed} elapsed`;
  }
  if (idle) {
    return `${idle} idle`;
  }
  return null;
}

function normalizeErrorText(error: string | undefined): string | null {
  if (!hasText(error)) {
    return null;
  }

  const trimmed = error.trim();
  try {
    const parsed: unknown = JSON.parse(trimmed);
    if (typeof parsed === "object" && parsed !== null && "data" in parsed) {
      const data = (parsed as { data?: unknown }).data;
      if (typeof data === "object" && data !== null && "error" in data) {
        const nested = (data as { error?: unknown }).error;
        if (typeof nested === "string" && nested.trim().length > 0) {
          return nested.trim();
        }
      }
    }
    if (typeof parsed === "object" && parsed !== null && "message" in parsed) {
      const message = (parsed as { message?: unknown }).message;
      if (typeof message === "string" && message.trim().length > 0) {
        return message.trim();
      }
    }
  } catch {
    return trimmed;
  }

  return trimmed;
}

function sessionErrorDescription(event: AgentEventPayload): string {
  return (
    normalizeErrorText(event.error) ||
    normalizeErrorText(event.failure?.summary) ||
    "The session stopped before completing this turn."
  );
}

export function RuntimeActivityNotice({ event }: { event: AgentEventPayload }) {
  if (isSessionErrorEvent(event)) {
    const failureKind = event.failure?.kind?.trim();

    return (
      <Alert
        role="alert"
        data-testid="session-error-notice"
        data-tone="danger"
        className="my-2 max-w-3xl px-3 py-2"
        variant="danger"
      >
        <AlertCircle aria-hidden="true" className="mt-0.5 size-3 shrink-0" />
        <AlertTitle>Session failed</AlertTitle>
        {failureKind ? (
          <AlertMeta data-testid="session-error-meta">
            <Pill mono tone="danger">
              {failureKind}
            </Pill>
          </AlertMeta>
        ) : null}
        <AlertDescription className="break-words" data-testid="session-error-detail">
          {sessionErrorDescription(event)}
        </AlertDescription>
      </Alert>
    );
  }

  if (!isRuntimeActivityEvent(event)) {
    return null;
  }

  const isWarning = event.type === "runtime_warning";
  const activity = event.runtime;
  const Icon = isWarning ? AlertTriangle : Activity;
  const title = event.text?.trim() || (isWarning ? "Runtime warning" : "Still working");
  const detail = describeActivity(activity);
  const meta = activityMeta(activity);

  return (
    <Alert
      role={isWarning ? "alert" : "status"}
      data-testid="runtime-activity-notice"
      data-tone={isWarning ? "warning" : "progress"}
      className="my-2 max-w-3xl px-3 py-2"
      variant={isWarning ? "warning" : "accent"}
    >
      <Icon aria-hidden="true" className="mt-0.5 size-3 shrink-0" />
      <AlertTitle>{title}</AlertTitle>
      {meta ? <AlertMeta data-testid="runtime-activity-meta">{meta}</AlertMeta> : null}
      <AlertDescription className="truncate" data-testid="runtime-activity-detail">
        {detail}
      </AlertDescription>
    </Alert>
  );
}

export function SessionActivityInline({ activity }: { activity?: RuntimeActivityPayload | null }) {
  if (!activity) {
    return null;
  }

  const detail = describeActivity(activity);
  const idle = formatDuration(activity.idle_seconds);

  return (
    <span
      data-testid="session-activity-inline"
      className={cn(
        "hidden min-w-0 max-w-80 items-center gap-1.5 rounded-sm border px-2 py-1 md:flex",
        "border-line bg-canvas",
        "text-eyebrow text-muted"
      )}
    >
      {activity.current_tool ? (
        <Wrench aria-hidden="true" className="size-3 shrink-0 text-accent" />
      ) : (
        <Clock aria-hidden="true" className="size-3 shrink-0 text-subtle" />
      )}
      <span className="truncate" title={detail}>
        {detail}
      </span>
      {idle ? (
        <Pill mono tone="neutral" className="shrink-0">
          {idle}
        </Pill>
      ) : null}
    </span>
  );
}
