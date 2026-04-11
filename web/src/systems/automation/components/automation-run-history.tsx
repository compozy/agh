import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { automationStatusTone, formatDateTime, formatRunTitle } from "../lib/automation-formatters";
import type { AutomationRun } from "../types";

interface AutomationRunHistoryProps {
  error: Error | null;
  isLoading: boolean;
  runs: AutomationRun[];
  title?: string;
}

const TONE_CLASSES = {
  accent: "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
  success: "bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
  warning: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
  danger: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
  neutral: "bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-tertiary)]",
} as const;

export function AutomationRunHistory({
  error,
  isLoading,
  runs,
  title = "Run history",
}: AutomationRunHistoryProps) {
  return (
    <section
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4"
      data-testid="automation-run-history"
    >
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="space-y-1">
          <h3 className="font-mono text-[10px] font-semibold uppercase tracking-[0.1em] text-[color:var(--color-text-label)]">
            {title}
          </h3>
          <p className="text-sm text-[color:var(--color-text-secondary)]">
            Recent dispatch results for the selected automation.
          </p>
        </div>
        <span className="font-mono text-[0.65rem] text-[color:var(--color-text-tertiary)]">
          {runs.length}
        </span>
      </div>

      {isLoading ? (
        <div
          className="flex min-h-28 items-center justify-center"
          data-testid="automation-run-history-loading"
        >
          <Loader2 className="size-4 animate-spin text-[color:var(--color-text-tertiary)]" />
        </div>
      ) : error ? (
        <div
          className="min-h-28 rounded-lg border border-dashed border-[color:var(--color-divider)] px-4 py-6 text-sm text-[color:var(--color-danger)]"
          data-testid="automation-run-history-error"
        >
          Failed to load automation runs
        </div>
      ) : runs.length === 0 ? (
        <div
          className="min-h-28 rounded-lg border border-dashed border-[color:var(--color-divider)] px-4 py-6 text-sm text-[color:var(--color-text-secondary)]"
          data-testid="automation-run-history-empty"
        >
          No runs recorded yet.
        </div>
      ) : (
        <div className="space-y-2">
          {runs.map(run => (
            <div
              key={run.id}
              className="rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-3 py-3"
              data-testid={`automation-run-${run.id}`}
            >
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={cn(
                    "inline-flex h-[22px] items-center rounded-md px-2 font-mono text-[10px] font-semibold uppercase tracking-[0.08em]",
                    TONE_CLASSES[automationStatusTone(run.status)]
                  )}
                >
                  {run.status}
                </span>
                <span className="text-sm font-medium text-[color:var(--color-text-primary)]">
                  {formatRunTitle(run)}
                </span>
              </div>
              <div className="mt-2 grid gap-1 text-xs text-[color:var(--color-text-secondary)] md:grid-cols-2">
                <span>Started {formatDateTime(run.started_at)}</span>
                <span>Ended {formatDateTime(run.ended_at)}</span>
                <span>Session {run.session_id ?? "pending"}</span>
                <span>ID {run.id}</span>
              </div>
              {run.error ? (
                <p className="mt-2 text-xs leading-relaxed text-[color:var(--color-danger)]">
                  {run.error}
                </p>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
