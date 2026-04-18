import { Link } from "@tanstack/react-router";

import { Pill } from "@agh/ui";

import { taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

import { pillVariantFromTone } from "@/lib/pill-variant";
export interface TaskRunIdentityPanelProps {
  run: TaskRunDetailView;
}

function SidePanelLabel({ children }: { children: string }) {
  return (
    <span className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
      {children}
    </span>
  );
}

export function TaskRunIdentityPanel({ run }: TaskRunIdentityPanelProps) {
  const record = run.run;
  const session = run.session;

  return (
    <section
      aria-label="Run identity"
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
      data-testid="task-run-detail-identity"
    >
      <h3 className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        Run Identity
      </h3>
      <dl className="mt-3 flex flex-col gap-2 text-sm">
        <div className="flex items-center justify-between gap-4">
          <SidePanelLabel>Run ID</SidePanelLabel>
          <span
            className="truncate font-mono text-[0.78rem] text-[color:var(--color-text-primary)]"
            data-testid="task-run-detail-identity-run"
          >
            {record.id}
          </span>
        </div>
        <div className="flex items-center justify-between gap-4">
          <SidePanelLabel>Status</SidePanelLabel>
          <Pill variant={pillVariantFromTone(taskRunStatusTone(record.status))}>
            {record.status}
          </Pill>
        </div>
        <div className="flex items-center justify-between gap-4">
          <SidePanelLabel>Attempt</SidePanelLabel>
          <span
            className="text-[color:var(--color-text-primary)]"
            data-testid="task-run-detail-identity-attempt"
          >
            {record.attempt}
          </span>
        </div>
        {record.idempotency_key ? (
          <div className="flex items-center justify-between gap-4">
            <SidePanelLabel>Idempotency</SidePanelLabel>
            <span
              className="truncate font-mono text-[0.78rem] text-[color:var(--color-text-primary)]"
              data-testid="task-run-detail-identity-idempotency"
            >
              {record.idempotency_key}
            </span>
          </div>
        ) : null}
        {record.claimed_by?.ref ? (
          <div className="flex items-center justify-between gap-4">
            <SidePanelLabel>Claimed By</SidePanelLabel>
            <span
              className="text-[color:var(--color-text-primary)]"
              data-testid="task-run-detail-identity-claimed-by"
            >
              {record.claimed_by.ref}
            </span>
          </div>
        ) : null}
        <div className="flex items-center justify-between gap-4">
          <SidePanelLabel>Session</SidePanelLabel>
          {session?.session_id ? (
            <Link
              className="font-mono text-[0.78rem] text-[color:var(--color-accent)] hover:underline"
              data-testid="task-run-detail-session-link"
              params={{ id: session.session_id }}
              to="/session/$id"
            >
              {session.session_id}
            </Link>
          ) : (
            <span
              className="text-[color:var(--color-text-tertiary)]"
              data-testid="task-run-detail-session-missing"
            >
              None
            </span>
          )}
        </div>
      </dl>
    </section>
  );
}

export interface TaskRunProgressPanelProps {
  run: TaskRunDetailView;
}

function formatCount(value?: number | null): string {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "—";
  }

  return value.toLocaleString();
}

function formatElapsed(startedAt?: string | null, endedAt?: string | null): string {
  if (!startedAt) {
    return "—";
  }

  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) {
    return "—";
  }

  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) {
    return "—";
  }

  const delta = Math.max(0, end - start);
  const totalSeconds = Math.floor(delta / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }

  return `${seconds}s`;
}

function formatCost(value?: number | null, currency?: string | null): string {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "—";
  }

  const formatted = value.toLocaleString(undefined, {
    maximumFractionDigits: 4,
    minimumFractionDigits: 2,
  });

  if (currency) {
    return `${currency} ${formatted}`;
  }

  return formatted;
}

