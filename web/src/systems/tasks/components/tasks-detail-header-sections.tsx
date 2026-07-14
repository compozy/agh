import { Link } from "@tanstack/react-router";
import {
  AlertTriangle,
  BellOff,
  BellRing,
  LifeBuoy,
  PauseCircle,
  PlayCircle,
  Radio,
} from "lucide-react";

import {
  Button,
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
} from "@agh/ui";

import { useTaskPauseDialog } from "../hooks/use-task-pause-dialog";
import {
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskApprovalStateLabel,
  taskCanRecover,
  taskHandoffActionCopy,
  taskHasApprovalPending,
  taskIsDraft,
  taskLifecyclePhase,
  taskLifecyclePhaseDescription,
  taskLifecyclePhaseLabel,
  taskLifecyclePhaseTone,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskShortId,
  taskStatusLabel,
  taskStatusTone,
  taskWakeIndicatorApplies,
} from "../lib/task-formatters";
import { taskRunCanRecover } from "../lib/task-run-recovery";
import type { TaskDetailView } from "../types";
import { TaskDeleteAction } from "./task-delete-action";

export function TasksDetailHeaderPills({ detail }: { detail: TaskDetailView }) {
  const record = detail.task;
  const activeRun = detail.summary?.active_run ?? null;
  const lifecyclePhase = taskLifecyclePhase({
    status: record.status,
    approval_state: record.approval_state,
    draft: record.draft,
    active_run: activeRun,
  });
  const channelLabel = runIsCoordinated(activeRun) ? runCoordinationChannelLabel(activeRun) : null;
  const isDirectlyPaused = Boolean(record.paused);
  const isEffectivelyPaused = Boolean(detail.summary?.effective_paused ?? record.paused);
  const pausedByTaskId = detail.summary?.paused_by_task_id;

  return (
    <>
      <MonoId data-testid="tasks-detail-id" value={taskShortId(record)} />
      <Pill data-testid="tasks-detail-status" tone={taskStatusTone(record.status)}>
        {taskStatusLabel(record.status)}
      </Pill>
      <Pill
        data-testid="tasks-detail-lifecycle"
        title={taskLifecyclePhaseDescription(lifecyclePhase)}
        tone={lifecyclePhase === "running" ? "info" : taskLifecyclePhaseTone(lifecyclePhase)}
      >
        {taskLifecyclePhaseLabel(lifecyclePhase)}
      </Pill>
      {channelLabel ? (
        <Pill
          data-testid="tasks-detail-coordination"
          title="Coordination channel is bound to the active run. Channel messages support coordination only -- task ownership stays in the task service."
          tone="info"
        >
          <span className="inline-flex items-center gap-1">
            <Radio className="size-3" aria-hidden="true" />
            Channel: {channelLabel}
          </span>
        </Pill>
      ) : null}
      {record.priority ? (
        <Pill data-testid="tasks-detail-priority" tone={taskPriorityTone(record.priority)}>
          {taskPriorityLabel(record.priority)}
        </Pill>
      ) : null}
      {taskHasApprovalPending(record) ? (
        <Pill data-testid="tasks-detail-approval" tone="warning">
          {taskApprovalStateLabel(record.approval_state)}
        </Pill>
      ) : null}
      {isEffectivelyPaused ? (
        <Pill
          data-testid="tasks-detail-pause-state"
          title={
            isDirectlyPaused
              ? record.paused_reason || "Task is paused for future scheduler claims."
              : pausedByTaskId
                ? `Task is paused through ancestor ${pausedByTaskId}.`
                : "Task is paused through an ancestor."
          }
          tone="warning"
        >
          {isDirectlyPaused ? "Paused" : "Paused by ancestor"}
        </Pill>
      ) : null}
      {taskCanRecover(record) ? (
        <Pill
          data-testid="tasks-detail-needs-attention"
          tone="warning"
          title={
            record.needs_attention_reason ||
            "The unblock-loop breaker escalated this task for operator recovery."
          }
        >
          <span className="inline-flex min-w-0 items-center gap-1">
            <AlertTriangle className="size-3 shrink-0" aria-hidden="true" />
            <span className="max-w-[32ch] truncate">
              {record.needs_attention_reason?.trim() || "Escalated for recovery"}
            </span>
          </span>
        </Pill>
      ) : null}
      {taskWakeIndicatorApplies(record) ? (
        <Pill
          data-testid="tasks-detail-wake"
          tone={record.wake_creator ? "info" : "neutral"}
          title={
            record.wake_creator
              ? "The creator session is woken on this task's terminal, blocked, and needs-attention transitions."
              : "Creator wake is opted out for this task (created with --no-wake-creator)."
          }
        >
          <span className="inline-flex items-center gap-1">
            {record.wake_creator ? (
              <BellRing className="size-3" aria-hidden="true" />
            ) : (
              <BellOff className="size-3" aria-hidden="true" />
            )}
            {record.wake_creator ? "Wake on" : "Wake off"}
          </span>
        </Pill>
      ) : null}
    </>
  );
}

