import { Link } from "@tanstack/react-router";
import { ListChecks, Radio } from "lucide-react";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
  Button,
  PageHeader,
  Pill,
} from "@agh/ui";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
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
  isDeletePending?: boolean;
  isPublishPending?: boolean;
  isCancelPending?: boolean;
  isEnqueuePending?: boolean;
  onDelete?: (taskId: string) => void;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
}

/**
 * Detail page header -- `PageHeader` with task title, `Pill.Dot`, short id `Pill`,
 * status pills, and action buttons (edit, cancel, publish, enqueue) in the meta
 * slot. The eyebrow row below surfaces secondary metadata (owner, origin,
 * created-by, last update, priority + approval pills).
 */
export function TasksDetailHeader({
  detail,
  isDeletePending = false,
  isPublishPending = false,
  isCancelPending = false,
  isEnqueuePending = false,
  onDelete,
  onPublish,
  onCancel,
  onEnqueueRun,
}: TasksDetailHeaderProps) {
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
    <header
      className="flex flex-col gap-3 border-b border-(--color-divider)"
      data-testid="tasks-detail-header"
    >
      <PageHeader
        icon={() => <ListChecks className="size-3.5" />}
        title={
          <span className="flex min-w-0 items-center gap-2">
            <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
            <span
              className="truncate text-item-title font-semibold text-(--color-text-primary)"
              data-testid="tasks-detail-title"
            >
              {record.title}
            </span>
            <Pill mono data-testid="tasks-detail-id">
              {identifier}
            </Pill>
            <Pill
              data-testid="tasks-detail-status"
              tone={pillToneFromLegacyTone(taskStatusTone(record.status))}
            >
              {taskStatusLabel(record.status)}
            </Pill>
            <Pill
              data-testid="tasks-detail-lifecycle"
              title={taskLifecyclePhaseDescription(lifecyclePhase)}
              tone={pillToneFromLegacyTone(taskLifecyclePhaseTone(lifecyclePhase))}
            >
              {taskLifecyclePhaseLabel(lifecyclePhase)}
            </Pill>
            {channelLabel ? (
              <Pill
                data-testid="tasks-detail-coordination"
                title="Coordination channel is bound to the active run. Channel messages support coordination only -- task ownership stays in the task service."
                tone={pillToneFromLegacyTone("violet")}
              >
                <span className="inline-flex items-center gap-1">
                  <Radio className="size-3" aria-hidden="true" />
                  Channel: {channelLabel}
                </span>
              </Pill>
            ) : null}
          </span>
        }
        breadcrumb={
          <Breadcrumb data-testid="tasks-detail-breadcrumb">
            <BreadcrumbList className="text-(--color-text-label)">
              <BreadcrumbItem>
                <BreadcrumbLink
                  data-testid="tasks-detail-breadcrumb-tasks"
                  render={<Link to="/tasks" />}
                >
                  Tasks
                </BreadcrumbLink>
              </BreadcrumbItem>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbPage className="text-(--color-text-secondary)">
                  {identifier}
                </BreadcrumbPage>
              </BreadcrumbItem>
            </BreadcrumbList>
          </Breadcrumb>
        }
        meta={
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
        statusRow={
          <div
            className="flex flex-wrap items-center gap-2 text-small-body text-(--color-text-secondary)"
            data-testid="tasks-detail-meta"
          >
            {record.priority ? (
              <Pill tone={pillToneFromLegacyTone(taskPriorityTone(record.priority))}>
                {taskPriorityLabel(record.priority)}
              </Pill>
            ) : null}
            {taskHasApprovalPending(record) ? (
              <Pill tone="accent">{taskApprovalStateLabel(record.approval_state)}</Pill>
            ) : null}
            <span>Owner {taskOwnerLabel(record.owner)}</span>
            <span>Origin {record.origin?.kind?.toUpperCase() ?? "UNKNOWN"}</span>
            <span>
              Created by{" "}
              <span className="text-(--color-text-primary)">
                {record.created_by?.ref ?? "unknown"}
              </span>
            </span>
            <span>Updated {formatRelativeTime(record.updated_at)}</span>
          </div>
        }
        subtitle={
          <span data-testid="tasks-detail-lifecycle-hint">
            {taskLifecyclePhaseDescription(lifecyclePhase)}
          </span>
        }
      />
    </header>
  );
}