export function TaskRunProgressPanel({ run }: TaskRunProgressPanelProps) {
  const summary = run.summary;
  const record = run.run;
  const elapsed = formatElapsed(record.started_at, record.ended_at);

  return (
    <section
      aria-label="Run progress"
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
      data-testid="task-run-detail-progress"
    >
      <h3 className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        Progress
      </h3>
      <dl className="mt-3 flex flex-col gap-2 text-sm">
        <MetricRow
          label="Tool calls"
          value={formatCount(summary?.tool_call_count)}
          testId="task-run-detail-progress-tool-calls"
        />
        <MetricRow
          label="Input tokens"
          value={formatCount(summary?.input_tokens)}
          testId="task-run-detail-progress-input-tokens"
        />
        <MetricRow
          label="Output tokens"
          value={formatCount(summary?.output_tokens)}
          testId="task-run-detail-progress-output-tokens"
        />
        <MetricRow
          label="Total tokens"
          value={formatCount(summary?.total_tokens)}
          testId="task-run-detail-progress-total-tokens"
        />
        <MetricRow
          label="Turns"
          value={formatCount(summary?.turn_count)}
          testId="task-run-detail-progress-turns"
        />
        <MetricRow label="Elapsed" value={elapsed} testId="task-run-detail-progress-elapsed" />
        <MetricRow
          label="Cost"
          value={formatCost(summary?.total_cost, summary?.cost_currency)}
          testId="task-run-detail-progress-cost"
        />
      </dl>
    </section>
  );
}

interface MetricRowProps {
  label: string;
  value: string;
  testId: string;
}

function MetricRow({ label, value, testId }: MetricRowProps) {
  return (
    <div className="flex items-center justify-between gap-4" data-testid={testId}>
      <SidePanelLabel>{label}</SidePanelLabel>
      <span className="text-[color:var(--color-text-primary)]">{value}</span>
    </div>
  );
}

export interface TaskRunActivityPanelProps {
  run: TaskRunDetailView;
}

export function TaskRunActivityPanel({ run }: TaskRunActivityPanelProps) {
  const summary = run.summary;
  const record = run.run;
  const lastEventType = summary?.last_event_type;
  const lastActivityAt = summary?.last_activity_at;
  const error = record.error;
  const result = record.result;

  return (
    <section
      aria-label="Run activity"
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
      data-testid="task-run-detail-activity"
    >
      <h3 className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        Activity
      </h3>
      <dl className="mt-3 flex flex-col gap-3 text-sm">
        {lastEventType ? (
          <div className="flex items-center justify-between gap-4">
            <SidePanelLabel>Last event</SidePanelLabel>
            <span
              className="truncate font-mono text-[0.78rem] text-[color:var(--color-text-primary)]"
              data-testid="task-run-detail-activity-event"
            >
              {lastEventType}
            </span>
          </div>
        ) : null}
        {lastActivityAt ? (
          <div className="flex items-center justify-between gap-4">
            <SidePanelLabel>Last activity</SidePanelLabel>
            <span
              className="text-[color:var(--color-text-primary)]"
              data-testid="task-run-detail-activity-timestamp"
            >
              {lastActivityAt}
            </span>
          </div>
        ) : null}
      </dl>
      {error ? (
        <div
          className="mt-3 rounded-md border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-3 py-2"
          data-testid="task-run-detail-activity-error"
        >
          <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-danger)]">
            Error
          </p>
          <p className="mt-1 text-sm text-[color:var(--color-danger)]">{error}</p>
        </div>
      ) : null}
      {result !== undefined && result !== null ? (
        <details className="mt-3" data-testid="task-run-detail-activity-result">
          <summary className="cursor-pointer font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            Result
          </summary>
          <pre className="mt-2 max-h-48 overflow-auto rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-3 py-2 font-mono text-[0.72rem] text-[color:var(--color-text-primary)]">
            {JSON.stringify(result, null, 2)}
          </pre>
        </details>
      ) : null}
    </section>
  );
}