export function TasksDetailHeaderMeta({ detail }: { detail: TaskDetailView }) {
  const record = detail.task;
  return (
    <div
      data-testid="tasks-detail-meta"
      className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1"
    >
      <span>Owner {taskOwnerLabel(record.owner)}</span>
      <span aria-hidden="true" className="text-faint">
        ·
      </span>
      <span>Origin {record.origin?.kind?.toUpperCase() ?? "UNKNOWN"}</span>
      <span aria-hidden="true" className="text-faint">
        ·
      </span>
      <span>
        Created by <span className="text-fg">{record.created_by?.ref ?? "unknown"}</span>
      </span>
      <span aria-hidden="true" className="text-faint">
        ·
      </span>
      <span className="inline-flex items-center gap-1">
        Updated <Time iso={record.updated_at} mode="relative" />
      </span>
    </div>
  );
}

interface TasksDetailHeaderActionsProps {
  detail: TaskDetailView;
  pending?: {
    delete?: boolean;
    publish?: boolean;
    cancel?: boolean;
    enqueue?: boolean;
    pause?: boolean;
    resume?: boolean;
    recover?: boolean;
  };
  onDelete?: (taskId: string) => void;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
  onPause?: (reason: string) => void | Promise<void>;
  onResume?: () => void | Promise<void>;
  onRecover?: () => void | Promise<void>;
}

