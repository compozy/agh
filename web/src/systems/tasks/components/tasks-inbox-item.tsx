import { AlertCircle, Archive, ArchiveX, Check, Eye, RotateCcw, X } from "lucide-react";
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Button, Pill, StatusDot } from "@agh/ui";

import { pillVariantFromTone } from "@/lib/pill-variant";
import {
  formatAttemptLabel,
  formatRelativeTime,
  taskApprovalStateLabel,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskInboxItem, TaskListItem } from "../types";
import { TasksListRow } from "./tasks-list-row";

export interface TasksInboxItemProps {
  item: TaskInboxItem;
  onApprove?: (taskId: string) => void;
  onReject?: (taskId: string) => void;
  onRetry?: (taskId: string) => void;
  onArchive?: (taskId: string) => void;
  onDismiss?: (taskId: string) => void;
  onMarkRead?: (taskId: string) => void;
  onOpen?: (taskId: string) => void;
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
  onOpen,
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
  const failedError = run?.error ?? null;

  const handleSelect = () => onOpen?.(taskId);

  const trailing = (
    <>
      {unread ? (
        <StatusDot data-testid={`tasks-inbox-item-unread-${taskId}`} tone="accent" />
      ) : null}
      <Pill size="sm" variant={pillVariantFromTone(taskStatusTone(task.status))}>
        {taskStatusLabel(task.status)}
      </Pill>
      {run ? (
        <span
          className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
          data-testid={`tasks-inbox-item-run-${taskId}`}
        >
          run {run.id}
          {typeof run.attempt === "number"
            ? ` · ${formatAttemptLabel(run.attempt, run.max_attempts) ?? ""}`
            : ""}
        </span>
      ) : null}
    </>
  );

  const footer = (
    <InboxItemFooter
      approval_policy={item.approval_policy}
      approval_state={item.approval_state}
      blocking_reason={item.blocking_reason}
      failedError={failedError}
      isApprovalItem={isApprovalItem}
      isArchived={isArchived}
      isFailedRun={isFailedRun}
      item={item}
      onApprove={onApprove}
      onArchive={onArchive}
      onDismiss={onDismiss}
      onMarkRead={onMarkRead}
      onReject={onReject}
      onRetry={onRetry}
      pendingApproveId={pendingApproveId}
      pendingArchiveId={pendingArchiveId}
      pendingDismissId={pendingDismissId}
      pendingMarkReadId={pendingMarkReadId}
      pendingRejectId={pendingRejectId}
      pendingRetryId={pendingRetryId}
      taskId={taskId}
      unread={unread}
    />
  );

  return (
    <TasksListRow
      className="rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-lane={lane}
      data-unread={unread ? "true" : "false"}
      footer={footer}
      lane={lane}
      onSelect={onOpen ? handleSelect : undefined}
      task={task as unknown as TaskListItem}
      testId={`tasks-inbox-item-${taskId}`}
      trailing={trailing}
    />
  );
}

