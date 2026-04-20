import { AlertCircle } from "lucide-react";

import { MonoBadge, Pill } from "@agh/ui";
import { pillVariantFromTone } from "@/lib/pill-variant";

import {
  formatAttemptLabel,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskIsBlocked,
  taskIsDraft,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
} from "../lib/task-formatters";
import type { TaskListItem } from "../types";
import { TasksListRow } from "./tasks-list-row";

export interface TaskCardProps {
  task: TaskListItem;
  selected?: boolean;
  onSelect?: () => void;
  onPublish?: () => void;
  onRetry?: () => void;
  isPublishPending?: boolean;
}

/**
 * Full-detail list card — composes the shared `tasks-list-row` primitive and
 * layers the rich task metadata (attempts, children, deps, priority, publish/retry
 * actions, failure summary) that the Tasks list column surfaces. Kanban + Inbox
 * (task 18) will consume `TasksListRow` directly with their own slot content.
 */
export function TaskCard({
  task,
  selected = false,
  onSelect,
  onPublish,
  onRetry,
  isPublishPending = false,
}: TaskCardProps) {
  const isDraft = taskIsDraft(task);
  const isBlocked = taskIsBlocked(task);
  const showApproval = taskHasApprovalPending(task);
  const activeRun = task.active_run ?? null;
  const ownerLabel = taskOwnerLabel(task.owner);
  const childCount = task.child_count ?? 0;
  const dependencyCount = task.dependency_count ?? 0;
  const failedRunError =
    task.status === "failed" && task.active_run?.error ? task.active_run.error : null;

  return (
    <TasksListRow
      task={task}
      selected={selected}
      onSelect={onSelect ? () => onSelect() : undefined}
      footer={
        <>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-[color:var(--color-text-secondary)]">
            <span data-testid={`task-card-owner-${task.id}`}>{ownerLabel}</span>
            {activeRun ? (
              <span data-testid={`task-card-attempt-${task.id}`}>
                {formatAttemptLabel(activeRun.attempt, activeRun.max_attempts) ?? ""}
              </span>
            ) : null}
            {childCount > 0 ? (
              <span data-testid={`task-card-children-${task.id}`}>
                {childCount} {childCount === 1 ? "child" : "children"}
              </span>
            ) : null}
            {dependencyCount > 0 ? (
              <span data-testid={`task-card-deps-${task.id}`}>
                {dependencyCount} {dependencyCount === 1 ? "dep" : "deps"}
              </span>
            ) : null}
            {task.parent_task_id ? (
              <span className="font-mono text-[color:var(--color-text-tertiary)]">
                parent {task.parent_task_id}
              </span>
            ) : null}
          </div>

          {failedRunError ? (
            <p
              className="flex items-start gap-1 text-[11px] text-[color:var(--color-danger)]"
              data-testid={`task-card-error-${task.id}`}
            >
              <AlertCircle className="mt-0.5 size-3 shrink-0" />
              <span className="truncate">{failedRunError}</span>
            </p>
          ) : null}

          <div className="flex flex-wrap items-center gap-1.5">
            {task.priority ? (
              <Pill variant={pillVariantFromTone(taskPriorityTone(task.priority))}>
                {taskPriorityLabel(task.priority)}
              </Pill>
            ) : null}
            {showApproval ? (
              <Pill variant="accent">{taskApprovalStateLabel(task.approval_state)}</Pill>
            ) : null}
            {isDraft && onPublish ? (
              <button
                type="button"
                aria-label={`Publish ${task.title}`}
                disabled={isPublishPending}
                data-testid={`task-card-publish-${task.id}`}
                onClick={event => {
                  event.stopPropagation();
                  onPublish();
                }}
                className="ml-auto rounded-lg border border-[color:var(--color-divider)] px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] transition-colors hover:border-[color:var(--color-text-label)] hover:text-[color:var(--color-text-primary)] disabled:opacity-50"
              >
                Publish
              </button>
            ) : null}
            {isBlocked ? (
              <MonoBadge
                tone="warning"
                data-testid={`task-card-blocked-${task.id}`}
                className="ml-auto"
              >
                Blocked
              </MonoBadge>
            ) : null}
            {task.status === "failed" && onRetry ? (
              <button
                type="button"
                aria-label={`Retry ${task.title}`}
                data-testid={`task-card-retry-${task.id}`}
                onClick={event => {
                  event.stopPropagation();
                  onRetry();
                }}
                className="ml-auto rounded-lg border border-[color:var(--color-divider)] px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] transition-colors hover:border-[color:var(--color-text-label)] hover:text-[color:var(--color-text-primary)]"
              >
                Retry
              </button>
            ) : null}
          </div>
        </>
      }
    />
  );
}
