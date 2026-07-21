import { taskRunCanRecover } from "./task-run-recovery";
import { taskCanRecover, taskHasApprovalPending, taskIsDraft } from "./task-formatters";
import type { TaskDetailView, TaskRun } from "../types";

/**
 * The single accent action for the task head (§6 of the redesign plan).
 * Exactly one primary per state; every kind maps to a real HTTP/CLI verb
 * (publish / approve / enqueue / retry / recover / resume) or a navigation
 * (open_run). `null` means the state carries no primary action.
 */
export type TaskPrimaryCommand =
  | { kind: "publish" }
  | { kind: "approve" }
  | { kind: "start" }
  | { kind: "open_run"; runId: string }
  | { kind: "resume" }
  | { kind: "recover" }
  | { kind: "retry"; runId: string };

export interface TaskCommandState {
  primary: TaskPrimaryCommand | null;
  /** Ghost "Edit" beside the primary — only for quiet terminal states. */
  showEditButton: boolean;
  overflow: {
    edit: boolean;
    pause: boolean;
    resume: boolean;
    cancel: boolean;
    /** Enqueue an additional run from the overflow (failed-with-attempts path). */
    startNewRun: boolean;
    delete: boolean;
  };
}

function isOpenRunStatus(status: string | undefined | null): boolean {
  return (
    status === "queued" || status === "claimed" || status === "starting" || status === "running"
  );
}

function latestFailedRun(runs: readonly TaskRun[]): TaskRun | null {
  let best: TaskRun | null = null;
  for (const run of runs) {
    if (run.status !== "failed") continue;
    if (!best || run.attempt > best.attempt) best = run;
  }
  return best;
}

function attemptsUsed(runs: readonly TaskRun[], activeRun: { attempt?: number } | null): number {
  let used = typeof activeRun?.attempt === "number" ? activeRun.attempt : 0;
  for (const run of runs) {
    if (typeof run.attempt === "number" && run.attempt > used) used = run.attempt;
  }
  return used;
}

/**
 * Derives the §6 primary-action state machine from runtime truth. Precedence:
 * recover > publish > approve > resume > open run > retry > start. States with
 * no truthful next verb (blocked, completed, canceled, failed-exhausted) render
 * no accent target — resolving actions live on the Now-strip cards instead.
 */
export function resolveTaskCommandState(
  detail: TaskDetailView,
  runs: readonly TaskRun[]
): TaskCommandState {
  const record = detail.task;
  const activeRun = detail.summary?.active_run ?? null;
  const isDraft = taskIsDraft(record);
  const isDirectlyPaused = Boolean(record.paused);
  const isEffectivelyPaused = Boolean(detail.summary?.effective_paused ?? record.paused);
  const approvalPending = taskHasApprovalPending(record);
  const activeRunNeedsAttention = activeRun?.status === "needs_attention";
  const canRecover = activeRunNeedsAttention
    ? taskRunCanRecover(activeRun, record.max_attempts)
    : taskCanRecover(record);
  const hasOpenRun = isOpenRunStatus(activeRun?.status) || activeRunNeedsAttention;
  const maxAttempts = record.max_attempts ?? null;
  const used = attemptsUsed(runs, activeRun);
  const attemptsRemain = maxAttempts === null || used < maxAttempts;
  const failedRun = record.status === "failed" ? latestFailedRun(runs) : null;
  const isTerminal =
    record.status === "completed" ||
    record.status === "canceled" ||
    (record.status === "failed" && (!attemptsRemain || !failedRun));

  let primary: TaskPrimaryCommand | null = null;
  if (record.status === "needs_attention" || activeRunNeedsAttention) {
    primary = canRecover ? { kind: "recover" } : null;
  } else if (isDraft) {
    primary = { kind: "publish" };
  } else if (approvalPending) {
    primary = { kind: "approve" };
  } else if (isDirectlyPaused) {
    primary = { kind: "resume" };
  } else if (activeRun && isOpenRunStatus(activeRun.status)) {
    primary = { kind: "open_run", runId: activeRun.id };
  } else if (record.status === "failed" && attemptsRemain && failedRun) {
    primary = { kind: "retry", runId: failedRun.id };
  } else if (
    !hasOpenRun &&
    !isEffectivelyPaused &&
    record.status !== "blocked" &&
    (record.status === "pending" || record.status === "ready")
  ) {
    primary = { kind: "start" };
  }

  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";

  return {
    primary,
    showEditButton: isTerminal,
    overflow: {
      edit: !isTerminal,
      pause: !isDirectlyPaused && !isTerminal && !isDraft,
      resume: isDirectlyPaused,
      cancel: canCancel,
      startNewRun: record.status === "failed" && attemptsRemain && !hasOpenRun,
      delete: true,
    },
  };
}

export { attemptsUsed as taskAttemptsUsed };
