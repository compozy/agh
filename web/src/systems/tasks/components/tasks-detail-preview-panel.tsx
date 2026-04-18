import { Link } from "@tanstack/react-router";
import { AlertCircle, Loader2 } from "lucide-react";

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
import type { TaskDetailView, TaskListItem } from "../types";

export interface TasksDetailPreviewPanelProps {
  task: TaskListItem | null;
  detail: TaskDetailView | null;
  isLoading?: boolean;
  errorMessage?: string | null;
  onCancelTask?: (taskId: string) => void;
  onEnqueueRun?: (taskId: string) => void;
  onPublishTask?: (taskId: string) => void;
  isPublishPending?: boolean;
}

export function TasksDetailPreviewPanel({
  task,
  detail,
  isLoading = false,
  errorMessage = null,
  onCancelTask,
  onEnqueueRun,
  onPublishTask,
  isPublishPending = false,
}: TasksDetailPreviewPanelProps) {
  if (!task) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-6 py-10 text-sm text-[color:var(--color-text-tertiary)]"
        data-testid="tasks-detail-preview-empty"
      >
        Select a task to inspect its overview.
      </div>
    );
  }

  if (isLoading && !detail) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-detail-preview-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage && !detail) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-detail-preview-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">{errorMessage}</p>
        </div>
      </div>
    );
  }

  const record = detail?.task ?? task;
  const childCount = detail?.children?.length ?? task.child_count ?? 0;
  const dependencyReferences = detail?.dependency_references ?? detail?.dependencies ?? [];
  const dependencyCount = dependencyReferences.length || (task.dependency_count ?? 0);
  const runs = detail?.runs ?? [];
  const isDraft = taskIsDraft(record);

  return (
    <section
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid="tasks-detail-preview-panel"
    >
      <div className="flex items-start gap-3 border-b border-[color:var(--color-divider)] px-6 py-4">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-tertiary)]">
            {record.identifier ? (
              <span className="font-mono uppercase tracking-[0.12em]">{record.identifier}</span>
            ) : null}
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
          </div>
          <h2
            className="mt-2 truncate text-xl font-semibold text-[color:var(--color-text-primary)]"
            data-testid="tasks-detail-preview-title"
          >
            {record.title}
          </h2>
          <p className="mt-1 text-xs text-[color:var(--color-text-secondary)]">
            Owner: {taskOwnerLabel(record.owner)} · Scope: {record.scope} · Updated{" "}
            {formatRelativeTime(record.updated_at)}
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {isDraft && onPublishTask ? (
            <Button
              data-testid="tasks-detail-preview-publish"
              disabled={isPublishPending}
              onClick={() => onPublishTask(record.id)}
              size="sm"
              type="button"
            >
              Publish
            </Button>
          ) : null}
          {!isDraft && onEnqueueRun ? (
            <Button
              data-testid="tasks-detail-preview-enqueue"
              onClick={() => onEnqueueRun(record.id)}
              size="sm"
              type="button"
              variant="outline"
            >
              Enqueue Run
            </Button>
          ) : null}
          {onCancelTask &&
          (record.status === "ready" ||
            record.status === "in_progress" ||
            record.status === "blocked") ? (
            <Button
              data-testid="tasks-detail-preview-cancel"
              onClick={() => onCancelTask(record.id)}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
          ) : null}
          <Link
            className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] underline-offset-2 hover:underline"
            data-testid="tasks-detail-preview-deeplink"
            params={{ id: record.id }}
            to="/tasks/$id"
          >
            Open detail
          </Link>
        </div>
      </div>

      <div className="grid gap-4 border-b border-[color:var(--color-divider)] px-6 py-4 md:grid-cols-3">
        <div data-testid="tasks-detail-preview-counts-children">
          <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            Children
          </p>
          <p className="mt-1 text-lg font-semibold text-[color:var(--color-text-primary)]">
            {childCount}
          </p>
        </div>
        <div data-testid="tasks-detail-preview-counts-deps">
          <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            Dependencies
          </p>
          <p className="mt-1 text-lg font-semibold text-[color:var(--color-text-primary)]">
            {dependencyCount}
          </p>
        </div>
        <div data-testid="tasks-detail-preview-counts-runs">
          <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            Runs
          </p>
          <p className="mt-1 text-lg font-semibold text-[color:var(--color-text-primary)]">
            {runs.length}
          </p>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4 text-sm text-[color:var(--color-text-secondary)]">
        {detail?.task.description ? (
          <p className="whitespace-pre-wrap leading-relaxed">{detail.task.description}</p>
        ) : (
          <p className="italic text-[color:var(--color-text-tertiary)]">
            No description provided. Use the deep-link to view timeline, runs, and dependencies.
          </p>
        )}
      </div>
    </section>
  );
}
