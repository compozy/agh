import { AlertCircle } from "lucide-react";

import { Pill } from "@/components/design-system";
import { cn } from "@/lib/utils";

import {
  formatAttemptLabel,
  formatRelativeTime,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskIsBlocked,
  taskIsDraft,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskListItem } from "../types";

export interface TaskCardProps {
  task: TaskListItem;
  selected?: boolean;
  onSelect?: () => void;
  onPublish?: () => void;
  onRetry?: () => void;
  isPublishPending?: boolean;
}

export function TaskCard({
  task,
  selected = false,
  onSelect,
  onPublish,
  onRetry,
  isPublishPending = false,
}: TaskCardProps) {
  const lastActivity = task.last_activity_at ?? task.updated_at;
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
    <button
      aria-pressed={selected}
      className={cn(
        "relative block w-full border-b border-[color:rgba(58,58,60,0.45)] px-4 py-3 text-left transition-colors",
        "hover:bg-[color:var(--color-surface)]",
        selected && "bg-[color:var(--color-surface)]"
      )}
      data-testid={`task-card-${task.id}`}
      onClick={onSelect}
      type="button"
    >
      {selected ? (
        <span className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]" />
      ) : null}

      <div className="flex items-start gap-2 text-xs text-[color:var(--color-text-tertiary)]">
        {task.identifier ? (
          <span className="font-mono uppercase tracking-[0.12em]">{task.identifier}</span>
        ) : null}
        <span className="ml-auto shrink-0 text-[color:var(--color-text-tertiary)]">
          {formatRelativeTime(lastActivity)}
        </span>
      </div>

      <p className="mt-1 truncate text-sm font-medium text-[color:var(--color-text-primary)]">
        {task.title}
      </p>

      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[color:var(--color-text-secondary)]">
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
          className="mt-1.5 flex items-start gap-1 text-xs text-[color:var(--color-danger)]"
          data-testid={`task-card-error-${task.id}`}
        >
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <span className="truncate">{failedRunError}</span>
        </p>
      ) : null}

      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        <Pill kind="state" tone={taskStatusTone(task.status)}>
          {taskStatusLabel(task.status)}
        </Pill>
        {task.priority ? (
          <Pill kind="state" tone={taskPriorityTone(task.priority)}>
            {taskPriorityLabel(task.priority)}
          </Pill>
        ) : null}
        {showApproval ? (
          <Pill kind="state" tone="amber">
            {taskApprovalStateLabel(task.approval_state)}
          </Pill>
        ) : null}
        {isDraft && onPublish ? (
          <button
            aria-label={`Publish ${task.title}`}
            className="ml-auto rounded-full border border-[color:var(--color-accent)] px-2.5 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-accent)] transition-colors hover:bg-[color:var(--color-accent-tint)] disabled:opacity-50"
            data-testid={`task-card-publish-${task.id}`}
            disabled={isPublishPending}
            onClick={event => {
              event.stopPropagation();
              onPublish();
            }}
            type="button"
          >
            Publish
          </button>
        ) : null}
        {isBlocked ? (
          <span
            className="ml-auto truncate font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-warning)]"
            data-testid={`task-card-blocked-${task.id}`}
          >
            Blocked
          </span>
        ) : null}
        {task.status === "failed" && onRetry ? (
          <button
            aria-label={`Retry ${task.title}`}
            className="ml-auto rounded-full border border-[color:var(--color-accent)] px-2.5 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-accent)] transition-colors hover:bg-[color:var(--color-accent-tint)]"
            data-testid={`task-card-retry-${task.id}`}
            onClick={event => {
              event.stopPropagation();
              onRetry();
            }}
            type="button"
          >
            Retry
          </button>
        ) : null}
      </div>
    </button>
  );
}
