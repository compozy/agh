import { useCallback } from "react";
import { Link, useRouter } from "@tanstack/react-router";
import { ArrowUpRight } from "lucide-react";

import { Button, DetailHeader, MonoId, Pill, Time, formatDuration } from "@agh/ui";

import { taskRunStatusLabel, taskRunStatusTone, taskStatusSignal } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

export interface TaskRunDetailHeaderProps {
  run: TaskRunDetailView;
  onCancelRun?: () => void;
  isCancelPending?: boolean;
}

function computeElapsedLabel(startedAt?: string | null, endedAt?: string | null): string | null {
  if (!startedAt) return null;
  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) return null;
  const end = endedAt ? Date.parse(endedAt) : Date.now();
  if (Number.isNaN(end)) return null;
  return formatDuration(Math.max(0, end - start));
}

function normalizeText(value?: string | null): string {
  return typeof value === "string" ? value.trim() : "";
}

export function TaskRunDetailHeader({
  run,
  onCancelRun,
  isCancelPending = false,
}: TaskRunDetailHeaderProps) {
  const router = useRouter();
  const handleBack = useCallback(() => {
    router.history.back();
  }, [router]);

  const record = run.run;
  const task = run.task;
  const session = run.session;
  const identifier = task.identifier ?? task.id;
  const canCancel =
    record.status === "queued" ||
    record.status === "claimed" ||
    record.status === "starting" ||
    record.status === "running";
  const signal = taskStatusSignal(record.status);
  const elapsedLabel = computeElapsedLabel(record.started_at, record.ended_at);
  const linkedSessionID = normalizeText(session?.session_id ?? record.session_id);
  const linkedSessionAgent = normalizeText(session?.agent_name);
  const claimedRef = normalizeText(record.claimed_by?.ref);

  return (
    <DetailHeader
      data-testid="task-run-detail-header"
      back={handleBack}
      backLabel="Back to task"
      crumbs={
        <span data-testid="task-run-detail-breadcrumb" className="inline-flex items-center gap-1.5">
          <Link
            data-testid="task-run-detail-breadcrumb-tasks"
            to="/tasks"
            className="transition-colors duration-base ease-out hover:text-fg"
          >
            Tasks
          </Link>
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
          <Link
            data-testid="task-run-detail-breadcrumb-task"
            params={{ id: task.id }}
            to="/tasks/$id"
            className="transition-colors duration-base ease-out hover:text-fg"
          >
            {identifier}
          </Link>
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
          <span>{record.id}</span>
        </span>
      }
      preTitle="Task run"
      title={
        <span
          data-testid="task-run-detail-title"
          className="inline-flex min-w-0 items-center gap-2"
        >
          <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
          <span className="truncate">Run</span>
        </span>
      }
      pills={
        <>
          <MonoId data-testid="task-run-detail-run-id" value={record.id} />
          <Pill data-testid="task-run-detail-status" tone={taskRunStatusTone(record.status)}>
            {taskRunStatusLabel(record.status)}
          </Pill>
          {elapsedLabel ? (
            <Pill data-testid="task-run-detail-duration" tone="neutral">
              {elapsedLabel}
            </Pill>
          ) : null}
        </>
      }
      meta={
        <div
          data-testid="task-run-detail-meta"
          className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1"
        >
          <span>Attempt {record.attempt}</span>
          {linkedSessionID ? (
            <>
              <span aria-hidden="true" className="text-faint">
                ·
              </span>
              <span className="inline-flex items-center gap-1.5">
                Session <MonoId size="sm" value={linkedSessionID} />
              </span>
            </>
          ) : null}
          {claimedRef ? (
            <>
              <span aria-hidden="true" className="text-faint">
                ·
              </span>
              <span>
                Claimed by <span className="text-fg">{claimedRef}</span>
              </span>
            </>
          ) : null}
          {record.started_at ? (
            <>
              <span aria-hidden="true" className="text-faint">
                ·
              </span>
              <span className="inline-flex items-center gap-1">
                Started <Time iso={record.started_at} mode="relative" />
              </span>
            </>
          ) : null}
        </div>
      }
      actions={
        <div
          data-testid="task-run-detail-actions"
          className="flex shrink-0 flex-wrap items-center gap-2"
        >
          {linkedSessionID && linkedSessionAgent ? (
            <Link
              params={{ name: linkedSessionAgent, id: linkedSessionID }}
              to="/agents/$name/sessions/$id"
            >
              <Button data-testid="task-run-detail-open-session" size="sm" variant="neutral">
                Open session
                <ArrowUpRight className="size-3" strokeWidth={1.75} />
              </Button>
            </Link>
          ) : linkedSessionID ? (
            <Link params={{ id: linkedSessionID }} to="/session/$id">
              <Button data-testid="task-run-detail-open-session" size="sm" variant="neutral">
                Open session
                <ArrowUpRight className="size-3" strokeWidth={1.75} />
              </Button>
            </Link>
          ) : null}
          {canCancel && onCancelRun ? (
            <Button
              data-testid="task-run-detail-cancel"
              disabled={isCancelPending}
              onClick={onCancelRun}
              size="sm"
              type="button"
              variant="destructive"
            >
              Cancel run
            </Button>
          ) : null}
        </div>
      }
    />
  );
}
