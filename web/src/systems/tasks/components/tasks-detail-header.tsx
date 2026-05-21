import { Link, useRouter } from "@tanstack/react-router";
import { PauseCircle, PlayCircle, Radio } from "lucide-react";

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
} from "@agh/ui";

import {
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskApprovalStateLabel,
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
  taskStatusSignal,
  taskStatusTone,
} from "../lib/task-formatters";
import { useTaskPauseDialog } from "../hooks/use-task-pause-dialog";
import type { TaskDetailView } from "../types";
import { TaskDeleteAction } from "./task-delete-action";

export interface TasksDetailHeaderProps {
  detail: TaskDetailView;
  pending?: {
    delete?: boolean;
    publish?: boolean;
    cancel?: boolean;
    enqueue?: boolean;
    pause?: boolean;
    resume?: boolean;
  };
  onDelete?: (taskId: string) => void;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
  onPause?: (reason: string) => void | Promise<void>;
  onResume?: () => void | Promise<void>;
}

/**
 * `/tasks/$id` hero — consumes `<DetailHeader>` so the 6-row anatomy
 * (crumbs → pre-title → 24px H1 → pills → meta → actions) stays in lockstep
 * with the task detail route. The H1 is the only inline-status surface; the
 * pre-title Eyebrow row carries the static "Task" label, pills render under
 * the title, and meta carries owner / origin / updated-at relative time.
 */
export function TasksDetailHeader({
  detail,
  pending,
  onDelete,
  onPublish,
  onCancel,
  onEnqueueRun,
  onPause,
  onResume,
}: TasksDetailHeaderProps) {
  const router = useRouter();
  const pauseDialog = useTaskPauseDialog(onPause);

  const isDeletePending = pending?.delete ?? false;
  const isPublishPending = pending?.publish ?? false;
  const isCancelPending = pending?.cancel ?? false;
  const isEnqueuePending = pending?.enqueue ?? false;
  const isPausePending = pending?.pause ?? false;
  const isResumePending = pending?.resume ?? false;
  const record = detail.task;
  const identifier = taskShortId(record);
  const isDraft = taskIsDraft(record);
  const isDirectlyPaused = Boolean(record.paused);
  const isEffectivelyPaused = Boolean(detail.summary?.effective_paused ?? record.paused);
  const pausedByTaskId = detail.summary?.paused_by_task_id;
  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";
  const signal = taskStatusSignal(record.status);
  const activeRun = detail.summary?.active_run ?? null;
  const hasOpenRun =
    activeRun?.status === "queued" ||
    activeRun?.status === "claimed" ||
    activeRun?.status === "starting" ||
    activeRun?.status === "running";
  const lifecyclePhase = taskLifecyclePhase({
    status: record.status,
    approval_state: record.approval_state,
    draft: record.draft,
    active_run: activeRun,
  });
  const publishCopy = taskHandoffActionCopy("publish");
  const startCopy = taskHandoffActionCopy("start");
  const channelLabel = runIsCoordinated(activeRun) ? runCoordinationChannelLabel(activeRun) : null;
  const startTitle = isEffectivelyPaused
    ? "Resume task dispatch before starting a run."
    : startCopy.tooltip;

  return (
    <DetailHeader
      data-testid="tasks-detail-header"
      back={() => router.history.back()}
      backLabel="Back to tasks"
      crumbs={
        <span data-testid="tasks-detail-breadcrumb" className="inline-flex items-center gap-1.5">
          <Link
            data-testid="tasks-detail-breadcrumb-tasks"
            to="/tasks"
            className="transition-colors duration-base ease-out hover:text-fg"
          >
            Tasks
          </Link>
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
          <span>{identifier}</span>
        </span>
      }
      preTitle="Task"
      title={
        <span data-testid="tasks-detail-title" className="inline-flex min-w-0 items-center gap-2">
          <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
          <span className="truncate">{record.title}</span>
        </span>
      }
      pills={
        <>
          <MonoId data-testid="tasks-detail-id" value={identifier} />
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
        </>
      }
      meta={
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
      }
      actions={
        <div
          data-testid="tasks-detail-actions"
          className="flex shrink-0 flex-wrap items-center gap-2"
        >
          <Link params={{ id: record.id }} to="/tasks/$id/edit">
            <Button data-testid="tasks-detail-edit" size="sm" type="button" variant="neutral">
              Edit
            </Button>
          </Link>
          {canCancel && onCancel ? (
            <Button
              data-testid="tasks-detail-cancel"
              disabled={isCancelPending}
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
              disabled={isResumePending}
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
              isPending={isDeletePending}
              triggerTestId="tasks-detail-delete"
              dialogTestId="tasks-detail-delete-dialog"
              cancelTestId="tasks-detail-delete-cancel"
              confirmTestId="tasks-detail-delete-confirm"
            />
          ) : null}
          {isDraft && onPublish ? (
            <Button
              data-testid="tasks-detail-publish"
              disabled={isPublishPending}
              onClick={onPublish}
              size="sm"
              title={publishCopy.tooltip}
              type="button"
            >
              {publishCopy.label}
            </Button>
          ) : null}
          {!isDraft && !hasOpenRun && onEnqueueRun ? (
            <Button
              data-testid="tasks-detail-enqueue"
              disabled={isEnqueuePending || isEffectivelyPaused}
              onClick={onEnqueueRun}
              size="sm"
              title={startTitle}
              type="button"
            >
              {startCopy.label}
            </Button>
          ) : null}
        </div>
      }
    >
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
    </DetailHeader>
  );
}
