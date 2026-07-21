import { ArrowUpRight } from "lucide-react";
import type { ReactNode } from "react";

import { Button, Time } from "@agh/ui";

import { useLiveElapsed } from "../hooks/use-live-elapsed";
import { computeElapsed } from "../lib/task-formatters";
import { taskAttemptsUsed } from "../lib/task-command-state";
import type { TaskDetailView, TaskRun } from "../types";
import { TaskStateBand } from "./task-state-band";

type ActiveRun = NonNullable<NonNullable<TaskDetailView["summary"]>["active_run"]>;

export interface TaskNowStripHandlers {
  onOpenRun: (runId: string) => void;
  onOpenTask: (taskId: string) => void;
  onApprove: () => void;
  onReject: () => void;
  onResume: () => void;
  onRecover: () => void;
  onClearBlock: (blockId: string) => void;
  onViewResult: () => void;
}

export interface TaskNowStripProps {
  detail: TaskDetailView;
  runs: readonly TaskRun[];
  handlers: TaskNowStripHandlers;
  pending?: {
    approve?: boolean;
    reject?: boolean;
    resume?: boolean;
    recover?: boolean;
    clearBlock?: boolean;
  };
}

function ActiveRunCard({
  run,
  maxAttempts,
  onOpenRun,
}: {
  run: ActiveRun;
  maxAttempts?: number | null;
  onOpenRun: (runId: string) => void;
}) {
  const isRunning = run.status === "running" || run.status === "starting";
  const elapsed = useLiveElapsed(run.started_at ?? undefined, isRunning);
  const attempts = maxAttempts
    ? `Attempt ${run.attempt} of ${maxAttempts}`
    : `Attempt ${run.attempt}`;
  const title = isRunning
    ? `${attempts} is running`
    : run.status === "claimed"
      ? `${attempts} is assigned`
      : `${attempts} is queued`;
  const claimant = run.claimed_by?.ref;

  return (
    <section
      data-slot="task-active-run-card"
      data-testid="tasks-detail-now-run"
      className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 rounded-lg border border-accent-dim bg-canvas-soft px-4 py-3.5"
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2.5 text-[13.5px] font-medium text-fg-strong">
          <span
            aria-hidden="true"
            className="size-[7px] shrink-0 rounded-full bg-accent motion-safe:animate-pulse"
          />
          {title}
        </div>
        <p className="mt-1 text-small-body text-muted">
          {claimant ? (
            <>
              <span className="font-medium text-fg">{claimant}</span> picked this up
            </>
          ) : (
            <>Waiting for a worker to pick this up</>
          )}
          {run.started_at ? (
            <>
              {" "}
              · started <Time iso={run.started_at} mode="relative" />
            </>
          ) : null}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3.5">
        {elapsed ? (
          <span
            aria-label="Elapsed"
            className="font-mono text-form-label tabular-nums text-muted"
            data-testid="tasks-detail-now-elapsed"
          >
            {elapsed}
          </span>
        ) : null}
        <Button
          data-testid="tasks-detail-now-open-run"
          onClick={() => onOpenRun(run.id)}
          size="sm"
          type="button"
          variant="ghost"
        >
          Open run
          <ArrowUpRight aria-hidden="true" className="size-3" />
        </Button>
      </div>
    </section>
  );
}

function dependencyTitle(
  detail: TaskDetailView,
  ids: readonly string[] | undefined
): string | null {
  const first = ids?.[0];
  if (!first) return null;
  const ref = detail.dependency_references?.find(entry => entry.depends_on.id === first);
  const dep = ref?.depends_on;
  if (!dep) return null;
  return dep.title ?? dep.identifier ?? dep.id;
}

/**
 * The Overview "Now" strip (§4.3): renders only while something demands
 * attention — approval gate, blocking causes, a stuck or active run, or the
 * terminal outcome. Quiet page otherwise (returns null).
 */
