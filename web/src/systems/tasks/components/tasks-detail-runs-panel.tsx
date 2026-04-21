import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, Inbox, Loader2 } from "lucide-react";

import {
  Empty,
  MonoBadge,
  Pill,
  Section,
  StatusDot,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { pillVariantFromTone } from "@/lib/pill-variant";

import { formatRelativeTime, taskRunStatusTone, taskStatusSignal } from "../lib/task-formatters";
import type { TaskRun } from "../types";

export interface TasksDetailRunsPanelProps {
  taskId: string;
  runs: TaskRun[];
  isLoading?: boolean;
  errorMessage?: string | null;
}

export function TasksDetailRunsPanel({
  taskId,
  runs,
  isLoading = false,
  errorMessage = null,
}: TasksDetailRunsPanelProps) {
  if (isLoading && runs.length === 0) {
    return (
      <div
        className="flex min-h-[240px] items-center justify-center"
        data-testid="tasks-detail-runs-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage && runs.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Unable to load runs"
        description={errorMessage}
        data-testid="tasks-detail-runs-error"
      />
    );
  }

  if (runs.length === 0) {
    return (
      <Empty
        icon={Inbox}
        title="No runs yet"
        description="Enqueue a run to execute this task."
        data-testid="tasks-detail-runs-empty"
      />
    );
  }

  return (
    <Section
      aria-label="Task runs"
      className="w-full gap-6 px-6 py-5"
      data-testid="tasks-detail-runs-panel"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead />
            <TableHead>Run</TableHead>
            <TableHead>Attempt</TableHead>
            <TableHead>Queued</TableHead>
            <TableHead>Ended</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {runs.map(run => {
            const signal = taskStatusSignal(run.status);
            return (
              <TableRow data-testid={`tasks-detail-runs-item-${run.id}`} key={run.id}>
                <TableCell className="w-8 pl-4">
                  <StatusDot tone={signal.tone} pulse={signal.pulse} />
                </TableCell>
                <TableCell className="max-w-[360px]">
                  <div className="flex min-w-0 flex-col gap-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <MonoBadge>{run.id}</MonoBadge>
                      <Pill variant={pillVariantFromTone(taskRunStatusTone(run.status))}>
                        {run.status}
                      </Pill>
                      {run.session_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          session {run.session_id}
                        </span>
                      ) : null}
                    </div>
                    {run.error ? (
                      <p
                        className="text-[11px] text-[color:var(--color-danger)]"
                        data-testid={`tasks-detail-runs-error-${run.id}`}
                      >
                        {run.error}
                      </p>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                  attempt {run.attempt}
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                  {formatRelativeTime(run.queued_at)}
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                  {run.ended_at ? formatRelativeTime(run.ended_at) : "—"}
                </TableCell>
                <TableCell className="w-8 pr-4">
                  <Link
                    aria-label={`Open run ${run.id}`}
                    className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
                    data-testid={`tasks-detail-runs-link-${run.id}`}
                    params={{ id: taskId, runId: run.id }}
                    to="/tasks/$id/runs/$runId"
                  >
                    Open <ChevronRight className="size-3" />
                  </Link>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Section>
  );
}
