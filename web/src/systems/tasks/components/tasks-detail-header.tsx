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
 * Detail page header rendered inside the task body. After P4 the route's
 * shell topbar shows the static title; this header renders the dynamic task
 * identity (title, status pills, breadcrumb, actions, status row).
 */
export function TasksDetailHeader({
  detail,
  pending,
  onDelete,
  onPublish,
  onCancel,
  onEnqueueRun,
}: TasksDetailHeaderProps) {
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
    <header
      data-slot="page-header"
      className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-4 py-2.5"
      data-testid="tasks-detail-header"
    >
      <div
        data-slot="page-header-breadcrumb"
        className="min-w-0 font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
      >
        <Breadcrumb data-testid="tasks-detail-breadcrumb">
          <BreadcrumbList className="text-(--muted)">
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
              <BreadcrumbPage className="text-(--muted)">{identifier}</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div
        data-slot="page-header-main"
        className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
      >
        <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
          <span
            aria-hidden="true"
            data-slot="page-header-icon"
            className="inline-flex size-6 shrink-0 items-center justify-center rounded-(--radius-sm) bg-(--elevated) text-(--accent)"
          >
            <ListChecks className="size-3.5" />
          </span>
          <h1 className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)">
            <span className="flex min-w-0 items-center gap-2">
              <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
              <span
                className="truncate text-item-title font-semibold text-(--fg)"
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
          </h1>
        </div>
        <div
          data-slot="page-header-meta"
          data-testid="tasks-detail-actions"
          className="ml-auto flex shrink-0 flex-wrap items-center gap-2 text-[13px] text-(--muted)"
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
      </div>
      <div
        data-slot="page-header-subtitle"
        className="max-w-152 text-small-body text-(--muted)"
      >
        <span data-testid="tasks-detail-lifecycle-hint">
          {taskLifecyclePhaseDescription(lifecyclePhase)}
        </span>
      </div>
      <div
        data-slot="page-header-status-row"
        className="flex flex-wrap items-center gap-x-4 gap-y-2 text-small-body text-(--muted)"
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
          Created by <span className="text-(--fg)">{record.created_by?.ref ?? "unknown"}</span>
        </span>
        <span>Updated {formatRelativeTime(record.updated_at)}</span>
      </div>
    </header>
  );
}
