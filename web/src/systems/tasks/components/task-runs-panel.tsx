import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, CornerUpLeft, Play } from "lucide-react";

import { BlockLoading, Button, cn, Empty, OwnerAvatar, Pill, Time } from "@agh/ui";

import { useLiveElapsed } from "../hooks/use-live-elapsed";
import {
  computeElapsed,
  ownerAvatarKindFor,
  taskRunStatusLabel,
  taskRunStatusTone,
} from "../lib/task-formatters";
import type { TaskRun, TaskRunReview } from "../types";

export interface TaskRunsPanelProps {
  taskId: string;
  runs: TaskRun[];
  reviewsByRun?: ReadonlyMap<string, readonly TaskRunReview[]>;
  isLoading?: boolean;
  errorMessage?: string | null;
  /** Present only when the runtime allows starting a run right now. */
  onStartRun?: () => void;
  workerName?: string | null;
}

const ROW_GRID =
  "grid grid-cols-[104px_96px_minmax(0,1fr)_14px] items-center gap-3 px-4 md:grid-cols-[104px_96px_minmax(0,1fr)_88px_76px_minmax(0,1.1fr)_14px]";

function reviewLine(review: TaskRunReview): string {
  const outcome = review.outcome ?? review.status;
  const label = outcome.replaceAll("_", " ");
  return review.reason ? `Review: ${label} · ${review.reason}` : `Review: ${label}`;
}

function RunRow({
  taskId,
  run,
  lineageAttempt,
  reviews,
}: {
  taskId: string;
  run: TaskRun;
  lineageAttempt: number | null;
  reviews: readonly TaskRunReview[];
}) {
  const isActive = run.status === "running" || run.status === "starting";
  const liveElapsed = useLiveElapsed(run.started_at ?? undefined, isActive);
  const duration = isActive ? liveElapsed : computeElapsed(run);
  const claimant = run.claimed_by?.ref;
  const resultText =
    run.status === "failed"
      ? (run.error ?? "Failed")
      : run.status === "completed"
        ? run.result != null
          ? "Result ready"
          : "Completed"
        : null;

  return (
    <div className="border-t border-line-soft first:border-t-0">
      <Link
        className={cn(
          ROW_GRID,
          "py-2.5 transition-colors duration-fast hover:bg-row-hover",
          "focus-visible:outline-none focus-visible:shadow-focus-ring"
        )}
        data-testid={`tasks-runs-row-${run.id}`}
        params={{ id: taskId, runId: run.id }}
        to="/tasks/$id/runs/$runId"
      >
        <span className="min-w-0">
          <span className="block text-ws-name font-medium text-fg-strong">
            Attempt {run.attempt}
          </span>
          {lineageAttempt !== null ? (
            <span className="mt-0.5 flex items-center gap-1 text-eyebrow text-subtle">
              <CornerUpLeft aria-hidden="true" className="size-[11px] text-faint" />
              retried from attempt {lineageAttempt}
            </span>
          ) : null}
        </span>
        <Pill tone={taskRunStatusTone(run.status)}>
          <Pill.Dot pulse={isActive} tone={taskRunStatusTone(run.status)} />
          {taskRunStatusLabel(run.status)}
        </Pill>
        <span className="inline-flex min-w-0 items-center gap-1.5 text-small-body text-muted">
          {claimant ? (
            <>
              <OwnerAvatar
                name={claimant}
                ownerId={claimant}
                ownerKind={ownerAvatarKindFor(run.claimed_by?.kind)}
                size="sm"
              />
              <span className="truncate">{claimant}</span>
            </>
          ) : (
            <span className="text-subtle">—</span>
          )}
        </span>
        <span className="hidden text-form-label tabular-nums text-muted md:inline">
          {run.started_at ? <Time iso={run.started_at} mode="relative" /> : "—"}
        </span>
        <span className="hidden font-mono text-eyebrow tabular-nums text-muted md:inline">
          {duration ?? "—"}
        </span>
        <span
          className={cn(
            "hidden truncate text-form-label md:inline",
            run.status === "failed" ? "text-danger" : "text-muted"
          )}
        >
          {resultText ?? "—"}
        </span>
        <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
      </Link>
      {reviews.length > 0 ? (
        <div className="flex flex-col gap-0.5 px-4 pb-2.5 pl-[116px]">
          {reviews.map(review => (
            <p
              className={cn(
                "truncate text-form-label",
                review.outcome === "approved"
                  ? "text-success"
                  : review.outcome === "rejected"
                    ? "text-warning"
                    : "text-subtle"
              )}
              data-testid={`tasks-runs-review-${review.review_id}`}
              key={review.review_id}
            >
              {reviewLine(review)}
            </p>
          ))}
        </div>
      ) : null}
    </div>
  );
}

/**
 * Runs tab (§4.5): attempts newest-first, whole row links to the run page,
 * reviews render as a quiet secondary line under their run — never a separate
 * table. Empty state teaches the next useful action.
 */
export function TaskRunsPanel({
  taskId,
  runs,
  reviewsByRun,
  isLoading = false,
  errorMessage = null,
  onStartRun,
  workerName,
}: TaskRunsPanelProps) {
  if (isLoading && runs.length === 0) {
    return (
      <BlockLoading
        label="Loading runs"
        size="md"
        surface="bare"
        data-testid="tasks-runs-loading"
      />
    );
  }

  if (errorMessage && runs.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Couldn't load runs"
        description={errorMessage}
        data-testid="tasks-runs-error"
      />
    );
  }

  if (runs.length === 0) {
    return (
      <Empty
        icon={Play}
        title="Not started yet"
        description={
          workerName
            ? `Start a run to have ${workerName} work on this task.`
            : "Start a run to have a worker pick this task up."
        }
        action={
          onStartRun ? (
            <Button
              data-testid="tasks-runs-start"
              onClick={onStartRun}
              size="sm"
              type="button"
              variant="neutral"
            >
              Start run
            </Button>
          ) : undefined
        }
        data-testid="tasks-runs-empty"
      />
    );
  }

  const byAttempt = [...runs].sort((a, b) => b.attempt - a.attempt);
  const attemptById = new Map(runs.map(run => [run.id, run.attempt]));

  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="tasks-runs-panel"
    >
      <div aria-hidden="true" className={cn(ROW_GRID, "border-b border-line-soft py-2")}>
        <span className="eyebrow text-subtle">Attempt</span>
        <span className="eyebrow text-subtle">Status</span>
        <span className="eyebrow text-subtle">Claimed by</span>
        <span className="eyebrow hidden text-subtle md:inline">Started</span>
        <span className="eyebrow hidden text-subtle md:inline">Duration</span>
        <span className="eyebrow hidden text-subtle md:inline">Result</span>
        <span />
      </div>
      {byAttempt.map(run => (
        <RunRow
          key={run.id}
          lineageAttempt={
            run.previous_run_id ? (attemptById.get(run.previous_run_id) ?? null) : null
          }
          reviews={reviewsByRun?.get(run.id) ?? []}
          run={run}
          taskId={taskId}
        />
      ))}
    </div>
  );
}
