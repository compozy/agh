import { AlertCircle, History } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  Empty,
  Pill,
  Section,
  Spinner,
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
      right={<Pill mono>{runs.length}</Pill>}
    >
      {isLoading ? (
        <div
          className="flex min-h-28 items-center justify-center rounded-md border border-(--color-divider) bg-(--color-surface) px-4 py-8"
          data-testid="automation-run-history-loading"
        >
          <Spinner className="text-(--color-text-tertiary)" />
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
        <div className="overflow-hidden rounded-md border border-(--color-divider) bg-(--color-surface)">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
                  Status
                </TableHead>
                <TableHead className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
                  Attempt
                </TableHead>
                <TableHead className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
                  Started
                </TableHead>
                <TableHead className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
                  Duration
                </TableHead>
                <TableHead className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
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
                        <Pill.Dot pulse={pulse} tone={tone} />
                        <Pill mono tone={tone}>
                          {runStatusLabel(run)}
                        </Pill>
                      </div>
                      {run.error ? (
                        <p className="mt-1 text-xs leading-relaxed text-(--color-danger)">
                          {run.error}
                        </p>
                      ) : null}
                      {run.delivery_error ? (
                        <p className="mt-1 text-xs leading-relaxed text-(--color-danger)">
                          {`Delivery: ${run.delivery_error}`}
                        </p>
                      ) : null}
                      {run.fire_id ? (
                        <p className="mt-1 break-all font-mono text-badge text-(--color-text-tertiary)">
                          {run.fire_id}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-small-body text-(--color-text-secondary)">
                      {run.attempt}
                    </TableCell>
                    <TableCell className="text-small-body text-(--color-text-secondary)">
                      <span>{formatDateTime(run.started_at)}</span>
                      {run.scheduled_at ? (
                        <span className="mt-1 block text-eyebrow text-(--color-text-tertiary)">
                          {`scheduled ${formatDateTime(run.scheduled_at)}`}
                        </span>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-small-body text-(--color-text-secondary)">
                      {formatRunDuration(run)}
                    </TableCell>
                    <TableCell className="text-small-body text-(--color-text-secondary)">
                      {run.session_id ? (
                        <Link
                          className="inline-flex h-7 items-center rounded-md border border-(--color-divider) bg-(--color-surface-panel) px-2.5 font-mono text-xs text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
                          data-testid={`automation-run-session-link-${run.id}`}
                          params={{ id: run.session_id }}
                          to="/session/$id"
                        >
                          View Session
                        </Link>
                      ) : (
                        <span className="font-mono text-xs text-(--color-text-tertiary)">
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
