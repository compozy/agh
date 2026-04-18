import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, Loader2 } from "lucide-react";

import { Pill } from "@agh/ui";

import { formatRelativeTime, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRun } from "../types";

import { pillVariantFromTone } from "@/lib/pill-variant";
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
      <div
        className="flex min-h-[240px] flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-detail-runs-error"
      >
        <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
        <p className="text-sm text-[color:var(--color-text-secondary)]">{errorMessage}</p>
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div
        className="flex min-h-[240px] flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-detail-runs-empty"
      >
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          No runs yet. Enqueue a run to execute this task.
        </p>
      </div>
    );
  }

  return (
    <section
      aria-label="Task runs"
      className="flex min-h-0 flex-1 flex-col"
      data-testid="tasks-detail-runs-panel"
    >
      <ol className="flex flex-col divide-y divide-[color:var(--color-divider)]">
        {runs.map(run => (
          <li
            className="flex items-center gap-3 px-6 py-3 hover:bg-[color:var(--color-surface)]"
            data-testid={`tasks-detail-runs-item-${run.id}`}
            key={run.id}
          >
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]">
                <Pill variant={pillVariantFromTone(taskRunStatusTone(run.status))}>
                  {run.status}
                </Pill>
                <span className="font-mono text-[color:var(--color-text-primary)]">{run.id}</span>
                <span>attempt {run.attempt}</span>
                {run.session_id ? (
                  <span className="font-mono">session {run.session_id}</span>
                ) : null}
                {run.claimed_by?.ref ? <span>· claimed by {run.claimed_by.ref}</span> : null}
              </div>
              <div className="flex flex-wrap items-center gap-3 text-[0.66rem] font-mono uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                <span>queued {formatRelativeTime(run.queued_at)}</span>
                {run.started_at ? <span>started {formatRelativeTime(run.started_at)}</span> : null}
                {run.ended_at ? <span>ended {formatRelativeTime(run.ended_at)}</span> : null}
              </div>
              {run.error ? (
                <p
                  className="text-xs text-[color:var(--color-danger)]"
                  data-testid={`tasks-detail-runs-error-${run.id}`}
                >
                  {run.error}
                </p>
              ) : null}
            </div>
            <Link
              aria-label={`Open run ${run.id}`}
              className="flex shrink-0 items-center gap-1 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
              data-testid={`tasks-detail-runs-link-${run.id}`}
              params={{ id: taskId, runId: run.id }}
              to="/tasks/$id/runs/$runId"
            >
              Open
              <ChevronRight className="size-3" />
            </Link>
          </li>
        ))}
      </ol>
    </section>
  );
}
