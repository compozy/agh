import { Link } from "@tanstack/react-router";

import { CodeBlock, Metric, Pill, Section, Table, TableBody, TableCell, TableRow } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

export interface TaskRunIdentityPanelProps {
  run: TaskRunDetailView;
}

function IdentityRow({
  label,
  children,
  testId,
}: {
  label: string;
  children: React.ReactNode;
  testId?: string;
}) {
  return (
    <TableRow>
      <TableCell className="w-[140px] pl-4 align-middle font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
        {label}
      </TableCell>
      <TableCell className="pr-4 align-middle text-[13px] text-[color:var(--color-text-primary)]">
        <span data-testid={testId}>{children}</span>
      </TableCell>
    </TableRow>
  );
}

function normalizeSessionText(value?: string | null): string {
  return typeof value === "string" ? value.trim() : "";
}

export function TaskRunIdentityPanel({ run }: TaskRunIdentityPanelProps) {
  const record = run.run;
  const session = run.session;
  const linkedSessionID = normalizeSessionText(session?.session_id ?? record.session_id);
  const linkedSessionAgent = normalizeSessionText(session?.agent_name);

  return (
    <Section aria-label="Run identity" data-testid="task-run-detail-identity" label="Run identity">
      <div className="rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
        <Table className="text-[13px]">
          <TableBody>
            <IdentityRow label="Run ID" testId="task-run-detail-identity-run">
              <Pill mono>{record.id}</Pill>
            </IdentityRow>
            <IdentityRow label="Status">
              <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(record.status))}>
                {record.status}
              </Pill>
            </IdentityRow>
            <IdentityRow label="Attempt" testId="task-run-detail-identity-attempt">
              {record.attempt}
            </IdentityRow>
            {record.idempotency_key ? (
              <IdentityRow label="Idempotency" testId="task-run-detail-identity-idempotency">
                <Pill mono>{record.idempotency_key}</Pill>
              </IdentityRow>
            ) : null}
            {record.claimed_by?.ref ? (
              <IdentityRow label="Claimed by" testId="task-run-detail-identity-claimed-by">
                {record.claimed_by.ref}
              </IdentityRow>
            ) : null}
            <IdentityRow label="Session">
              {linkedSessionID && linkedSessionAgent ? (
                <Link
                  className="font-mono text-[12px] text-[color:var(--color-accent)] hover:underline"
                  data-testid="task-run-detail-session-link"
                  params={{ name: linkedSessionAgent, id: linkedSessionID }}
                  to="/agents/$name/sessions/$id"
                >
                  {linkedSessionID}
                </Link>
              ) : linkedSessionID ? (
                <Link
                  className="font-mono text-[12px] text-[color:var(--color-accent)] hover:underline"
                  data-testid="task-run-detail-session-link"
                  params={{ id: linkedSessionID }}
                  to="/session/$id"
                >
                  {linkedSessionID}
                </Link>
              ) : (
                <span
                  className="text-[color:var(--color-text-tertiary)]"
                  data-testid="task-run-detail-session-missing"
                >
                  None
                </span>
              )}
            </IdentityRow>
          </TableBody>
        </Table>
      </div>
    </Section>
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
    <Section aria-label="Run progress" data-testid="task-run-detail-progress" label="Progress">
      <div className="grid gap-3 sm:grid-cols-2">
        <Metric
          data-testid="task-run-detail-progress-tool-calls"
          label="Tool calls"
          value={formatCount(summary?.tool_call_count)}
        />
        <Metric
          data-testid="task-run-detail-progress-turns"
          label="Turns"
          value={formatCount(summary?.turn_count)}
        />
        <Metric
          data-testid="task-run-detail-progress-input-tokens"
          label="Input tokens"
          value={formatCount(summary?.input_tokens)}
        />
        <Metric
          data-testid="task-run-detail-progress-output-tokens"
          label="Output tokens"
          value={formatCount(summary?.output_tokens)}
        />
        <Metric
          data-testid="task-run-detail-progress-total-tokens"
          label="Total tokens"
          value={formatCount(summary?.total_tokens)}
        />
        <Metric data-testid="task-run-detail-progress-elapsed" label="Elapsed" value={elapsed} />
        <Metric
          className="sm:col-span-2"
          data-testid="task-run-detail-progress-cost"
          label="Cost"
          value={formatCost(summary?.total_cost, summary?.cost_currency)}
        />
      </div>
    </Section>
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
    <Section aria-label="Run activity" data-testid="task-run-detail-activity" label="Activity">
      <div className="flex flex-col gap-3 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4">
        <dl className="flex flex-col gap-2 text-[13px]">
          {lastEventType ? (
            <div className="flex items-center justify-between gap-3">
              <dt className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
                Last event
              </dt>
              <dd>
                <Pill mono data-testid="task-run-detail-activity-event">
                  {lastEventType}
                </Pill>
              </dd>
            </div>
          ) : null}
          {lastActivityAt ? (
            <div className="flex items-center justify-between gap-3">
              <dt className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
                Last activity
              </dt>
              <dd
                className="text-[color:var(--color-text-primary)]"
                data-testid="task-run-detail-activity-timestamp"
              >
                {lastActivityAt}
              </dd>
            </div>
          ) : null}
        </dl>
        {error ? (
          <div
            className="rounded-md border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-3 py-2"
            data-testid="task-run-detail-activity-error"
          >
            <p className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-danger)]">
              Error
            </p>
            <p className="mt-1 text-[13px] text-[color:var(--color-danger)]">{error}</p>
          </div>
        ) : null}
        {result !== undefined && result !== null ? (
          <div data-testid="task-run-detail-activity-result">
            <p className="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
              Result
            </p>
            <CodeBlock
              code={JSON.stringify(result, null, 2)}
              copyable={false}
              language="json"
              showPrompt={false}
            />
          </div>
        ) : null}
      </div>
    </Section>
  );
}