interface InboxItemFooterProps {
  item: TaskInboxItem;
  taskId: string;
  unread: boolean;
  isApprovalItem: boolean;
  isFailedRun: boolean;
  isArchived: boolean;
  failedError: string | null;
  approval_policy?: TaskInboxItem["approval_policy"];
  approval_state?: TaskInboxItem["approval_state"];
  blocking_reason?: TaskInboxItem["blocking_reason"];
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

function InboxItemFooter({
  item,
  taskId,
  unread,
  isApprovalItem,
  isFailedRun,
  isArchived,
  failedError,
  approval_policy,
  approval_state,
  blocking_reason,
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
}: InboxItemFooterProps) {
  const ownerLabel = item.task.owner?.ref ?? "—";

  return (
    <>
      {blocking_reason ? (
        <p
          className="text-xs text-[color:var(--color-text-secondary)]"
          data-testid={`tasks-inbox-item-blocking-${taskId}`}
        >
          {blocking_reason}
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

      {approval_policy === "manual" && approval_state ? (
        <p
          className="text-xs text-[color:var(--color-text-secondary)]"
          data-testid={`tasks-inbox-item-approval-${taskId}`}
        >
          Approval state: {taskApprovalStateLabel(approval_state)}
        </p>
      ) : null}

      <div
        className="mt-1 flex flex-wrap items-center justify-between gap-2 text-[11px]"
        data-testid={`tasks-inbox-item-actions-${taskId}`}
      >
        <span className="flex items-center gap-2 text-[color:var(--color-text-tertiary)]">
          <span data-testid={`tasks-inbox-item-owner-${taskId}`}>{ownerLabel}</span>
          <span>·</span>
          <span>{formatRelativeTime(item.latest_activity_at)} ago</span>
        </span>

        <div
          className="flex flex-wrap items-center gap-1.5"
          onClick={stopPropagation}
          onKeyDown={stopPropagation}
          role="presentation"
        >
          {isApprovalItem && onReject ? (
            <ActionButton
              icon={<X />}
              label="Reject"
              onClick={() => onReject(taskId)}
              pending={pendingRejectId === taskId}
              testId={`tasks-inbox-item-reject-${taskId}`}
              variant="danger"
            />
          ) : null}
          {isApprovalItem && onApprove ? (
            <ActionButton
              icon={<Check />}
              label="Approve"
              onClick={() => onApprove(taskId)}
              pending={pendingApproveId === taskId}
              testId={`tasks-inbox-item-approve-${taskId}`}
              variant="primary"
            />
          ) : null}
          {isFailedRun && onDismiss ? (
            <ActionButton
              icon={<ArchiveX />}
              label="Dismiss"
              onClick={() => onDismiss(taskId)}
              pending={pendingDismissId === taskId}
              testId={`tasks-inbox-item-dismiss-${taskId}`}
              variant="ghost"
            />
          ) : null}
          {isFailedRun && onRetry ? (
            <ActionButton
              icon={<RotateCcw />}
              label="Retry"
              onClick={() => onRetry(taskId)}
              pending={pendingRetryId === taskId}
              testId={`tasks-inbox-item-retry-${taskId}`}
              variant="primary"
            />
          ) : null}
          {!isApprovalItem && !isFailedRun && onMarkRead && unread ? (
            <ActionButton
              icon={<Eye />}
              label="Mark read"
              onClick={() => onMarkRead(taskId)}
              pending={pendingMarkReadId === taskId}
              testId={`tasks-inbox-item-mark-read-${taskId}`}
              variant="ghost"
            />
          ) : null}
          {!isArchived && onArchive ? (
            <ActionButton
              icon={<Archive />}
              label="Archive"
              onClick={() => onArchive(taskId)}
              pending={pendingArchiveId === taskId}
              testId={`tasks-inbox-item-archive-${taskId}`}
              variant="ghost"
            />
          ) : null}
          <Button
            data-testid={`tasks-inbox-item-open-${taskId}`}
            nativeButton={false}
            render={<Link params={{ id: taskId }} to="/tasks/$id" />}
            size="xs"
            variant="outline"
          >
            Open
          </Button>
        </div>
      </div>
    </>
  );
}

interface ActionButtonProps {
  label: string;
  icon: ReactNode;
  onClick: () => void;
  pending: boolean;
  testId: string;
  variant: "primary" | "ghost" | "danger";
}

function ActionButton({ label, icon, onClick, pending, testId, variant }: ActionButtonProps) {
  const buttonVariant =
    variant === "primary" ? "default" : variant === "danger" ? "destructive" : "ghost";
  return (
    <Button
      aria-label={label}
      data-testid={testId}
      data-variant={variant}
      disabled={pending}
      onClick={onClick}
      size="xs"
      type="button"
      variant={buttonVariant}
    >
      {icon}
      {label}
    </Button>
  );
}

function stopPropagation(event: { stopPropagation: () => void }) {
  event.stopPropagation();
}
