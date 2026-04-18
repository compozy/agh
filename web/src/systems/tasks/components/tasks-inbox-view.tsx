import { AlertCircle, Loader2, Search } from "lucide-react";

import { Empty, Input } from "@agh/ui";

import type { InboxLaneFilter } from "@/hooks/routes/use-tasks-page";

import { taskInboxLaneLabel } from "../lib/task-formatters";
import type { TaskInboxView } from "../types";
import { TasksInboxItem, type TasksInboxItemProps } from "./tasks-inbox-item";
import { TasksInboxLaneTabs } from "./tasks-inbox-lane-tabs";

export interface TasksInboxViewProps {
  inbox: TaskInboxView | null;
  laneFilter: InboxLaneFilter;
  onLaneChange: (lane: InboxLaneFilter) => void;
  unreadOnly: boolean;
  onToggleUnread: (next: boolean) => void;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  onApprove?: TasksInboxItemProps["onApprove"];
  onReject?: TasksInboxItemProps["onReject"];
  onRetry?: TasksInboxItemProps["onRetry"];
  onArchive?: TasksInboxItemProps["onArchive"];
  onDismiss?: TasksInboxItemProps["onDismiss"];
  onMarkRead?: TasksInboxItemProps["onMarkRead"];
  pendingApproveId?: string | null;
  pendingRejectId?: string | null;
  pendingRetryId?: string | null;
  pendingArchiveId?: string | null;
  pendingDismissId?: string | null;
  pendingMarkReadId?: string | null;
}

export function TasksInboxView({
  inbox,
  laneFilter,
  onLaneChange,
  unreadOnly,
  onToggleUnread,
  searchQuery,
  onSearchChange,
  isLoading = false,
  errorMessage = null,
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
}: TasksInboxViewProps) {
  const groups = inbox?.groups ?? [];
  const itemActionProps: Omit<TasksInboxItemProps, "item"> = {
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
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-inbox-view">
      <TasksInboxLaneTabs inbox={inbox} onChange={onLaneChange} value={laneFilter} />

      <div className="flex flex-wrap items-center gap-2 border-b border-[color:var(--color-divider)] px-4 py-3">
        <div className="relative flex-1 min-w-[220px]">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-[color:var(--color-text-tertiary)]" />
          <Input
            className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] pl-9"
            data-testid="tasks-inbox-search"
            onChange={event => onSearchChange(event.target.value)}
            placeholder="Search inbox..."
            value={searchQuery}
          />
        </div>
        <label
          className="inline-flex items-center gap-2 font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-secondary)]"
          data-testid="tasks-inbox-unread-toggle"
        >
          <input
            checked={unreadOnly}
            className="size-3.5 rounded border-[color:var(--color-divider)] bg-transparent"
            onChange={event => onToggleUnread(event.target.checked)}
            type="checkbox"
          />
          Unread only
        </label>
        <span
          className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]"
          data-testid="tasks-inbox-totals"
        >
          {inbox?.unread_total ?? 0} unread · {inbox?.archived_total ?? 0} archived
        </span>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4">
        {isLoading && !inbox ? (
          <div
            className="flex min-h-full items-center justify-center py-10"
            data-testid="tasks-inbox-loading"
          >
            <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
          </div>
        ) : errorMessage && !inbox ? (
          <div
            className="flex min-h-full items-center justify-center py-10"
            data-testid="tasks-inbox-error"
          >
            <div className="flex max-w-md flex-col items-center gap-2 text-center">
              <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
              <p className="text-sm text-[color:var(--color-text-secondary)]">{errorMessage}</p>
            </div>
          </div>
        ) : groups.length === 0 ? (
          <div
            className="flex min-h-full items-center justify-center py-10"
            data-testid="tasks-inbox-empty"
          >
            <Empty
              className="max-w-xl"
              icon={Search}
              title="Nothing is waiting in the inbox"
              description="Approval requests, failed runs, blockers, and archived items will appear here as work progresses."
            />
          </div>
        ) : (
          <div className="flex flex-col gap-6">
            {groups.map(group => (
              <section
                className="flex flex-col gap-2"
                data-testid={`tasks-inbox-group-${group.lane}`}
                key={group.lane}
              >
                <header className="flex items-center gap-2 font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                  <span>{taskInboxLaneLabel(group.lane)}</span>
                  <span
                    className="text-[color:var(--color-text-tertiary)]"
                    data-testid={`tasks-inbox-group-count-${group.lane}`}
                  >
                    {group.count} {group.count === 1 ? "item" : "items"}
                  </span>
                  {group.unread_count > 0 ? (
                    <span
                      className="rounded-full bg-[color:var(--color-warning)] px-1.5 py-[1px] text-[0.58rem] tracking-[0.12em] text-[color:var(--color-accent-ink)]"
                      data-testid={`tasks-inbox-group-unread-${group.lane}`}
                    >
                      {group.unread_count} unread
                    </span>
                  ) : null}
                </header>
                <div className="flex flex-col gap-2">
                  {(group.items ?? []).map(item => (
                    <TasksInboxItem {...itemActionProps} item={item} key={item.task.id} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
