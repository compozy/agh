import { useCallback } from "react";
import { Link, useRouter } from "@tanstack/react-router";
import { Radio } from "lucide-react";

import { Button, DetailHeader, MonoId, Pill, Time } from "@agh/ui";

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
import type { TaskDetailView } from "../types";
import { TaskDeleteAction } from "./task-delete-action";

export interface TasksDetailHeaderProps {
  detail: TaskDetailView;
  pending?: {
    delete?: boolean;
    publish?: boolean;
    cancel?: boolean;
    enqueue?: boolean;
  };
  onDelete?: (taskId: string) => void;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
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
}: TasksDetailHeaderProps) {
  const router = useRouter();
  const handleBack = useCallback(() => {
    router.history.back();
  }, [router]);

  const isDeletePending = pending?.delete ?? false;
  const isPublishPending = pending?.publish ?? false;
  const isCancelPending = pending?.cancel ?? false;
  const isEnqueuePending = pending?.enqueue ?? false;
  const record = detail.task;
  const identifier = taskShortId(record);
  const isDraft = taskIsDraft(record);
  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";
  const signal = taskStatusSignal(record.status);
  const activeRun = detail.summary?.active_run ?? null;
  const lifecyclePhase = taskLifecyclePhase({
    status: record.status,
    approval_state: record.approval_state,
    draft: record.draft,
    active_run: activeRun,
  });
  const publishCopy = taskHandoffActionCopy("publish");
  const startCopy = taskHandoffActionCopy("start");
  const channelLabel = runIsCoordinated(activeRun) ? runCoordinationChannelLabel(activeRun) : null;

  return (
    <DetailHeader
      data-testid="tasks-detail-header"
      back={handleBack}
      backLabel="Back to tasks"
      crumbs={
        <span data-testid="tasks-detail-breadcrumb" className="inline-flex items-center gap-1.5">
          <Link
            data-testid="tasks-detail-breadcrumb-tasks"
            to="/tasks"
            className="transition-colors duration-(--dur) ease-(--ease) hover:text-(--fg)"
          >
            Tasks
          </Link>
          <span aria-hidden="true" className="text-(--faint)">
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
            tone={taskLifecyclePhaseTone(lifecyclePhase)}
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
            <Pill data-testid="tasks-detail-approval" tone="accent">
              {taskApprovalStateLabel(record.approval_state)}
            </Pill>
          ) : null}
        </>
      }
      meta={
        <div
          data-testid="tasks-detail-meta"
          className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1"
        >
          <span data-testid="tasks-detail-lifecycle-hint">
            {taskLifecyclePhaseDescription(lifecyclePhase)}
          </span>
          <span aria-hidden="true" className="text-(--faint)">
            ·
          </span>
          <span>Owner {taskOwnerLabel(record.owner)}</span>
          <span aria-hidden="true" className="text-(--faint)">
            ·
          </span>
          <span>Origin {record.origin?.kind?.toUpperCase() ?? "UNKNOWN"}</span>
          <span aria-hidden="true" className="text-(--faint)">
            ·
          </span>
          <span>
            Created by <span className="text-(--fg)">{record.created_by?.ref ?? "unknown"}</span>
          </span>
          <span aria-hidden="true" className="text-(--faint)">
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
            <Button data-testid="tasks-detail-edit" size="sm" type="button" variant="outline">
              Edit
            </Button>
          </Link>
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
          {canCancel && onCancel ? (
            <Button
              data-testid="tasks-detail-cancel"
              disabled={isCancelPending}
              onClick={onCancel}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
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
          {!isDraft && onEnqueueRun ? (
            <Button
              data-testid="tasks-detail-enqueue"
              disabled={isEnqueuePending}
              onClick={onEnqueueRun}
              size="sm"
              title={startCopy.tooltip}
              type="button"
            >
              {startCopy.label}
            </Button>
          ) : null}
        </div>
      }
    />
  );
}
