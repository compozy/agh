import { Link } from "@tanstack/react-router";
import { ArrowUpRight, LifeBuoy, RotateCw } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Pill,
  TopbarOverflowIcon,
} from "@agh/ui";

import { taskStatusLabel, taskStatusSignal } from "../lib/task-formatters";
import type { TaskCommandState } from "../lib/task-command-state";
import type { TaskStatus } from "../types";

/** Single source of task status in the window head (w2-status contract). */
export function TaskPageStatus({ status }: { status?: TaskStatus | null }) {
  const signal = taskStatusSignal(status);
  return (
    <Pill data-testid="tasks-detail-status" tone={signal.tone}>
      <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
      {taskStatusLabel(status)}
    </Pill>
  );
}

export interface TaskPageActionHandlers {
  onPublish: () => void;
  onApprove: () => void;
  onStartRun: () => void;
  onOpenRun: (runId: string) => void;
  onResume: () => void;
  onRecover: () => void;
  onRetry: (runId: string) => void;
  onEdit: () => void;
}

export interface TaskPageActionsProps {
  command: TaskCommandState;
  handlers: TaskPageActionHandlers;
  pending?: boolean;
}

const PRIMARY_LABEL: Record<NonNullable<TaskCommandState["primary"]>["kind"], string> = {
  publish: "Publish",
  approve: "Approve",
  start: "Start run",
  open_run: "Open run",
  resume: "Resume",
  recover: "Recover",
  retry: "Retry",
};

/** The one accent target in the head, driven by the §6 state machine. */
export function TaskPageActions({ command, handlers, pending = false }: TaskPageActionsProps) {
  const primary = command.primary;

  if (!primary) {
    if (!command.showEditButton) return null;
    return (
      <Button
        data-testid="tasks-detail-edit-button"
        onClick={handlers.onEdit}
        size="sm"
        type="button"
        variant="neutral"
      >
        Edit
      </Button>
    );
  }

  const onClick = () => {
    switch (primary.kind) {
      case "publish":
        return handlers.onPublish();
      case "approve":
        return handlers.onApprove();
      case "start":
        return handlers.onStartRun();
      case "open_run":
        return handlers.onOpenRun(primary.runId);
      case "resume":
        return handlers.onResume();
      case "recover":
        return handlers.onRecover();
      case "retry":
        return handlers.onRetry(primary.runId);
    }
  };

  return (
    <Button
      data-testid={`tasks-detail-primary-${primary.kind}`}
      disabled={pending}
      onClick={onClick}
      size="sm"
      type="button"
    >
      {primary.kind === "recover" ? <LifeBuoy aria-hidden="true" className="size-3" /> : null}
      {primary.kind === "retry" ? <RotateCw aria-hidden="true" className="size-3" /> : null}
      {PRIMARY_LABEL[primary.kind]}
      {primary.kind === "open_run" ? <ArrowUpRight aria-hidden="true" className="size-3" /> : null}
    </Button>
  );
}

export interface TaskPageOverflowProps {
  taskId: string;
  command: TaskCommandState;
  pending: {
    cancel?: boolean;
    pause?: boolean;
    resume?: boolean;
    enqueue?: boolean;
  };
  showFanOut: boolean;
  onPause: () => void;
  onResume: () => void;
  onCancel: () => void;
  onStartNewRun: () => void;
  onFanOut: () => void;
  onCopyId: () => void;
  onDelete: () => void;
}

/** Low-frequency verbs behind the head `⋯` trigger (destructive last). */
export function TaskPageOverflow({
  taskId,
  command,
  pending,
  showFanOut,
  onPause,
  onResume,
  onCancel,
  onStartNewRun,
  onFanOut,
  onCopyId,
  onDelete,
}: TaskPageOverflowProps) {
  const { overflow } = command;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More actions"
        data-testid="tasks-detail-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="tasks-detail-overflow-menu">
        {overflow.edit ? (
          <DropdownMenuItem
            data-testid="tasks-detail-edit"
            render={<Link params={{ id: taskId }} to="/tasks/$id/edit" />}
          >
            Edit
          </DropdownMenuItem>
        ) : null}
        {overflow.pause ? (
          <DropdownMenuItem
            data-testid="tasks-detail-pause"
            disabled={pending.pause}
            onClick={onPause}
          >
            Pause
          </DropdownMenuItem>
        ) : null}
        {overflow.resume ? (
          <DropdownMenuItem
            data-testid="tasks-detail-resume"
            disabled={pending.resume}
            onClick={onResume}
          >
            Resume
          </DropdownMenuItem>
        ) : null}
        {overflow.cancel ? (
          <DropdownMenuItem
            data-testid="tasks-detail-cancel"
            disabled={pending.cancel}
            onClick={onCancel}
          >
            Cancel task
          </DropdownMenuItem>
        ) : null}
        {overflow.startNewRun ? (
          <DropdownMenuItem
            data-testid="tasks-detail-start-new-run"
            disabled={pending.enqueue}
            onClick={onStartNewRun}
          >
            Start new run
          </DropdownMenuItem>
        ) : null}
        {showFanOut ? (
          <DropdownMenuItem data-testid="tasks-detail-fan-out" onClick={onFanOut}>
            Fan out runs…
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuItem data-testid="tasks-detail-copy-id" onClick={onCopyId}>
          Copy task id
        </DropdownMenuItem>
        {overflow.delete ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              data-testid="tasks-detail-delete"
              onClick={onDelete}
              variant="destructive"
            >
              Delete task…
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
