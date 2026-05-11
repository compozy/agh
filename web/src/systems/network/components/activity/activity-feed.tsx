import { ActivitySquare } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Empty, Eyebrow, Pill, Skeleton, SkeletonRows } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary, NetworkThreadSummary } from "../../types";

export interface ActivityFeedProps {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  isLoading: boolean;
}

type ThreadEntry = {
  kind: "thread";
  id: string;
  preview: string;
  title: string;
  timestamp: string | null;
  to: "/network/$channel/threads/$threadId";
  params: { channel: string; threadId: string };
};

type DirectEntry = {
  kind: "direct";
  id: string;
  preview: string;
  title: string;
  timestamp: string | null;
  to: "/network/$channel/directs/$directId";
  params: { channel: string; directId: string };
};

type ActivityEntry = ThreadEntry | DirectEntry;

function buildEntries(
  channel: string,
  threads: ReadonlyArray<NetworkThreadSummary>,
  directs: ReadonlyArray<NetworkDirectRoomSummary>
): ActivityEntry[] {
  const entries: ActivityEntry[] = [];
  for (const thread of threads) {
    entries.push({
      id: `thread:${thread.thread_id}`,
      kind: "thread",
      params: { channel, threadId: thread.thread_id },
      preview: thread.last_message_preview ?? "No messages yet.",
      timestamp: thread.last_activity_at ?? null,
      title: thread.title ?? "Untitled thread",
      to: "/network/$channel/threads/$threadId",
    });
  }
  for (const direct of directs) {
    entries.push({
      id: `direct:${direct.direct_id}`,
      kind: "direct",
      params: { channel, directId: direct.direct_id },
      preview: direct.last_message_preview ?? "No messages yet.",
      timestamp: direct.last_activity_at ?? null,
      title: `${direct.peer_a} ↔ ${direct.peer_b}`,
      to: "/network/$channel/directs/$directId",
    });
  }

  return entries.sort((left, right) => {
    const leftTs = left.timestamp ? new Date(left.timestamp).getTime() : 0;
    const rightTs = right.timestamp ? new Date(right.timestamp).getTime() : 0;
    return rightTs - leftTs;
  });
}

function ActivityFeedSkeleton() {
  return (
    <SkeletonRows
      count={4}
      data-testid="network-activity-feed-skeleton"
      rowClassName="border-b border-(--line) px-5 py-3"
    >
      <Skeleton className="h-3 w-24" />
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-3 w-full" />
    </SkeletonRows>
  );
}

export function ActivityFeed({ channel, threads, directs, isLoading }: ActivityFeedProps) {
  const entries = buildEntries(channel, threads, directs);

  if (isLoading && entries.length === 0) {
    return <ActivityFeedSkeleton />;
  }

  if (entries.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <Empty
          className="max-w-md"
          description="No activity yet across threads or direct rooms."
          icon={ActivitySquare}
          title="Quiet across the channel."
        />
      </div>
    );
  }

  return (
    <div
      aria-label={`Activity in #${channel}`}
      className="flex flex-1 flex-col overflow-y-auto"
      data-testid="network-activity-feed"
    >
      <div className="border-b border-(--line) px-5 py-2" data-testid="network-activity-subheader">
        <Eyebrow>Recent activity / Read-only</Eyebrow>
      </div>
      {entries.map(entry => {
        const linkClass = cn(
          "flex flex-col gap-1 border-b border-(--line) px-5 py-3 text-left transition-colors hover:bg-(--hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--accent)"
        );
        const meta = (
          <>
            <div className="flex items-baseline gap-2">
              <Pill
                data-testid={`network-activity-tag-${entry.kind}`}
                mono
                size="xs"
                tone={entry.kind === "thread" ? "info" : "neutral"}
              >
                {entry.kind === "thread" ? "[TH]" : "[DM]"}
              </Pill>
              <Eyebrow aria-hidden="true">/</Eyebrow>
              <Eyebrow>{formatNetworkRelativeTime(entry.timestamp)}</Eyebrow>
            </div>
            <p className="truncate text-sm font-medium text-(--fg)">{entry.title}</p>
            <p className="line-clamp-2 text-small-body text-(--muted)">{entry.preview}</p>
          </>
        );
        if (entry.kind === "thread") {
          return (
            <Link
              className={linkClass}
              data-testid={`network-activity-entry-${entry.id}`}
              key={entry.id}
              params={entry.params}
              to={entry.to}
            >
              {meta}
            </Link>
          );
        }
        return (
          <Link
            className={linkClass}
            data-testid={`network-activity-entry-${entry.id}`}
            key={entry.id}
            params={entry.params}
            to={entry.to}
          >
            {meta}
          </Link>
        );
      })}
    </div>
  );
}
