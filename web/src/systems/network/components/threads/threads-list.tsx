import { Hash } from "lucide-react";
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

function ThreadsListRow({ channel, thread, active }: ThreadsListRowProps) {
  const messageCount = thread.message_count ?? 0;
  const peerCount = thread.participant_count ?? 0;
  const lastActivity = formatNetworkRelativeTime(thread.last_activity_at ?? null);

  return (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(
        "group flex flex-col gap-1.5 border-b border-[color:var(--color-divider)] px-5 py-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]",
        active ? "bg-[color:var(--color-accent-tint)]" : "hover:bg-[color:var(--color-hover)]"
      )}
      data-testid={`network-thread-list-row-${thread.thread_id}`}
      params={{ channel, threadId: thread.thread_id }}
      to="/network/$channel/threads/$threadId"
    >
      <div className="flex items-start gap-2">
        <Hash
          aria-hidden="true"
          className="mt-1 size-3.5 text-[color:var(--color-text-tertiary)]"
        />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <p className="truncate text-[14px] font-semibold text-[color:var(--color-text-primary)]">
            {thread.title ?? "Untitled thread"}
          </p>
          <p className="line-clamp-2 text-[13px] text-[color:var(--color-text-secondary)]">
            {thread.last_message_preview ?? "No messages yet."}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-3 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
        <span data-testid="network-thread-list-row-meta-peers">
          {peerCount} {peerCount === 1 ? "peer" : "peers"}
        </span>
        <span aria-hidden="true">·</span>
        <span data-testid="network-thread-list-row-meta-messages">
          {messageCount} {messageCount === 1 ? "msg" : "msgs"}
        </span>
        <span aria-hidden="true">·</span>
        <span data-testid="network-thread-list-row-meta-time">{lastActivity}</span>
      </div>
    </Link>
  );
}

function ThreadsListSkeleton() {
  return (
    <div className="space-y-0" data-testid="network-thread-list-skeleton">
      {[0, 1, 2, 3, 4].map(index => (
        <div
          className="flex flex-col gap-2 border-b border-[color:var(--color-divider)] px-5 py-3"
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
