import { AlertCircle, Archive, ArchiveX, Check, Eye, RotateCcw, X } from "lucide-react";
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Pill } from "@/components/design-system";
import { cn } from "@/lib/utils";

import {
  formatAttemptLabel,
  formatRelativeTime,
  taskApprovalStateLabel,
  taskInboxLaneLabel,
  taskLaneTone,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskInboxItem } from "../types";

export interface TasksInboxItemProps {
  item: TaskInboxItem;
  onApprove?: (taskId: string) => void;
  onReject?: (taskId: string) => void;
  onRetry?: (taskId: string) => void;
  onArchive?: (taskId: string) => void;
  onDismiss?: (taskId: string) => void;
  onMarkRead?: (taskId: string) => void;
  pendingApproveId?: string | null;
  pendingRejectId?: string | null;
  pendingRetryId?: string | null;
  pendingArchiveId?: string | null;
  pendingDismissId?: string | null;
  pendingMarkReadId?: string | null;
}

export function TasksInboxItem({
  item,
  onApprove,
  onReject,
  onRetry,
  onArchive,
  onDismiss,
  onMarkRead,
  pendingApproveId,
  pendingRejectId,
  pendingRetryId,
  pendingArchiveId,
  pendingDismissId,
  pendingMarkReadId,
}: TasksInboxItemProps) {
  const { task, run, triage, lane } = item;
  const taskId = task.id;
  const unread = !triage.read && !triage.dismissed;
  const isApprovalItem = lane === "approvals";
  const isFailedRun = lane === "failed_runs";
  const isArchived = lane === "archived" || triage.archived;
  const ownerLabel = task.owner?.ref ?? "—";
  const failedError = run?.error ?? null;

  return (
    <article
      aria-label={`${taskInboxLaneLabel(lane)} item for ${task.title}`}
      className={cn(
        "flex flex-col gap-2 rounded-2xl border bg-[color:var(--color-surface)] px-4 py-3 transition-colors",
        unread
          ? "border-[color:var(--color-divider)]"
          : "border-[color:var(--color-divider)] opacity-90"
      )}
      data-lane={lane}
      data-testid={`tasks-inbox-item-${taskId}`}
      data-unread={unread ? "true" : "false"}
    >
      <header className="flex flex-wrap items-center gap-2 text-xs">
        {task.identifier ? (
          <span
            className="font-mono uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
            data-testid={`tasks-inbox-item-identifier-${taskId}`}
          >
            {task.identifier}
          </span>
        ) : null}
        {run ? (
          <span className="font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
            run {run.id}
            {typeof run.attempt === "number"
              ? ` · ${formatAttemptLabel(run.attempt, run.max_attempts) ?? ""}`
              : ""}
          </span>
        ) : null}
        <Pill kind="state" tone={taskLaneTone(lane)}>
          {taskInboxLaneLabel(lane)}
        </Pill>
        <Pill kind="state" tone={taskStatusTone(task.status)}>
          {taskStatusLabel(task.status)}
        </Pill>
        {unread ? (
          <span
            className="ml-auto inline-flex size-2 rounded-full bg-[color:var(--color-warning)]"
            data-testid={`tasks-inbox-item-unread-${taskId}`}
          />
        ) : (
          <span
            className="ml-auto font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
            data-testid={`tasks-inbox-item-read-${taskId}`}
          >
            read
          </span>
        )}
      </header>

      <Link
        className="truncate text-sm font-medium text-[color:var(--color-text-primary)] hover:underline"
        data-testid={`tasks-inbox-item-title-${taskId}`}
        params={{ id: taskId }}
        to="/tasks/$id"
      >
        {task.title}
      </Link>

      {item.blocking_reason ? (
        <p
          className="text-xs text-[color:var(--color-text-secondary)]"
          data-testid={`tasks-inbox-item-blocking-${taskId}`}
        >
          {item.blocking_reason}
        </p>
      ) : null}

      {failedError ? (
        <p
          className="flex items-start gap-1 text-xs text-[color:var(--color-danger)]"
          data-testid={`tasks-inbox-item-error-${taskId}`}
        >
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <span className="truncate">{failedError}</span>
        </p>
      ) : null}

      {item.approval_policy === "manual" && item.approval_state ? (
        <p
          className="text-xs text-[color:var(--color-text-secondary)]"
          data-testid={`tasks-inbox-item-approval-${taskId}`}
        >
          Approval state: {taskApprovalStateLabel(item.approval_state)}
        </p>
      ) : null}

      <footer className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="flex items-center gap-2 text-[color:var(--color-text-tertiary)]">
          <span data-testid={`tasks-inbox-item-owner-${taskId}`}>{ownerLabel}</span>
          <span>·</span>
          <span>{formatRelativeTime(item.latest_activity_at)} ago</span>
        </span>

        <div
          className="flex flex-wrap items-center gap-1.5"
          data-testid={`tasks-inbox-item-actions-${taskId}`}
        >
          {isApprovalItem && onReject ? (
            <ActionButton
              icon={<X className="size-3.5" />}
              label="Reject"
              onClick={() => onReject(taskId)}
              pending={pendingRejectId === taskId}
              testId={`tasks-inbox-item-reject-${taskId}`}
              tone="danger"
            />
          ) : null}
          {isApprovalItem && onApprove ? (
            <ActionButton
              icon={<Check className="size-3.5" />}
              label="Approve"
              onClick={() => onApprove(taskId)}
              pending={pendingApproveId === taskId}
              testId={`tasks-inbox-item-approve-${taskId}`}
              tone="accent"
            />
          ) : null}
          {isFailedRun && onDismiss ? (
            <ActionButton
              icon={<ArchiveX className="size-3.5" />}
              label="Dismiss"
              onClick={() => onDismiss(taskId)}
              pending={pendingDismissId === taskId}
              testId={`tasks-inbox-item-dismiss-${taskId}`}
              tone="neutral"
            />
          ) : null}
          {isFailedRun && onRetry ? (
            <ActionButton
              icon={<RotateCcw className="size-3.5" />}
              label="Retry"
              onClick={() => onRetry(taskId)}
              pending={pendingRetryId === taskId}
              testId={`tasks-inbox-item-retry-${taskId}`}
              tone="accent"
            />
          ) : null}
          {!isApprovalItem && !isFailedRun && onMarkRead && unread ? (
            <ActionButton
              icon={<Eye className="size-3.5" />}
              label="Mark read"
              onClick={() => onMarkRead(taskId)}
              pending={pendingMarkReadId === taskId}
              testId={`tasks-inbox-item-mark-read-${taskId}`}
              tone="neutral"
            />
          ) : null}
          {!isArchived && onArchive ? (
            <ActionButton
              icon={<Archive className="size-3.5" />}
              label="Archive"
              onClick={() => onArchive(taskId)}
              pending={pendingArchiveId === taskId}
              testId={`tasks-inbox-item-archive-${taskId}`}
              tone="neutral"
            />
          ) : null}
          <Link
            className="inline-flex items-center rounded-full border border-[color:var(--color-divider)] px-2.5 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
            data-testid={`tasks-inbox-item-open-${taskId}`}
            params={{ id: taskId }}
            to="/tasks/$id"
          >
            Open task
          </Link>
        </div>
      </footer>
    </article>
  );
}

interface ActionButtonProps {
  label: string;
  icon: ReactNode;
  onClick: () => void;
  pending: boolean;
  testId: string;
  tone: "accent" | "danger" | "neutral";
}

function ActionButton({ label, icon, onClick, pending, testId, tone }: ActionButtonProps) {
  const toneClass =
    tone === "accent"
      ? "border-[color:var(--color-accent)] text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)]"
      : tone === "danger"
        ? "border-[color:var(--color-danger)] text-[color:var(--color-danger)] hover:bg-[color:var(--color-danger-tint)]"
        : "border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]";

  return (
    <button
      aria-label={label}
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] transition-colors disabled:opacity-50",
        toneClass
      )}
      data-testid={testId}
      disabled={pending}
      onClick={onClick}
      type="button"
    >
      {icon}
      {label}
    </button>
  );
}