export function TasksDetailHeaderActions({
  detail,
  pending,
  onDelete,
  onPublish,
  onCancel,
  onEnqueueRun,
  onPause,
  onResume,
  onRecover,
}: TasksDetailHeaderActionsProps) {
  const pauseDialog = useTaskPauseDialog(onPause);
  const record = detail.task;
  const isDraft = taskIsDraft(record);
  const isDirectlyPaused = Boolean(record.paused);
  const isEffectivelyPaused = Boolean(detail.summary?.effective_paused ?? record.paused);
  const activeRun = detail.summary?.active_run ?? null;
  const activeRunNeedsAttention = activeRun?.status === "needs_attention";
  const canRecoverRun =
    activeRunNeedsAttention && taskRunCanRecover(activeRun, record.max_attempts);
  const canRecover = activeRunNeedsAttention ? canRecoverRun : taskCanRecover(record);
  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";
  const hasOpenRun =
    activeRun?.status === "queued" ||
    activeRun?.status === "claimed" ||
    activeRun?.status === "starting" ||
    activeRun?.status === "running" ||
    activeRunNeedsAttention;
  const canEnqueueRun = !isDraft && !hasOpenRun && record.status !== "needs_attention";
  const publishCopy = taskHandoffActionCopy("publish");
  const startCopy = taskHandoffActionCopy("start");
  const startTitle = isEffectivelyPaused
    ? "Resume task dispatch before starting a run."
    : startCopy.tooltip;
  const isPausePending = pending?.pause ?? false;

  return (
    <>
      <div
        data-testid="tasks-detail-actions"
        className="flex shrink-0 flex-wrap items-center gap-2"
      >
        <Link params={{ id: record.id }} to="/tasks/$id/edit">
          <Button data-testid="tasks-detail-edit" size="sm" type="button" variant="neutral">
            Edit
          </Button>
        </Link>
        {canRecover && onRecover ? (
          <Button
            data-testid="tasks-detail-recover"
            disabled={pending?.recover}
            onClick={() => void onRecover()}
            size="sm"
            title={
              canRecoverRun
                ? "Terminalize the needs-attention run and queue one continuation."
                : "Clear the needs_attention escalation and return the task to the claimable set."
            }
            type="button"
          >
            <LifeBuoy className="size-3" aria-hidden="true" />
            Recover
          </Button>
        ) : null}
        {canCancel && onCancel ? (
          <Button
            data-testid="tasks-detail-cancel"
            disabled={pending?.cancel}
            onClick={onCancel}
            size="sm"
            type="button"
            variant="neutral"
          >
            Cancel
          </Button>
        ) : null}
        {isDirectlyPaused && onResume ? (
          <Button
            data-testid="tasks-detail-resume"
            disabled={pending?.resume}
            onClick={() => void onResume()}
            size="sm"
            title="Resume scheduler claims for this task."
            type="button"
            variant="neutral"
          >
            <PlayCircle className="size-3" aria-hidden="true" />
            Resume
          </Button>
        ) : !isDirectlyPaused && onPause ? (
          <Button
            data-testid="tasks-detail-pause"
            disabled={isPausePending}
            onClick={pauseDialog.open}
            size="sm"
            title="Pause future scheduler claims for this task."
            type="button"
            variant="neutral"
          >
            <PauseCircle className="size-3" aria-hidden="true" />
            Pause
          </Button>
        ) : null}
        {onDelete ? (
          <TaskDeleteAction
            taskId={record.id}
            taskTitle={record.title}
            onDelete={onDelete}
            isPending={pending?.delete ?? false}
            triggerTestId="tasks-detail-delete"
            dialogTestId="tasks-detail-delete-dialog"
            cancelTestId="tasks-detail-delete-cancel"
            confirmTestId="tasks-detail-delete-confirm"
          />
        ) : null}
        {isDraft && onPublish ? (
          <Button
            data-testid="tasks-detail-publish"
            disabled={pending?.publish}
            onClick={onPublish}
            size="sm"
            title={publishCopy.tooltip}
            type="button"
          >
            {publishCopy.label}
          </Button>
        ) : null}
        {canEnqueueRun && onEnqueueRun ? (
          <Button
            data-testid="tasks-detail-enqueue"
            disabled={pending?.enqueue || isEffectivelyPaused}
            onClick={onEnqueueRun}
            size="sm"
            title={startTitle}
            type="button"
          >
            {startCopy.label}
          </Button>
        ) : null}
      </div>
      <Dialog open={pauseDialog.isOpen} onOpenChange={pauseDialog.onOpenChange}>
        <DialogContent
          data-testid="tasks-detail-pause-dialog"
          showCloseButton={!isPausePending}
          className="max-w-md"
        >
          <DialogHeader>
            <DialogTitle>Pause task?</DialogTitle>
            <DialogDescription>
              New scheduler claims stop for this task; active runs continue.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <label className="eyebrow text-muted" htmlFor="tasks-detail-pause-reason">
              Reason
            </label>
            <Textarea
              aria-invalid={Boolean(pauseDialog.error)}
              data-testid="tasks-detail-pause-reason"
              disabled={isPausePending}
              id="tasks-detail-pause-reason"
              onChange={pauseDialog.onReasonChange}
              rows={3}
              value={pauseDialog.reason}
            />
            {pauseDialog.error ? (
              <p className="text-form-hint text-danger" data-testid="tasks-detail-pause-error">
                {pauseDialog.error}
              </p>
            ) : null}
          </div>
          <DialogFooter className="gap-2">
            <Button
              disabled={isPausePending}
              onClick={pauseDialog.close}
              size="sm"
              type="button"
              variant="neutral"
            >
              Cancel
            </Button>
            <Button
              data-testid="tasks-detail-pause-confirm"
              disabled={isPausePending}
              onClick={() => void pauseDialog.confirm()}
              size="sm"
              type="button"
            >
              Pause
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
