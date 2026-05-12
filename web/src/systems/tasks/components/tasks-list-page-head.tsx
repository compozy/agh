import { Briefcase, Clock } from "lucide-react";

import { formatRelativeTime } from "../lib/task-formatters";

export interface TasksListPageHeadProps {
  visibleCount: number;
  totalCount: number;
  workspaceName?: string | null;
  /** Epoch ms of the last successful task list fetch (TanStack `dataUpdatedAt`). */
  listUpdatedAt?: number;
}

/**
 * Top of the `/tasks` page — the h1, the visible/total count chip, and the
 * meta line (workspace + sync freshness). Pure presentation; data comes from
 * `useTasksPage` via the surface wrapper.
 */
export function TasksListPageHead({
  visibleCount,
  totalCount,
  workspaceName,
  listUpdatedAt,
}: TasksListPageHeadProps) {
  const countLabel =
    visibleCount === totalCount ? `${totalCount}` : `${visibleCount} of ${totalCount}`;
  const syncedLabel = listUpdatedAt
    ? formatRelativeTime(new Date(listUpdatedAt).toISOString())
    : null;

  return (
    <div className="mb-6 flex flex-col gap-2" data-testid="tasks-list-page-head">
      <div className="flex items-center gap-3">
        <h1
          className="m-0 text-detail-h1 font-medium tracking-detail-h1 text-fg-strong"
          data-testid="tasks-list-page-title"
        >
          Tasks
        </h1>
        <span
          className="inline-flex min-h-5 items-center rounded bg-canvas-soft px-1.5 py-0.5 font-mono text-form-hint font-medium tabular-nums text-faint"
          data-testid="tasks-list-page-count"
        >
          {countLabel}
        </span>
      </div>
      <div
        className="flex flex-wrap items-center gap-2 text-form-label text-subtle"
        data-testid="tasks-list-page-meta"
      >
        {workspaceName ? (
          <span
            className="inline-flex items-center gap-1.5"
            data-testid="tasks-list-page-workspace"
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
          <span className="inline-flex items-center gap-1.5" data-testid="tasks-list-page-synced">
            <Clock aria-hidden="true" className="size-3 text-faint" />
            <span>synced {syncedLabel} ago</span>
          </span>
        ) : null}
      </div>
    </div>
  );
}
