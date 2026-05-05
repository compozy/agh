import { Link } from "@tanstack/react-router";

import { Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkThreadSummary } from "../../types";
import { ThreadsEmpty } from "../empty-states/threads-empty";

export interface ThreadsListProps {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  activeThreadId: string | null;
  isLoading: boolean;
  /** Reduced contrast applied when the right-rail thread overlay is open. */
  dim?: boolean;
  onStartThread?: () => void;
}

interface ThreadsListRowProps {
  channel: string;
  thread: NetworkThreadSummary;
  active: boolean;
}

function StateChip({ openWorkCount }: { openWorkCount: number }) {
  if (openWorkCount === 0) {
    return null;
  }
  // Without per-state breakdown on the summary we can only surface that the
  // thread has open work — clearly truthful, not invented.
  return (
    <span
      className="shrink-0 rounded-[3px] bg-[color:var(--color-warning-tint)] px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-warning)]"
      data-testid="network-thread-list-row-state-chip"
    >
      {openWorkCount === 1 ? "1 work open" : `${openWorkCount} work open`}
    </span>
  );
}

function ThreadsListRow({ channel, thread, active }: ThreadsListRowProps) {
  const messageCount = thread.message_count ?? 0;
  const replyCount = Math.max(0, messageCount - 1); // root + replies — guard against historical zero.
  const peerCount = thread.participant_count ?? 0;
  const openWorkCount = thread.open_work_count ?? 0;
  const lastActivity = formatNetworkRelativeTime(thread.last_activity_at ?? null);
  const opener = thread.opened_by_peer_id?.trim() || "unknown";

  return (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(
        "group flex items-start gap-3 border-b border-[color:var(--color-divider)] px-5 py-4 text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]",
        active ? "bg-[color:var(--color-surface)]" : "hover:bg-[color:var(--color-hover)]"
      )}
      data-testid={`network-thread-list-row-${thread.thread_id}`}
      params={{ channel, threadId: thread.thread_id }}
      to="/network/$channel/threads/$threadId"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex items-start justify-between gap-3">
          <p className="truncate text-[14px] font-semibold text-[color:var(--color-text-primary)]">
            {thread.title ?? "Untitled thread"}
          </p>
          <StateChip openWorkCount={openWorkCount} />
        </div>

        <p className="line-clamp-2 text-[13px] text-[color:var(--color-text-secondary)]">
          {thread.last_message_preview ?? "No messages yet."}
        </p>

        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          <span data-testid="network-thread-list-row-meta-peers">
            {peerCount} {peerCount === 1 ? "peer" : "peers"}
          </span>
          <span aria-hidden="true">·</span>
          <span data-testid="network-thread-list-row-meta-replies">
            {replyCount} {replyCount === 1 ? "reply" : "replies"}
          </span>
          <span aria-hidden="true">·</span>
          <span data-testid="network-thread-list-row-meta-opener">started by {opener}</span>
        </div>
      </div>

      <span
        className="shrink-0 self-start font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        data-testid="network-thread-list-row-meta-time"
      >
        {lastActivity}
      </span>
    </Link>
  );
}

function ThreadsListSkeleton() {
  return (
    <div className="space-y-0" data-testid="network-thread-list-skeleton">
      {[0, 1, 2, 3, 4].map(index => (
        <div
          className="flex flex-col gap-2 border-b border-[color:var(--color-divider)] px-5 py-4"
          key={index}
        >
          <Skeleton className="h-3.5 w-2/3" />
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-3/4" />
        </div>
      ))}
    </div>
  );
}

export function ThreadsList({
  channel,
  threads,
  activeThreadId,
  isLoading,
  dim = false,
  onStartThread,
}: ThreadsListProps) {
  if (isLoading && threads.length === 0) {
    return <ThreadsListSkeleton />;
  }

  if (threads.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <ThreadsEmpty className="max-w-md" onStartThread={onStartThread} />
      </div>
    );
  }

  const total = threads.length;

  return (
    <div
      aria-label={`Threads in #${channel}`}
      aria-live="polite"
      className={cn(
        "flex flex-1 flex-col overflow-y-auto transition-opacity",
        dim ? "opacity-55" : "opacity-100"
      )}
      data-dim={dim ? "true" : "false"}
      data-testid="network-thread-list"
    >
      <div
        className="flex items-center justify-between gap-3 border-b border-[color:var(--color-divider)] px-5 py-2 font-mono text-[10px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        data-testid="network-thread-list-subheader"
      >
        <span>
          {total} {total === 1 ? "thread" : "threads"}
        </span>
        <span aria-hidden="true">Sorted by recent activity</span>
      </div>
      {threads.map(thread => (
        <ThreadsListRow
          active={thread.thread_id === activeThreadId}
          channel={channel}
          key={thread.thread_id}
          thread={thread}
        />
      ))}
    </div>
  );
}
