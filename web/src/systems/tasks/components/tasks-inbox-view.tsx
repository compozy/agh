import { AlertCircle, Loader2, Search } from "lucide-react";

import { Empty, SearchInput, Switch } from "@agh/ui";

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
  onOpen?: TasksInboxItemProps["onOpen"];
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
  onOpen,
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
    onOpen,
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

      <div className="flex flex-wrap items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <SearchInput
          className="h-9 min-w-[220px] flex-1"
          data-testid="tasks-inbox-search"
          onChange={next => onSearchChange(next)}
          placeholder="Search inbox…"
          value={searchQuery}
        />
        <label
          className="inline-flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)]"
          data-testid="tasks-inbox-unread-toggle"
        >
          <Switch checked={unreadOnly} onCheckedChange={onToggleUnread} />
          Unread only
        </label>
        <span
          className="font-mono text-[11px] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]"
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
          <Empty
            icon={AlertCircle}
            title="Unable to load inbox"
            description={errorMessage}
            data-testid="tasks-inbox-error"
          />
        ) : groups.length === 0 ? (
          <Empty
            className="mx-auto max-w-xl"
            description="Approval requests, failed runs, blockers, and archived items will appear here as work progresses."
            icon={Search}
            title="Nothing is waiting in the inbox"
            data-testid="tasks-inbox-empty"
          />
        ) : (
          <div className="flex flex-col gap-6">
            {groups.map(group => (
              <section
                className="flex flex-col gap-2"
                data-testid={`tasks-inbox-group-${group.lane}`}
                key={group.lane}
              >
                <header className="flex items-baseline gap-2 font-mono text-[11px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
                  <span>{taskInboxLaneLabel(group.lane)}</span>
                  <span aria-hidden="true">·</span>
                  <span data-testid={`tasks-inbox-group-count-${group.lane}`}>({group.count})</span>
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
