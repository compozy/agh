import { ArrowUpRight, LifeBuoy, RotateCw } from "lucide-react";

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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Pill,
  Textarea,
  TopbarOverflowIcon,
} from "@agh/ui";

import type { useForceFailDialog } from "../hooks/use-force-fail-dialog";
import { taskRunStatusLabel, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskRunDetailView, TaskRunStatus } from "../types";

const ACTIVE_STATUSES: ReadonlySet<TaskRunStatus> = new Set([
  "queued",
  "claimed",
  "starting",
  "running",
]);

export function TaskRunPageStatus({ status }: { status: TaskRunStatus }) {
  const isActive = status === "running" || status === "starting";
  return (
    <Pill data-testid="tasks-run-status" tone={taskRunStatusTone(status)}>
      <Pill.Dot pulse={isActive} tone={taskRunStatusTone(status)} />
      {taskRunStatusLabel(status)}
    </Pill>
  );
}

export interface TaskRunPageActionsProps {
  run: TaskRunDetailView;
  maxAttempts?: number | null;
  isRetryPending?: boolean;
  onOpenSession: (sessionId: string) => void;
  onRetry: () => void;
}

/** Run-detail primary action (§6): Open session while live, Retry when failed. */
export function TaskRunPageActions({
  run,
  maxAttempts,
  isRetryPending = false,
  onOpenSession,
  onRetry,
}: TaskRunPageActionsProps) {
  const record = run.run;
  const sessionId = record.session_id ?? run.session?.session_id ?? null;
  const attemptsRemain = maxAttempts == null || record.attempt < maxAttempts;

  if (record.status === "failed" && attemptsRemain) {
    return (
      <Button
        data-testid="tasks-run-retry"
        disabled={isRetryPending}
        onClick={onRetry}
        size="sm"
        type="button"
      >
        <RotateCw aria-hidden="true" className="size-3" />
        Retry
      </Button>
    );
  }

  if (!sessionId) return null;

  const isTerminal =
    record.status === "completed" || record.status === "failed" || record.status === "canceled";

  return (
    <Button
      data-testid="tasks-run-open-session"
      onClick={() => onOpenSession(sessionId)}
      size="sm"
      type="button"
      variant={isTerminal ? "neutral" : "default"}
    >
      Open session
      <ArrowUpRight aria-hidden="true" className="size-3" />
    </Button>
  );
}

export interface TaskRunPageOverflowProps {
  run: TaskRunDetailView;
  canRecover: boolean;
  pending: {
    cancel?: boolean;
    release?: boolean;
    recover?: boolean;
  };
  onCancel: () => void;
  onRelease: () => void;
  onRecover: () => void;
  onForceFail: () => void;
  onCopyRunId: () => void;
}

/** Operator verbs behind the run head `⋯` (no delete — runs terminalize). */
export function TaskRunPageOverflow({
  run,
  canRecover,
  pending,
  onCancel,
  onRelease,
  onRecover,
  onForceFail,
  onCopyRunId,
}: TaskRunPageOverflowProps) {
  const status = run.run.status;
  const isActive = ACTIVE_STATUSES.has(status);
  const isClaimedPhase = status === "claimed" || status === "starting" || status === "running";
  const showForceFail = isClaimedPhase || status === "needs_attention";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More actions"
        data-testid="tasks-run-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="tasks-run-overflow-menu">
        {canRecover ? (
          <DropdownMenuItem
            data-testid="tasks-run-recover"
            disabled={pending.recover}
            onClick={onRecover}
          >
            <LifeBuoy aria-hidden="true" className="size-3" />
            Recover
          </DropdownMenuItem>
        ) : null}
        {isClaimedPhase ? (
          <DropdownMenuItem
            data-testid="tasks-run-release"
            disabled={pending.release}
            onClick={onRelease}
          >
            Release claim
          </DropdownMenuItem>
        ) : null}
        {isActive ? (
          <DropdownMenuItem
            data-testid="tasks-run-cancel"
            disabled={pending.cancel}
            onClick={onCancel}
          >
            Cancel run
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuItem data-testid="tasks-run-copy-id" onClick={onCopyRunId}>
          Copy run id
        </DropdownMenuItem>
        {showForceFail ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              data-testid="tasks-run-force-fail"
              onClick={onForceFail}
              variant="destructive"
            >
              Force fail…
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export interface TaskRunForceFailDialogProps {
  dialog: ReturnType<typeof useForceFailDialog>;
  isPending?: boolean;
}

/** Force-fail confirmation with a required reason (recorded in the audit log). */
export function TaskRunForceFailDialog({ dialog, isPending = false }: TaskRunForceFailDialogProps) {
  return (
    <Dialog onOpenChange={dialog.handleOpenChange} open={dialog.isOpen}>
      <DialogContent
        className="max-w-md"
        data-testid="tasks-run-force-fail-dialog"
        showCloseButton={!isPending}
      >
        <DialogHeader>
          <DialogTitle>Force fail this run?</DialogTitle>
          <DialogDescription>
            The run is marked failed immediately. The reason is recorded in the audit log.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <label className="eyebrow text-muted" htmlFor={dialog.reasonId}>
            Reason
          </label>
          <Textarea
            aria-describedby={dialog.reasonHintId}
            aria-invalid={Boolean(dialog.error)}
            data-testid="tasks-run-force-fail-reason"
            disabled={isPending}
            id={dialog.reasonId}
            onChange={event => dialog.changeReason(event.target.value)}
            rows={3}
            value={dialog.reason}
          />
          {dialog.error ? (
            <p className="text-form-hint text-danger" id={dialog.reasonHintId} role="alert">
              {dialog.error}
            </p>
          ) : null}
        </div>
        <DialogFooter className="gap-2">
          <Button
            disabled={isPending}
            onClick={() => dialog.handleOpenChange(false)}
            size="sm"
            type="button"
            variant="neutral"
          >
            Cancel
          </Button>
          <Button
            data-testid="tasks-run-force-fail-confirm"
            disabled={isPending}
            onClick={() => void dialog.confirm()}
            size="sm"
            type="button"
            variant="destructive"
          >
            Force fail run
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
