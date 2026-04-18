import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Pill } from "@/components/design-system";
import { Button } from "@agh/ui";

import { formatRelativeTime, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

export interface TaskRunDetailHeaderProps {
  run: TaskRunDetailView;
  onCancelRun?: () => void;
  isCancelPending?: boolean;
}

function formatElapsed(startedAt?: string | null, endedAt?: string | null): string | null {
  if (!startedAt) {
    return null;
  }

  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) {
    return null;
  }

  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) {
    return null;
  }

  const delta = Math.max(0, end - start);
  const totalSeconds = Math.floor(delta / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (minutes > 0) {
    return `${minutes}m ${seconds}s elapsed`;
  }

  return `${seconds}s elapsed`;
}

export function TaskRunDetailHeader({
  run,
  onCancelRun,
  isCancelPending = false,
}: TaskRunDetailHeaderProps) {
  const record = run.run;
  const task = run.task;
  const identifier = task.identifier ?? task.id;
  const canCancel =
    record.status === "queued" ||
    record.status === "claimed" ||
    record.status === "starting" ||
    record.status === "running";
  const elapsed = formatElapsed(record.started_at, record.ended_at);

  return (
    <header
      className="flex flex-col gap-4 border-b border-[color:var(--color-divider)] px-6 py-5"
      data-testid="task-run-detail-header"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-1 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
            data-testid="task-run-detail-breadcrumb"
          >
            <Link
              className="hover:text-[color:var(--color-text-secondary)]"
              data-testid="task-run-detail-breadcrumb-tasks"
              to="/tasks"
            >
              Tasks
            </Link>
            <ChevronRight className="size-3 text-[color:var(--color-text-tertiary)]" />
            <Link
              className="hover:text-[color:var(--color-text-secondary)]"
              data-testid="task-run-detail-breadcrumb-task"
              params={{ id: task.id }}
              to="/tasks/$id"
            >
              {identifier}
            </Link>
            <ChevronRight className="size-3 text-[color:var(--color-text-tertiary)]" />
            <span className="text-[color:var(--color-text-secondary)]">{record.id}</span>
          </nav>

          <div className="mt-2 flex flex-wrap items-center gap-3">
            <h1
              className="truncate text-2xl font-semibold text-[color:var(--color-text-primary)]"
              data-testid="task-run-detail-title"
            >
              Run {record.id}
            </h1>
            <Pill emphasis="strong" kind="state" tone={taskRunStatusTone(record.status)}>
              {record.status}
            </Pill>
          </div>

          <div
            className="mt-2 flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]"
            data-testid="task-run-detail-meta"
          >
            <span>attempt {record.attempt}</span>
            {record.session_id ? (
              <span className="font-mono">· session {record.session_id}</span>
            ) : null}
            {record.claimed_by?.ref ? <span>· claimed by {record.claimed_by.ref}</span> : null}
            {elapsed ? <span>· {elapsed}</span> : null}
            {record.started_at ? (
              <span>· started {formatRelativeTime(record.started_at)}</span>
            ) : null}
          </div>
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {canCancel && onCancelRun ? (
            <Button
              data-testid="task-run-detail-cancel"
              disabled={isCancelPending}
              onClick={onCancelRun}
              size="sm"
              type="button"
              variant="outline"
            >
              Kill run
            </Button>
          ) : null}
        </div>
      </div>
    </header>
  );
}
