import { Link, useRouter } from "@tanstack/react-router";
import { ArrowUpRight, LifeBuoy, RotateCcw, Unlock, XCircle } from "lucide-react";

import {
  Button,
  DetailHeader,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  MonoId,
  Pill,
  Textarea,
  Time,
  formatDuration,
} from "@agh/ui";

import { useForceFailDialog } from "../hooks/use-force-fail-dialog";
import { taskRunStatusLabel, taskRunStatusTone, taskStatusSignal } from "../lib/task-formatters";
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
  const router = useRouter();
  const forceFailDialog = useForceFailDialog(onForceFailRun);
  const handleBack = () => {
    router.history.back();
  };

  const record = run.run;
  const task = run.task ?? null;
  const session = run.session;
  const taskID = normalizeText(task?.id ?? record.task_id);
  const identifier = normalizeText(task?.identifier) || taskID || "Network wake";
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

  return (
    <>
      <DetailHeader
        data-testid="task-run-detail-header"
        back={handleBack}
        backLabel={taskID ? "Back to task" : "Back"}
        crumbs={
          <span
            data-testid="task-run-detail-breadcrumb"
            className="inline-flex items-center gap-1.5"
          >
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
            {taskID ? (
              <Link
                data-testid="task-run-detail-breadcrumb-task"
                params={{ id: taskID }}
                to="/tasks/$id"
                className="transition-colors duration-base ease-out hover:text-fg"
              >
                {identifier}
              </Link>
            ) : (
              <span data-testid="task-run-detail-breadcrumb-taskless">{identifier}</span>
            )}
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span>{record.id}</span>
          </span>
        }
        preTitle={taskID ? "Task run" : "Network wake"}
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
            {canForceRelease && onForceReleaseRun ? (
              <Button
                data-testid="task-run-detail-force-release"
                disabled={pendingActions?.has("force-release")}
                onClick={() => void onForceReleaseRun()}
                size="sm"
                type="button"
                variant="neutral"
              >
                <Unlock className="size-3" strokeWidth={1.75} />
                Release run
              </Button>
            ) : null}
            {canForceFail && onForceFailRun ? (
              <Button
                data-testid="task-run-detail-force-fail"
                disabled={pendingActions?.has("force-fail")}
                onClick={forceFailDialog.open}
                size="sm"
                type="button"
                variant="destructive"
              >
                <XCircle className="size-3" strokeWidth={1.75} />
                Fail run
              </Button>
            ) : null}
            {canRetry && onRetryRun ? (
              <Button
                data-testid="task-run-detail-retry"
                disabled={pendingActions?.has("retry")}
                onClick={() => void onRetryRun()}
                size="sm"
                type="button"
                variant="neutral"
              >
                <RotateCcw className="size-3" strokeWidth={1.75} />
                Retry run
              </Button>
            ) : null}
            {canRecover && onRecoverRun ? (
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
            ) : null}
            {canCancel && onCancelRun ? (
              <Button
                data-testid="task-run-detail-cancel"
                disabled={pendingActions?.has("cancel")}
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
