import { Activity } from "lucide-react";
import { Link } from "@tanstack/react-router";

import {
  Empty,
  Eyebrow,
  Item,
  ItemContent,
  ItemFooter,
  ItemTitle,
  Skeleton,
  SkeletonRows,
} from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary, NetworkThreadSummary } from "../../types";

export interface InspectorActivityFeedProps {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  isLoading?: boolean;
  className?: string;
  /** Caps the rendered feed - `_design.md` §5.8.3 inspector "LAST 10 TRANSITIONS". */
  limit?: number;
}

interface FeedEntry {
  id: string;
  kind: "thread" | "direct";
  preview: string;
  title: string;
  timestamp: string | null;
}

function buildEntries(
  threads: ReadonlyArray<NetworkThreadSummary>,
  directs: ReadonlyArray<NetworkDirectRoomSummary>,
  limit: number
): { entry: FeedEntry; href: string }[] {
  const entries: { entry: FeedEntry; href: string }[] = [];

  for (const thread of threads) {
    entries.push({
      entry: {
        id: `thread:${thread.thread_id}`,
        kind: "thread",
        preview: thread.last_message_preview ?? "No messages yet.",
        timestamp: thread.last_activity_at ?? null,
        title: thread.title ?? "Untitled thread",
      },
      href: thread.thread_id,
    });
  }

  for (const direct of directs) {
    entries.push({
      entry: {
        id: `direct:${direct.direct_id}`,
        kind: "direct",
        preview: direct.last_message_preview ?? "No messages yet.",
        timestamp: direct.last_activity_at ?? null,
        title: `${direct.peer_a} ↔ ${direct.peer_b}`,
      },
      href: direct.direct_id,
    });
  }

  entries.sort((left, right) => {
    const leftTs = left.entry.timestamp ? new Date(left.entry.timestamp).getTime() : 0;
    const rightTs = right.entry.timestamp ? new Date(right.entry.timestamp).getTime() : 0;
    return rightTs - leftTs;
  });

  return entries.slice(0, limit);
}

function ActivitySkeleton() {
  return (
    <SkeletonRows
      aria-hidden="true"
      count={3}
      data-testid="network-inspector-activity-skeleton"
      rowClassName="border-b border-(--color-divider) px-4 py-3"
    >
      <Skeleton className="h-3 w-24" />
      <Skeleton className="size-3/4" />
    </SkeletonRows>
  );
}

export function InspectorActivityFeed({
  channel,
  threads,
  directs,
  isLoading = false,
  className,
  limit = 10,
}: InspectorActivityFeedProps) {
  const entries = buildEntries(threads, directs, limit);

  if (isLoading && entries.length === 0) {
    return <ActivitySkeleton />;
  }

  if (entries.length === 0) {
    return (
      <div className="flex justify-center px-4 py-6">
        <Empty
          className="max-w-sm"
          description="No transitions in this channel yet."
          fill={false}
          icon={Activity}
          title="Quiet for now."
        />
      </div>
    );
  }

  return (
    <div
      aria-label="Recent transitions"
      className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto", className)}
      data-testid="network-inspector-activity-feed"
      role="list"
    >
      {entries.map(({ entry, href }) => (
        <Item
          className="rounded-none border-b border-(--color-divider) px-4 py-3 last:border-b-0"
          data-testid={`network-inspector-activity-${entry.id}`}
          key={entry.id}
          render={
            entry.kind === "thread" ? (
              <Link params={{ channel, threadId: href }} to="/network/$channel/threads/$threadId" />
            ) : (
              <Link params={{ channel, directId: href }} to="/network/$channel/directs/$directId" />
            )
          }
          role="listitem"
          selectable
        >
          <ItemContent>
            <ItemFooter>
              <ItemTitle className="min-w-0 text-xs">
                <span className="truncate">{entry.title}</span>
              </ItemTitle>
              <Eyebrow weight="medium">{formatNetworkRelativeTime(entry.timestamp)}</Eyebrow>
            </ItemFooter>
            <p className="line-clamp-2 text-xs text-(--color-text-secondary)">{entry.preview}</p>
          </ItemContent>
        </Item>
      ))}
    </div>
  );
}
