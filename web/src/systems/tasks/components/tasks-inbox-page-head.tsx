import { Briefcase, Clock } from "lucide-react";

import { formatRelativeTime } from "../lib/task-formatters";

export interface TasksInboxPageHeadProps {
  visibleCount: number;
  totalCount: number;
  unreadCount: number;
  archivedCount: number;
  workspaceName?: string | null;
  /** Epoch ms of the last successful inbox fetch (TanStack `dataUpdatedAt`). */
  inboxUpdatedAt?: number;
}

/**
 * Top of the `/tasks?mode=inbox` page — the h1, the visible/total count chip,
 * and the meta line (workspace + sync freshness + unread/archived totals).
 * Pure presentation; data flows from `useTasksPage` via `TasksInboxView`.
 */
export function TasksInboxPageHead({
  visibleCount,
  totalCount,
  unreadCount,
  archivedCount,
  workspaceName,
  inboxUpdatedAt,
}: TasksInboxPageHeadProps) {
  const countLabel =
    visibleCount === totalCount ? `${totalCount}` : `${visibleCount} of ${totalCount}`;
  const syncedLabel = inboxUpdatedAt
    ? formatRelativeTime(new Date(inboxUpdatedAt).toISOString())
    : null;

  return (
    <div className="mb-6 flex flex-col gap-2" data-testid="tasks-inbox-page-head">
      <div className="flex items-center gap-3">
        <h1
          className="m-0 text-detail-h1 font-medium tracking-detail-h1 text-fg-strong"
          data-testid="tasks-inbox-page-title"
        >
          Inbox
        </h1>
        <span
          className="inline-flex min-h-[20px] items-center rounded bg-canvas-soft px-1.5 py-0.5 font-mono text-[11.5px] font-medium tabular-nums text-faint"
          data-testid="tasks-inbox-page-count"
        >
          {countLabel}
        </span>
      </div>
      <div
        className="flex flex-wrap items-center gap-2 text-[12px] text-subtle"
        data-testid="tasks-inbox-page-meta"
      >
        {workspaceName ? (
          <span
            className="inline-flex items-center gap-1.5"
            data-testid="tasks-inbox-page-workspace"
          >
            <Briefcase aria-hidden="true" className="size-3 text-faint" />
            <span>workspace {workspaceName}</span>
          </span>
        ) : null}
        {workspaceName && syncedLabel ? (
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
        ) : null}
        {syncedLabel ? (
          <span className="inline-flex items-center gap-1.5" data-testid="tasks-inbox-page-synced">
            <Clock aria-hidden="true" className="size-3 text-faint" />
            <span>synced {syncedLabel} ago</span>
          </span>
        ) : null}
        {(workspaceName || syncedLabel) && (unreadCount > 0 || archivedCount > 0) ? (
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
        ) : null}
        <span
          className="font-mono text-[11.5px] tabular-nums text-faint"
          data-testid="tasks-inbox-page-totals"
        >
          {unreadCount} unread · {archivedCount} archived
        </span>
      </div>
    </div>
  );
}
