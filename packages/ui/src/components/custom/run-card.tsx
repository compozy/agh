"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";
import { MonoId } from "./mono-id";
import { Pill, type PillTone } from "./pill";
import { Time } from "./time";

export type RunCardStatus = "pending" | "in_progress" | "completed" | "failed" | "canceled";

const RUN_STATUS_TONE: Record<RunCardStatus, PillTone> = {
  pending: "neutral",
  in_progress: "info",
  completed: "success",
  failed: "danger",
  canceled: "neutral",
};

const RUN_STATUS_LABEL: Record<RunCardStatus, string> = {
  pending: "Pending",
  in_progress: "Running",
  completed: "Completed",
  failed: "Failed",
  canceled: "Canceled",
};

export type RunCardWarningTone = "warning" | "danger";

export interface RunCardWarning {
  tone: RunCardWarningTone;
  message: React.ReactNode;
}

export interface RunCardProps extends Omit<React.ComponentProps<"section">, "title"> {
  /** Run lifecycle status; resolves the pill tone + label via the internal dictionary. */
  status: RunCardStatus;
  /** Run identifier rendered as a `<MonoId>`. */
  runId: string;
  /** Optional context (e.g. "session 42 · agent claude") rendered next to the run id. */
  sessionInfo?: React.ReactNode;
  /** Optional attempt counter (1-based). Rendered as `attempt 3` when present. */
  attempt?: number;
  /** Optional inline warning rendered after the meta row. */
  warning?: RunCardWarning;
  /** Channel name (CLI, web, scheduler, …). */
  channel?: React.ReactNode;
  /** ISO timestamp of when the run entered the queue. */
  queuedAt?: string;
  /** ISO timestamp of when the run started executing. */
  startedAt?: string;
  /** Pre-formatted elapsed string (e.g. `"3m 42s"`). Reuse `formatDuration` upstream. */
  elapsed?: React.ReactNode;
}

const WARNING_TONE_CLASS: Record<RunCardWarningTone, string> = {
  warning: "bg-warning-tint text-warning",
  danger: "bg-danger-tint text-danger",
};

const PLACEHOLDER = "—";

function RunCard({
  status,
  runId,
  sessionInfo,
  attempt,
  warning,
  channel,
  queuedAt,
  startedAt,
  elapsed,
  className,
  ...props
}: RunCardProps) {
  const tone = RUN_STATUS_TONE[status];
  const statusLabel = RUN_STATUS_LABEL[status];
  return (
    <section
      data-slot="run-card"
      data-status={status}
      className={cn("flex flex-col gap-3 rounded-lg bg-canvas-soft px-[18px] py-4", className)}
      {...props}
    >
      <header
        data-slot="run-card-pills"
        className="flex flex-wrap items-center gap-2 text-[12px] text-muted"
      >
        <Pill data-slot="run-card-status" tone={tone}>
          {statusLabel}
        </Pill>
        <MonoId data-slot="run-card-id" value={runId} />
        {sessionInfo ? (
          <span data-slot="run-card-session-info" className="min-w-0 truncate text-muted">
            {sessionInfo}
          </span>
        ) : null}
        {typeof attempt === "number" ? (
          <span data-slot="run-card-attempt" className="text-muted tabular-nums">
            attempt {attempt}
          </span>
        ) : null}
      </header>
      {warning ? (
        <div
          data-slot="run-card-warning"
          data-tone={warning.tone}
          className={cn("rounded px-3 py-2 text-[12.5px]", WARNING_TONE_CLASS[warning.tone])}
          role="status"
        >
          {warning.message}
        </div>
      ) : null}
      <div data-slot="run-card-grid" className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <RunCardStat label="CHANNEL" value={channel ?? PLACEHOLDER} slot="channel" />
        <RunCardStat
          label="QUEUED"
          value={queuedAt ? <Time iso={queuedAt} mode="relative" /> : PLACEHOLDER}
          slot="queued"
        />
        <RunCardStat
          label="STARTED"
          value={startedAt ? <Time iso={startedAt} mode="relative" /> : PLACEHOLDER}
          slot="started"
        />
        <RunCardStat label="ELAPSED" value={elapsed ?? PLACEHOLDER} slot="elapsed" />
      </div>
    </section>
  );
}

function RunCardStat({
  label,
  value,
  slot,
}: {
  label: string;
  value: React.ReactNode;
  slot: string;
}) {
  return (
    <div data-slot={`run-card-${slot}`} className="flex min-w-0 flex-col gap-1">
      <Eyebrow data-slot={`run-card-${slot}-label`} className="text-muted">
        {label}
      </Eyebrow>
      <span data-slot={`run-card-${slot}-value`} className="min-w-0 truncate text-[12.5px] text-fg">
        {value}
      </span>
    </div>
  );
}

export { RunCard, RUN_STATUS_TONE, RUN_STATUS_LABEL };
