import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowUpRight, LifeBuoy, RotateCcw, Unlock, XCircle } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  MonoId,
  Pill,
  Textarea,
  Time,
  TopbarOverflowIcon,
  formatDuration,
  useTopbarSlot,
} from "@agh/ui";

import { describeCost } from "@/lib/cost-provenance";

import { useForceFailDialog } from "../hooks/use-force-fail-dialog";
import { taskRunStatusLabel, taskStatusSignal } from "../lib/task-formatters";
import { taskRunCanRecover } from "../lib/task-run-recovery";
import type { TaskRunDetailView } from "../types";

export interface TaskRunDetailHeaderProps {
  run: TaskRunDetailView;
  maxAttempts?: number | null;
  onCancelRun?: () => void;
  onForceReleaseRun?: (reason?: string) => Promise<void> | void;
  onForceFailRun?: (reason: string) => Promise<void> | void;
  onRecoverRun?: () => Promise<void> | void;
  onRetryRun?: () => Promise<void> | void;
  pendingActions?: ReadonlySet<"cancel" | "force-release" | "force-fail" | "recover" | "retry">;
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

/** Cost-provenance entry for the run meta row; renders nothing when the summary carries no cost. */
function TaskRunCostMeta({ summary }: { summary: TaskRunDetailView["summary"] }) {
  const cost = describeCost({
    status: summary?.cost_status,
    source: summary?.cost_source,
    amount: summary?.total_cost,
    currency: summary?.cost_currency,
  });
  if (!cost.hasCost) return null;
  return (
    <>
      <span aria-hidden="true" className="text-faint">
        ·
      </span>
      <span
        data-testid="task-run-detail-cost"
        data-cost-status={cost.status}
        className="inline-flex items-center gap-1.5"
      >
        <span>{cost.isEstimated ? "Est. cost" : "Cost"}</span>
        <span className="text-fg">{cost.value}</span>
        {cost.sourceLabel ? (
          <>
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span>{cost.sourceLabel}</span>
          </>
        ) : null}
      </span>
    </>
  );
}

export function TaskRunDetailHeader({
  run,
  maxAttempts,
  onCancelRun,
  onForceReleaseRun,
  onForceFailRun,
  onRecoverRun,
  onRetryRun,
  pendingActions,
}: TaskRunDetailHeaderProps) {
  const navigate = useNavigate();
  const forceFailDialog = useForceFailDialog(onForceFailRun);

  const record = run.run;
  const task = run.task ?? null;
  const taskID = normalizeText(task?.id ?? record.task_id);
  const session = run.session;
  const canCancel =
    record.status === "queued" ||
    record.status === "claimed" ||
    record.status === "starting" ||
    record.status === "running";
  const canForceRelease = record.status === "claimed";
  const canForceFail = record.status === "queued" || record.status === "claimed";
  const canRecover = Boolean(task) && taskRunCanRecover(record, maxAttempts);
  const canRetry = Boolean(task) && record.status === "failed";
  const signal = taskStatusSignal(record.status);
  const elapsedLabel = computeElapsedLabel(record.started_at, record.ended_at);
  const linkedSessionID = normalizeText(session?.session_id ?? record.session_id);
  const linkedSessionAgent = normalizeText(session?.agent_name);
  const claimedRef = normalizeText(record.claimed_by?.ref);

  const openSessionMenuItem =
    linkedSessionID && linkedSessionAgent ? (
      <DropdownMenuItem
        data-testid="task-run-detail-open-session"
        render={
          <Link
            params={{ name: linkedSessionAgent, id: linkedSessionID }}
            to="/agents/$name/sessions/$id"
          />
        }
      >
        Open session
        <ArrowUpRight className="size-3" strokeWidth={1.75} />
      </DropdownMenuItem>
    ) : linkedSessionID ? (
      <DropdownMenuItem
        data-testid="task-run-detail-open-session"
        render={<Link params={{ id: linkedSessionID }} to="/session/$id" />}
      >
        Open session
        <ArrowUpRight className="size-3" strokeWidth={1.75} />
      </DropdownMenuItem>
    ) : null;

  const accentPrimary =
    canRecover && onRecoverRun ? (
      <Button
        data-testid="task-run-detail-recover"
        disabled={pendingActions?.has("recover")}
        onClick={() => void onRecoverRun()}
        size="sm"
        title="Mark this needs-attention run as failed and queue one continuation."
        type="button"
      >
        <LifeBuoy className="size-3" strokeWidth={1.75} />
        Recover
      </Button>
    ) : canRetry && onRetryRun ? (
      <Button
        data-testid="task-run-detail-retry"
        disabled={pendingActions?.has("retry")}
        onClick={() => void onRetryRun()}
        size="sm"
        type="button"
      >
        <RotateCcw className="size-3" strokeWidth={1.75} />
        Retry run
      </Button>
    ) : null;

  const visibleAction =
    accentPrimary ??
    (canCancel && onCancelRun ? (
      <Button
        data-testid="task-run-detail-cancel"
        disabled={pendingActions?.has("cancel")}
        onClick={onCancelRun}
        size="sm"
        type="button"
        variant="neutral"
      >
        Cancel run
      </Button>
    ) : null);

  const hasOverflowItems =
    Boolean(openSessionMenuItem) ||
    (canForceRelease && Boolean(onForceReleaseRun)) ||
    (canForceFail && Boolean(onForceFailRun));

  const parentTaskId = taskID || record.task_id;
  const backToTasks = () => {
    void navigate({ to: "/tasks" });
  };
  const backToTask = () => {
    if (!parentTaskId) {
      backToTasks();
      return;
    }
    void navigate({ to: "/tasks/$id", params: { id: parentTaskId } });
  };

  useTopbarSlot({
    onBack: parentTaskId ? backToTask : backToTasks,
    crumbs: parentTaskId
      ? [
          { id: "tasks", label: "Tasks", onSelect: backToTasks },
          {
            id: "task",
            label: <span data-testid="task-run-detail-context">{task?.title ?? parentTaskId}</span>,
            onSelect: backToTask,
          },
        ]
      : [{ id: "tasks", label: "Tasks", onSelect: backToTasks }],
    crumb: <span data-testid="task-run-detail-title">Run {record.id}</span>,
    status: (
      <Pill data-testid="task-run-detail-status" tone={signal.tone}>
        <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
        {taskRunStatusLabel(record.status)}
      </Pill>
    ),
    actions: visibleAction ? (
      <div
        data-testid="task-run-detail-actions"
        className="flex shrink-0 flex-wrap items-center gap-2"
      >
        {visibleAction}
      </div>
    ) : undefined,
    overflow: hasOverflowItems ? (
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="More actions"
          data-testid="task-run-detail-overflow"
          render={<Button type="button" variant="ghost" size="icon-sm" />}
        >
          <TopbarOverflowIcon aria-hidden="true" className="size-3" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" data-testid="task-run-detail-overflow-menu">
          {openSessionMenuItem}
          {canForceRelease && onForceReleaseRun ? (
            <DropdownMenuItem
              data-testid="task-run-detail-force-release"
              disabled={pendingActions?.has("force-release")}
              onClick={() => void onForceReleaseRun()}
            >
              <Unlock className="size-3" strokeWidth={1.75} />
              Release run
            </DropdownMenuItem>
          ) : null}
          {canForceFail && onForceFailRun ? (
            <DropdownMenuItem
              data-testid="task-run-detail-force-fail"
              disabled={pendingActions?.has("force-fail")}
              onClick={forceFailDialog.open}
              variant="destructive"
            >
              <XCircle className="size-3" strokeWidth={1.75} />
              Fail run
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    ) : undefined,
  });

  return (
    <>
      <div className="flex flex-col gap-2 pt-4" data-testid="task-run-detail-header">
        <div className="flex flex-wrap items-center gap-1.5">
          <MonoId data-testid="task-run-detail-run-id" value={record.id} />
          {elapsedLabel ? (
            <Pill data-testid="task-run-detail-duration" tone="neutral">
              {elapsedLabel}
            </Pill>
          ) : null}
        </div>
        <div
          data-testid="task-run-detail-meta"
          className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-small-body text-subtle"
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
          <TaskRunCostMeta summary={run.summary} />
        </div>
      </div>
      <Dialog open={forceFailDialog.isOpen} onOpenChange={forceFailDialog.handleOpenChange}>
        <DialogContent
          data-testid="task-run-detail-force-fail-dialog"
          className="w-(--width-modal-sm)"
        >
          <DialogHeader>
            <DialogTitle>Fail run?</DialogTitle>
            <DialogDescription>
              This records an operator-forced failure and clears any active claim.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <label
              className="text-small-body font-medium text-fg"
              htmlFor={forceFailDialog.reasonId}
            >
              Reason
            </label>
            <Textarea
              aria-describedby={forceFailDialog.reasonHintId}
              aria-invalid={Boolean(forceFailDialog.error)}
              data-testid="task-run-detail-force-fail-reason"
              id={forceFailDialog.reasonId}
              onChange={event => {
                forceFailDialog.changeReason(event.target.value);
              }}
              required
              rows={4}
              value={forceFailDialog.reason}
            />
            <p className="text-small-body text-muted" id={forceFailDialog.reasonHintId}>
              Required for the audit event.
            </p>
            {forceFailDialog.error ? (
              <p className="text-small-body text-danger" role="alert">
                {forceFailDialog.error}
              </p>
            ) : null}
          </div>
          <DialogFooter className="gap-2">
            <Button
              disabled={pendingActions?.has("force-fail")}
              onClick={() => forceFailDialog.handleOpenChange(false)}
              type="button"
              variant="neutral"
            >
              Cancel
            </Button>
            <Button
              data-testid="task-run-detail-force-fail-confirm"
              disabled={pendingActions?.has("force-fail")}
              onClick={() => void forceFailDialog.confirm()}
              type="button"
              variant="destructive"
            >
              Fail run
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
