import { AlertCircle, History, Loader2 } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  Empty,
  MonoBadge,
  Section,
  StatusDot,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";

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
      right={<MonoBadge>{runs.length}</MonoBadge>}
    >
      {isLoading ? (
        <div
          className="flex min-h-28 items-center justify-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-8"
          data-testid="automation-run-history-loading"
        >
          <Loader2
            aria-hidden="true"
            className="size-4 animate-spin text-[color:var(--color-text-tertiary)]"
          />
        </div>
      ) : error ? (
        <div data-testid="automation-run-history-error">
          <Empty
            description={error.message ?? "Failed to load automation runs"}
            icon={AlertCircle}
            title="Unable to load runs"
          />
        </div>
      ) : runs.length === 0 ? (
        <div data-testid="automation-run-history-empty">
          <Empty description={emptyDescription} icon={History} title={emptyTitle} />
        </div>
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                  Status
                </TableHead>
                <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                  Attempt
                </TableHead>
                <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                  Started
                </TableHead>
                <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                  Duration
                </TableHead>
                <TableHead className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
                  Session
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {runs.map(run => {
                const tone = automationStatusTone(run.status);
                const pulse = run.status === "running";

                return (
                  <TableRow key={run.id} data-testid={`automation-run-${run.id}`}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <StatusDot pulse={pulse} tone={tone} />
                        <MonoBadge tone={tone}>{runStatusLabel(run)}</MonoBadge>
                      </div>
                      {run.error ? (
                        <p className="mt-1 text-[12px] leading-relaxed text-[color:var(--color-danger)]">
                          {run.error}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-[13px] text-[color:var(--color-text-secondary)]">
                      {run.attempt}
                    </TableCell>
                    <TableCell className="text-[13px] text-[color:var(--color-text-secondary)]">
                      {formatDateTime(run.started_at)}
                    </TableCell>
                    <TableCell className="text-[13px] text-[color:var(--color-text-secondary)]">
                      {formatRunDuration(run)}
                    </TableCell>
                    <TableCell className="text-[13px] text-[color:var(--color-text-secondary)]">
                      {run.session_id ? (
                        <Link
                          className="inline-flex h-7 items-center rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-2.5 font-mono text-[12px] text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)]"
                          data-testid={`automation-run-session-link-${run.id}`}
                          params={{ id: run.session_id }}
                          to="/session/$id"
                        >
                          View Session
                        </Link>
                      ) : (
                        <span className="font-mono text-[12px] text-[color:var(--color-text-tertiary)]">
                          pending
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </Section>
  );
}
