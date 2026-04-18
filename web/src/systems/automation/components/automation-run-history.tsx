import { Link } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { StatusDot, type StatusDotTone } from "@agh/ui";

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

function dotToneForRun(run: AutomationRun): StatusDotTone {
  return automationStatusTone(run.status);
}

function statusLabel(run: AutomationRun): string {
  return `${run.status.charAt(0).toUpperCase()}${run.status.slice(1)}`;
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
    <section
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
      data-testid="automation-run-history"
    >
      <div className="flex items-center justify-between gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <div className="flex items-center gap-2">
          <h3 className="font-mono text-[0.68rem] font-semibold tracking-[0.16em] text-[color:var(--color-text-label)] uppercase">
            {title}
          </h3>
          <span className="inline-flex h-5 items-center rounded-md bg-[color:var(--color-surface)] px-1.5 font-mono text-[0.64rem] text-[color:var(--color-text-secondary)]">
            {runs.length}
          </span>
        </div>
      </div>

      {isLoading ? (
        <div
          className="flex min-h-28 items-center justify-center px-4 py-8"
          data-testid="automation-run-history-loading"
        >
          <Loader2 className="size-4 animate-spin text-[color:var(--color-text-tertiary)]" />
        </div>
      ) : error ? (
        <div
          className="px-4 py-8 text-sm text-[color:var(--color-danger)]"
          data-testid="automation-run-history-error"
        >
          Failed to load automation runs
        </div>
      ) : runs.length === 0 ? (
        <div className="px-4 py-8 text-center" data-testid="automation-run-history-empty">
          <p className="text-lg font-medium text-[color:var(--color-text-primary)]">{emptyTitle}</p>
          <p className="mt-2 text-sm text-[color:var(--color-text-secondary)]">
            {emptyDescription}
          </p>
        </div>
      ) : (
        <div className="space-y-3 px-4 py-4">
          {runs.map(run => (
            <div
              key={run.id}
              className="rounded-xl bg-[color:var(--color-surface)] px-4 py-3"
              data-testid={`automation-run-${run.id}`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusDot tone={dotToneForRun(run)} />
                    <span className="text-lg font-medium text-[color:var(--color-text-primary)]">
                      {statusLabel(run)}
                    </span>
                    <span className="text-sm text-[color:var(--color-text-secondary)]">
                      {`attempt ${run.attempt}`}
                    </span>
                  </div>
                  <p className="mt-1 font-mono text-[0.78rem] text-[color:var(--color-text-secondary)]">
                    {run.session_id ?? "session pending"}
                  </p>
                  {run.session_id ? (
                    <div className="mt-3">
                      <Link
                        className="inline-flex h-8 items-center rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-3 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-surface-panel)]"
                        data-testid={`automation-run-session-link-${run.id}`}
                        params={{ id: run.session_id }}
                        to="/session/$id"
                      >
                        View Session
                      </Link>
                    </div>
                  ) : null}
                  {run.error ? (
                    <p className="mt-2 text-sm leading-6 text-[color:var(--color-danger)]">
                      {run.error}
                    </p>
                  ) : null}
                </div>

                <div className="shrink-0 text-right">
                  <p className="text-sm text-[color:var(--color-text-secondary)]">
                    {formatDateTime(run.started_at)}
                  </p>
                  <p className="mt-1 text-sm text-[color:var(--color-text-tertiary)]">
                    {formatRunDuration(run)}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
