import { Link } from "@tanstack/react-router";
import { ListChecks } from "lucide-react";

import { Button, MonoBadge, PageHeader, Pill, StatusDot } from "@agh/ui";
import { pillVariantFromTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskIsDraft,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskShortId,
  taskStatusLabel,
  taskStatusSignal,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

export interface TasksDetailHeaderProps {
  detail: TaskDetailView;
  isPublishPending?: boolean;
  isCancelPending?: boolean;
  isEnqueuePending?: boolean;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
}

/**
 * Detail page header — `PageHeader` with the task title + short id `MonoBadge` +
 * status pill, plus action buttons (edit, cancel, publish, enqueue) in the meta
 * slot. The eyebrow row below surfaces secondary metadata (owner, origin, created-
 * by, last update, priority + approval pills).
 */
export function TasksDetailHeader({
  detail,
  isPublishPending = false,
  isCancelPending = false,
  isEnqueuePending = false,
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

  return (
    <header
      className="flex flex-col gap-3 border-b border-[color:var(--color-divider)]"
      data-testid="tasks-detail-header"
    >
      <PageHeader
        icon={() => <ListChecks className="size-3.5" />}
        title={
          <span className="flex min-w-0 items-center gap-2">
            <StatusDot tone={signal.tone} pulse={signal.pulse} />
            <span
              className="truncate text-[15px] font-semibold text-[color:var(--color-text-primary)]"
              data-testid="tasks-detail-title"
            >
              {record.title}
            </span>
            <MonoBadge data-testid="tasks-detail-id">{identifier}</MonoBadge>
            <Pill
              data-testid="tasks-detail-status"
              variant={pillVariantFromTone(taskStatusTone(record.status))}
            >
              {taskStatusLabel(record.status)}
            </Pill>
          </span>
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
                type="button"
              >
                Publish
              </Button>
            ) : null}
            {!isDraft && onEnqueueRun ? (
              <Button
                data-testid="tasks-detail-enqueue"
                disabled={isEnqueuePending}
                onClick={onEnqueueRun}
                size="sm"
                type="button"
              >
                Enqueue Run
              </Button>
            ) : null}
          </div>
        }
      />

      <nav
        aria-label="Breadcrumb"
        className="flex items-center gap-1.5 px-4 pb-2 font-mono text-[11px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
        data-testid="tasks-detail-breadcrumb"
      >
        <Link
          className="hover:text-[color:var(--color-text-secondary)]"
          data-testid="tasks-detail-breadcrumb-tasks"
          to="/tasks"
        >
          Tasks
        </Link>
        <span aria-hidden="true">›</span>
        <span className="text-[color:var(--color-text-secondary)]">{identifier}</span>
      </nav>

      <div
        className="flex flex-wrap items-center gap-2 px-4 pb-3 text-[13px] text-[color:var(--color-text-secondary)]"
        data-testid="tasks-detail-meta"
      >
        {record.priority ? (
          <Pill variant={pillVariantFromTone(taskPriorityTone(record.priority))}>
            {taskPriorityLabel(record.priority)}
          </Pill>
        ) : null}
        {taskHasApprovalPending(record) ? (
          <Pill variant="accent">{taskApprovalStateLabel(record.approval_state)}</Pill>
        ) : null}
        <span>Owner {taskOwnerLabel(record.owner)}</span>
        <span>· Origin {record.origin?.kind?.toUpperCase() ?? "UNKNOWN"}</span>
        <span>
          · Created by{" "}
          <span className="text-[color:var(--color-text-primary)]">
            {record.created_by?.ref ?? "unknown"}
          </span>
        </span>
        <span>· Updated {formatRelativeTime(record.updated_at)}</span>
      </div>
    </header>
  );
}