export function TaskNowStrip({ detail, runs, handlers, pending = {} }: TaskNowStripProps) {
  const record = detail.task;
  const activeRun = detail.summary?.active_run ?? null;
  const bands: ReactNode[] = [];

  if (record.approval_state === "pending") {
    bands.push(
      <TaskStateBand
        key="approval"
        actions={
          <>
            <Button
              data-testid="tasks-detail-now-reject"
              disabled={pending.reject}
              onClick={handlers.onReject}
              size="sm"
              type="button"
              variant="ghost"
            >
              Reject
            </Button>
            <Button
              data-testid="tasks-detail-now-approve"
              disabled={pending.approve}
              onClick={handlers.onApprove}
              size="sm"
              type="button"
              variant="neutral"
            >
              Approve
            </Button>
          </>
        }
        body="A manual check is required before this task can start."
        data-testid="tasks-detail-now-approval"
        title="Waiting for your approval"
        tone="info"
      />
    );
  }

  for (const reason of record.blocked_reasons ?? []) {
    if (reason.source === "approval") continue;
    if (reason.source === "dependency") {
      const title = dependencyTitle(detail, reason.depends_on_task_ids);
      const firstId = reason.depends_on_task_ids?.[0];
      bands.push(
        <TaskStateBand
          key={`dep-${firstId ?? "unknown"}`}
          actions={
            firstId ? (
              <Button
                data-testid="tasks-detail-now-open-blocking"
                onClick={() => handlers.onOpenTask(firstId)}
                size="sm"
                type="button"
                variant="neutral"
              >
                Open blocking task
              </Button>
            ) : undefined
          }
          body={
            title
              ? `“${title}” has to finish first.`
              : "Another task has to finish before this one can start."
          }
          data-testid="tasks-detail-now-dependency"
          title="Waits on another task"
          tone="warning"
        />
      );
      continue;
    }
    if (reason.source === "paused") {
      bands.push(
        <TaskStateBand
          key="paused"
          actions={
            <Button
              data-testid="tasks-detail-now-resume"
              disabled={pending.resume}
              onClick={handlers.onResume}
              size="sm"
              type="button"
              variant="neutral"
            >
              Resume
            </Button>
          }
          body={record.paused_reason || "New runs stay queued until the task is resumed."}
          data-testid="tasks-detail-now-paused"
          title="Paused"
          tone="warning"
        />
      );
      continue;
    }
    const blockId = reason.block_id;
    bands.push(
      <TaskStateBand
        key={`block-${blockId ?? reason.source}`}
        actions={
          blockId ? (
            <Button
              data-testid="tasks-detail-now-clear-block"
              disabled={pending.clearBlock}
              onClick={() => handlers.onClearBlock(blockId)}
              size="sm"
              type="button"
              variant="neutral"
            >
              Clear block
            </Button>
          ) : undefined
        }
        body={reason.reason || "A block is holding this task."}
        data-testid="tasks-detail-now-block"
        title={reason.kind === "needs_input" ? "Needs input to continue" : "Blocked"}
        tone="warning"
      />
    );
  }

  const stuck = record.status === "needs_attention" || activeRun?.status === "needs_attention";
  if (stuck) {
    const heartbeat = activeRun?.heartbeat_at;
    bands.push(
      <TaskStateBand
        key="stuck"
        actions={
          <Button
            data-testid="tasks-detail-now-recover"
            disabled={pending.recover}
            onClick={handlers.onRecover}
            size="sm"
            type="button"
            variant="neutral"
          >
            Recover
          </Button>
        }
        body={
          <>
            {record.needs_attention_reason?.trim() || "The agent stopped responding."}
            {heartbeat ? (
              <>
                {" "}
                Last heartbeat <Time iso={heartbeat} mode="relative" />. Recover requeues the work
                as a fresh attempt; nothing done so far is lost.
              </>
            ) : (
              <> Recover requeues the work as a fresh attempt; nothing done so far is lost.</>
            )}
          </>
        }
        data-testid="tasks-detail-now-stuck"
        micro={activeRun ? `task_run_stuck · ${activeRun.id}` : undefined}
        title="This attempt looks stuck"
        tone="danger"
      />
    );
  }

  if (activeRun && !stuck && activeRun.status !== "needs_attention") {
    bands.push(
      <ActiveRunCard
        key="active"
        maxAttempts={record.max_attempts}
        onOpenRun={handlers.onOpenRun}
        run={activeRun}
      />
    );
  }

  if (!activeRun && record.status === "failed") {
    const lastFailed = [...runs]
      .filter(run => run.status === "failed")
      .sort((a, b) => b.attempt - a.attempt)[0];
    const used = taskAttemptsUsed(runs, null);
    const attempts = record.max_attempts
      ? `attempt ${used} of ${record.max_attempts}`
      : `attempt ${used}`;
    bands.push(
      <TaskStateBand
        key="failed"
        actions={
          lastFailed ? (
            <Button
              data-testid="tasks-detail-now-open-failed-run"
              onClick={() => handlers.onOpenRun(lastFailed.id)}
              size="sm"
              type="button"
              variant="neutral"
            >
              Open run
            </Button>
          ) : undefined
        }
        body={
          lastFailed?.error
            ? `${lastFailed.error} Retry to queue a new attempt, or open the run to see what happened.`
            : "Retry to queue a new attempt, or open the run to see what happened."
        }
        data-testid="tasks-detail-now-failed"
        micro={lastFailed ? `task.run_failed · ${lastFailed.id}` : undefined}
        title={`Failed on ${attempts}`}
        tone="danger"
      />
    );
  }

  if (record.status === "completed") {
    const lastCompleted = [...runs]
      .filter(run => run.status === "completed")
      .sort((a, b) => b.attempt - a.attempt)[0];
    const duration = lastCompleted ? computeElapsed(lastCompleted) : undefined;
    const attemptCount = lastCompleted?.attempt ?? taskAttemptsUsed(runs, null);
    bands.push(
      <TaskStateBand
        key="completed"
        actions={
          lastCompleted ? (
            <Button
              data-testid="tasks-detail-now-view-result"
              onClick={handlers.onViewResult}
              size="sm"
              type="button"
              variant="neutral"
            >
              View result
            </Button>
          ) : undefined
        }
        body={
          <>
            Finished{" "}
            {record.closed_at ? <Time iso={record.closed_at} mode="relative" /> : "recently"}
            {duration ? ` in ${duration}` : null}.
          </>
        }
        data-testid="tasks-detail-now-completed"
        title={`Completed in ${attemptCount} ${attemptCount === 1 ? "attempt" : "attempts"}`}
        tone="success"
      />
    );
  }

  if (record.status === "canceled") {
    bands.push(
      <TaskStateBand
        key="canceled"
        body={
          record.closed_at ? (
            <>
              Canceled <Time iso={record.closed_at} mode="relative" />. No further runs will start.
            </>
          ) : (
            "No further runs will start."
          )
        }
        data-testid="tasks-detail-now-canceled"
        title="Canceled"
        tone="neutral"
      />
    );
  }

  if (bands.length === 0) return null;

  return (
    <div className="flex flex-col gap-2.5" data-testid="tasks-detail-now-strip">
      {bands}
    </div>
  );
}
