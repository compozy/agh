import { Link } from "@tanstack/react-router";

import {
  Eyebrow,
  Item,
  ItemContent,
  ItemFooter,
  ItemHeader,
  ItemTitle,
  Pill,
  Skeleton,
  SkeletonRows,
} from "@agh/ui";

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

function ThreadWorkPill({ openWorkCount }: { openWorkCount: number }) {
  if (openWorkCount === 0) {
    return null;
  }
  // Without per-state breakdown on the summary we can only surface that the
  // thread has open work; clearly truthful, not invented.
  return (
    <Pill data-testid="network-thread-list-row-state-chip" mono size="xs" tone="warning">
      {openWorkCount === 1 ? "1 work open" : `${openWorkCount} work open`}
    </Pill>
  );
}

function ThreadsListRow({ channel, thread, active }: ThreadsListRowProps) {
  const messageCount = thread.message_count ?? 0;
  const replyCount = Math.max(0, messageCount - 1); // root + replies - guard against historical zero.
  const peerCount = thread.participant_count ?? 0;
  const openWorkCount = thread.open_work_count ?? 0;
  const lastActivity = formatNetworkRelativeTime(thread.last_activity_at ?? null);
  const opener = thread.opened_by_peer_id?.trim() || "unknown";

  return (
    <Item
      aria-current={active ? "page" : undefined}
      className={cn(
        "rounded-none border-b border-(--line) px-5 py-4",
        active ? "bg-(--canvas-soft)" : null
      )}
      data-testid={`network-thread-list-row-${thread.thread_id}`}
      indicator={active ? "rail" : "none"}
      render={
        <Link
          params={{ channel, threadId: thread.thread_id }}
          to="/network/$channel/threads/$threadId"
        />
      }
      selectable
      selected={active}
    >
      <ItemContent className="gap-1.5">
        <ItemHeader>
          <ItemTitle className="min-w-0 flex-1">
            <span className="truncate">{thread.title ?? "Untitled thread"}</span>
          </ItemTitle>
          <ThreadWorkPill openWorkCount={openWorkCount} />
        </ItemHeader>

        <p className="line-clamp-2 text-small-body text-(--muted)">
          {thread.last_message_preview ?? "No messages yet."}
        </p>

        <ItemFooter className="items-start">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <Eyebrow data-testid="network-thread-list-row-meta-peers" weight="medium">
              {peerCount} {peerCount === 1 ? "peer" : "peers"}
            </Eyebrow>
            <Eyebrow aria-hidden="true" weight="medium">
              /
            </Eyebrow>
            <Eyebrow data-testid="network-thread-list-row-meta-replies" weight="medium">
              {replyCount} {replyCount === 1 ? "reply" : "replies"}
            </Eyebrow>
            <Eyebrow aria-hidden="true" weight="medium">
              /
            </Eyebrow>
            <Eyebrow data-testid="network-thread-list-row-meta-opener" weight="medium">
              started by {opener}
            </Eyebrow>
          </div>
          <Eyebrow
            className="shrink-0"
            data-testid="network-thread-list-row-meta-time"
            weight="medium"
          >
            {lastActivity}
          </Eyebrow>
        </ItemFooter>
      </ItemContent>
    </Item>
  );
}

function ThreadsListSkeleton() {
  return (
    <SkeletonRows
      count={5}
      data-testid="network-thread-list-skeleton"
      rowClassName="border-b border-(--line) px-5 py-4"
    >
      <Skeleton className="h-3.5 w-2/3" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="size-3/4" />
    </SkeletonRows>
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
        className="flex items-center justify-between gap-3 border-b border-(--line) px-5 py-2"
        data-testid="network-thread-list-subheader"
      >
        <Eyebrow>
          {total} {total === 1 ? "thread" : "threads"}
        </Eyebrow>
        <Eyebrow aria-hidden="true">Sorted by recent activity</Eyebrow>
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
