import { AlertCircle, ChevronRight, History } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Empty, Eyebrow, Pill, Section, Spinner } from "@agh/ui";

import {
  automationStatusTone,
  formatDateTime,
  formatRunDuration,
} from "../lib/automation-formatters";
import type { AutomationRun } from "../types";

interface AutomationRunHistoryProps {
  emptyDescription?: string;
  emptyTitle?: string;
  error: Error | null;
  isLoading: boolean;
  runs: AutomationRun[];
  title?: string;
}

function runStatusLabel(run: AutomationRun): string {
  return run.status.toUpperCase();
}

interface AutomationRunRowProps {
  run: AutomationRun;
}

function AutomationRunRow({ run }: AutomationRunRowProps) {
  const tone = automationStatusTone(run.status);
  const pulse = run.status === "running";
  const startedAt = formatDateTime(run.started_at);
  const duration = formatRunDuration(run);
  const statusLabel = runStatusLabel(run);
  const testId = `automation-run-${run.id}`;
  const ariaLabel = `${statusLabel} run · attempt ${run.attempt} · started ${startedAt} · duration ${duration}`;

  const body = (
    <>
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <Pill.Dot pulse={pulse} tone={tone} />
          <Pill mono tone={tone}>
            {statusLabel}
          </Pill>
          <span className="font-mono text-eyebrow text-(--subtle)">attempt {run.attempt}</span>
        </div>
        {run.error ? <p className="text-xs leading-relaxed text-(--danger)">{run.error}</p> : null}
        {run.delivery_error ? (
          <p className="text-xs leading-relaxed text-(--danger)">
            {`Delivery: ${run.delivery_error}`}
          </p>
        ) : null}
        {run.fire_id ? (
          <p className="break-all font-mono text-badge text-(--subtle)">{run.fire_id}</p>
        ) : null}
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1 text-right">
        <span className="text-small-body text-(--muted)">{startedAt}</span>
        {run.scheduled_at ? (
          <Eyebrow className="text-(--subtle)">
            {`scheduled ${formatDateTime(run.scheduled_at)}`}
          </Eyebrow>
        ) : null}
        <span className="font-mono text-eyebrow text-(--subtle)">{duration}</span>
      </div>
    </>
  );

  if (run.session_id) {
    return (
      <Link
        aria-label={ariaLabel}
        className="group/run-row flex min-w-0 items-start gap-4 border-b border-(--line) px-4 py-3 text-left text-(--fg) transition-colors duration-(--dur) ease-(--ease) last:border-b-0 hover:bg-(--hover) focus-visible:bg-(--hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-(--line-strong)"
        data-testid={testId}
        params={{ id: run.session_id }}
        to="/session/$id"
      >
        {body}
        <ChevronRight
          aria-hidden="true"
          className="ml-2 mt-1 size-3 shrink-0 text-(--subtle) transition-colors duration-(--dur) ease-(--ease) group-hover/run-row:text-(--fg)"
          strokeWidth={1.75}
          width={12}
          height={12}
        />
      </Link>
    );
  }

  return (
    <div
      aria-label={ariaLabel}
      className="flex min-w-0 items-start gap-4 border-b border-(--line) px-4 py-3 last:border-b-0"
      data-testid={testId}
    >
      {body}
      <span
        aria-hidden="true"
        className="ml-2 mt-1 inline-flex shrink-0 items-center font-mono text-eyebrow text-(--subtle)"
      >
        pending
      </span>
    </div>
  );
}

export function AutomationRunHistory({
  emptyDescription = "Runs will appear here after the first execution.",
  emptyTitle = "No runs recorded yet",
  error,
  isLoading,
  runs,
  title = "Runs",
}: AutomationRunHistoryProps) {
  return (
    <Section
      data-testid="automation-run-history"
      label={title}
      right={<Pill mono>{runs.length}</Pill>}
    >
      {isLoading ? (
        <div
          className="flex min-h-28 items-center justify-center rounded-md bg-(--canvas-soft) px-4 py-8"
          data-testid="automation-run-history-loading"
        >
          <Spinner className="text-(--subtle)" />
        </div>
      ) : error ? (
        <div className="flex justify-center px-2 py-6" data-testid="automation-run-history-error">
          <Empty
            description={error.message ?? "Failed to load automation runs"}
            icon={AlertCircle}
            title="Unable to load runs"
            fill={false}
          />
        </div>
      ) : runs.length === 0 ? (
        <div className="flex justify-center px-2 py-6" data-testid="automation-run-history-empty">
          <Empty description={emptyDescription} icon={History} title={emptyTitle} fill={false} />
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg bg-(--canvas-soft)"
          data-testid="automation-run-history-rows"
          role="list"
        >
          {runs.map(run => (
            <div key={run.id} role="listitem">
              <AutomationRunRow run={run} />
            </div>
          ))}
        </div>
      )}
    </Section>
  );
}
