import { Link } from "@tanstack/react-router";
import { AlertCircle, Loader2 } from "lucide-react";

import {
  Panel,
  PanelBody,
  PanelDescription,
  PanelHeader,
  PanelTitle,
  Pill,
} from "@/components/design-system";
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
      className="flex min-h-0 flex-1 flex-col overflow-y-auto bg-[color:var(--color-canvas)]"
      data-testid="tasks-detail-preview-panel"
    >
      <div className="border-b border-[color:var(--color-divider)] px-6 py-5">
        <div className="flex items-start gap-3">
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
              className="mt-3 text-[1.7rem] font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)]"
              data-testid="tasks-detail-preview-title"
            >
              {record.title}
            </h2>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-[color:var(--color-text-secondary)]">
              Owner {taskOwnerLabel(record.owner)} · Scope {record.scope} · Updated{" "}
              {formatRelativeTime(record.updated_at)}
            </p>
          </div>
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <Link
              data-testid="tasks-detail-preview-edit-link"
              params={{ id: record.id }}
              to="/tasks/$id/edit"
            >
              <Button size="sm" type="button" variant="outline">
                Edit
              </Button>
            </Link>
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
                Enqueue run
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
          </div>
        </div>
      </div>

      <div className="grid gap-4 px-6 py-5 md:grid-cols-3">
        <MetricPanel
          label="Children"
          testId="tasks-detail-preview-counts-children"
          value={childCount}
        />
        <MetricPanel
          label="Dependencies"
          testId="tasks-detail-preview-counts-deps"
          value={dependencyCount}
        />
        <MetricPanel label="Runs" testId="tasks-detail-preview-counts-runs" value={runs.length} />
      </div>

      <div className="px-6 pb-6">
        <Panel data-testid="tasks-detail-preview-overview">
          <PanelHeader>
            <PanelTitle>Overview</PanelTitle>
            <PanelDescription>
              Open the full detail view for timeline, descendant work, and run history.
            </PanelDescription>
          </PanelHeader>
          <PanelBody className="gap-4">
            {detail?.task.description ? (
              <p className="whitespace-pre-wrap text-sm leading-6 text-[color:var(--color-text-secondary)]">
                {detail.task.description}
              </p>
            ) : (
              <p className="text-sm italic text-[color:var(--color-text-tertiary)]">
                No description provided yet. Open the full detail view to inspect timeline, runs,
                and dependencies.
              </p>
            )}

            <div className="flex items-center justify-between gap-3 border-t border-[color:var(--color-divider)] pt-4">
              <div className="text-xs text-[color:var(--color-text-secondary)]">
                Created by {record.created_by?.ref ?? "unknown"} · origin{" "}
                {record.origin?.kind ?? "unknown"}
              </div>
              <Link
                className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] underline-offset-2 hover:underline"
                data-testid="tasks-detail-preview-deeplink"
                params={{ id: record.id }}
                to="/tasks/$id"
              >
                Open detail
              </Link>
            </div>
          </PanelBody>
        </Panel>
      </div>
    </section>
  );
}

function MetricPanel({ label, testId, value }: { label: string; testId: string; value: number }) {
  return (
    <Panel className="gap-3" data-testid={testId}>
      <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <p className="text-2xl font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)]">
        {value}
      </p>
    </Panel>
  );
}
