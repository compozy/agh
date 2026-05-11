import type { ReactNode } from "react";

import { Pill } from "@agh/ui";

import {
  formatAttemptLabel,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskIsBlocked,
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
}

/**
 * Full-detail list card -- composes the shared `tasks-list-row` primitive and
 * pushes the rich task metadata into a single inline `__meta` row (proposal
 * `.task-row__meta`). Pills (priority, approval, blocked) sit in the
 * `trailing` column. Publish + retry actions live in the detail header
 * (`tasks-detail-header.tsx`), not on the row.
 */
export function TaskCard({ task, selected = false, onSelect }: TaskCardProps) {
  const isBlocked = taskIsBlocked(task);
  const showApproval = taskHasApprovalPending(task);
  const activeRun = task.active_run ?? null;
  const ownerLabel = taskOwnerLabel(task.owner);
  const childCount = task.child_count ?? 0;
  const dependencyCount = task.dependency_count ?? 0;
  const failedRunError =
    task.status === "failed" && task.active_run?.error ? task.active_run.error : null;

  const metaItems: ReactNode[] = [
    <span data-testid={`task-card-owner-${task.id}`} key="owner">
      {ownerLabel}
    </span>,
  ];
  if (activeRun) {
    metaItems.push(
      <span data-testid={`task-card-attempt-${task.id}`} key="attempt">
        {formatAttemptLabel(activeRun.attempt, activeRun.max_attempts) ?? ""}
      </span>
    );
  }
  if (childCount > 0) {
    metaItems.push(
      <span data-testid={`task-card-children-${task.id}`} key="children">
        {childCount} {childCount === 1 ? "child" : "children"}
      </span>
    );
  }
  if (dependencyCount > 0) {
    metaItems.push(
      <span data-testid={`task-card-deps-${task.id}`} key="deps">
        {dependencyCount} {dependencyCount === 1 ? "dep" : "deps"}
      </span>
    );
  }
  if (task.parent_task_id) {
    metaItems.push(
      <span className="font-mono text-[10.5px] text-(--faint)" key="parent">
        parent {task.parent_task_id}
      </span>
    );
  }
  if (failedRunError) {
    metaItems.push(
      <span
        className="min-w-0 truncate text-(--danger)"
        data-testid={`task-card-error-${task.id}`}
        key="error"
        title={failedRunError}
      >
        {failedRunError}
      </span>
    );
  }

  const trailing = (
    <>
      {task.priority ? (
        <Pill size="sm" tone={taskPriorityTone(task.priority)}>
          {taskPriorityLabel(task.priority)}
        </Pill>
      ) : null}
      {showApproval ? (
        <Pill size="sm" tone="accent">
          {taskApprovalStateLabel(task.approval_state)}
        </Pill>
      ) : null}
      {isBlocked ? (
        <Pill data-testid={`task-card-blocked-${task.id}`} mono size="sm" tone="warning">
          Blocked
        </Pill>
      ) : null}
    </>
  );

  return (
    <TasksListRow
      meta={metaItems}
      onSelect={onSelect ? () => onSelect() : undefined}
      selected={selected}
      task={task}
      trailing={trailing}
    />
  );
}
