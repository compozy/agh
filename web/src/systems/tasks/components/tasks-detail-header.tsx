import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Pill } from "@/components/design-system";
import { Button } from "@agh/ui";

import {
  formatRelativeTime,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskIsDraft,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskStatusLabel,
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
  const identifier = record.identifier ?? record.id;
  const isDraft = taskIsDraft(record);
  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";

  return (
    <header
      className="flex flex-col gap-4 border-b border-[color:var(--color-divider)] px-6 py-5"
      data-testid="tasks-detail-header"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-1 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
            data-testid="tasks-detail-breadcrumb"
          >
            <Link
              className="hover:text-[color:var(--color-text-secondary)]"
              data-testid="tasks-detail-breadcrumb-tasks"
              to="/tasks"
            >
              Tasks
            </Link>
            <ChevronRight className="size-3 text-[color:var(--color-text-tertiary)]" />
            <span className="text-[color:var(--color-text-secondary)]">{identifier}</span>
          </nav>

          <h1
            className="mt-2 truncate text-2xl font-semibold text-[color:var(--color-text-primary)]"
            data-testid="tasks-detail-title"
          >
            {record.title}
          </h1>

          <div
            className="mt-3 flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]"
            data-testid="tasks-detail-meta"
          >
            <Pill emphasis="strong" kind="state" tone={taskStatusTone(record.status)}>
              {taskStatusLabel(record.status)}
            </Pill>
            {record.priority ? (
              <Pill kind="state" tone={taskPriorityTone(record.priority)}>
                {taskPriorityLabel(record.priority)}
              </Pill>
            ) : null}
            {taskHasApprovalPending(record) ? (
              <Pill kind="state" tone="amber">
                {taskApprovalStateLabel(record.approval_state)}
              </Pill>
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
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2">
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
      </div>
    </header>
  );
}
