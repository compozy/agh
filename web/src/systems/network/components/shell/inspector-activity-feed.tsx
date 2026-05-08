import { Activity } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Empty, Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary, NetworkThreadSummary } from "../../types";

export interface InspectorActivityFeedProps {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  isLoading?: boolean;
  className?: string;
  /** Caps the rendered feed — `_design.md` §5.8.3 inspector "LAST 10 TRANSITIONS". */
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
    <ul
      aria-hidden="true"
      className="flex flex-col"
      data-testid="network-inspector-activity-skeleton"
    >
      {[0, 1, 2].map(index => (
        <li className="flex flex-col gap-2 border-b border-(--color-divider) px-4 py-3" key={index}>
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-3/4" />
        </li>
      ))}
    </ul>
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
    <ul
      aria-label="Recent transitions"
      className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto", className)}
      data-testid="network-inspector-activity-feed"
    >
      {entries.map(({ entry, href }) =>
        entry.kind === "thread" ? (
          <li className="border-b border-(--color-divider) last:border-b-0" key={entry.id}>
            <Link
              className="flex flex-col gap-1 px-4 py-3 text-left transition-colors hover:bg-(--color-hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
              data-testid={`network-inspector-activity-${entry.id}`}
              params={{ channel, threadId: href }}
              to="/network/$channel/threads/$threadId"
            >
              <div className="flex items-center justify-between gap-2 font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
                <span>{entry.title}</span>
                <span>{formatNetworkRelativeTime(entry.timestamp)}</span>
              </div>
              <p className="line-clamp-2 text-xs text-(--color-text-secondary)">{entry.preview}</p>
            </Link>
          </li>
        ) : (
          <li className="border-b border-(--color-divider) last:border-b-0" key={entry.id}>
            <Link
              className="flex flex-col gap-1 px-4 py-3 text-left transition-colors hover:bg-(--color-hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
              data-testid={`network-inspector-activity-${entry.id}`}
              params={{ channel, directId: href }}
              to="/network/$channel/directs/$directId"
            >
              <div className="flex items-center justify-between gap-2 font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
                <span>{entry.title}</span>
                <span>{formatNetworkRelativeTime(entry.timestamp)}</span>
              </div>
              <p className="line-clamp-2 text-xs text-(--color-text-secondary)">{entry.preview}</p>
            </Link>
          </li>
        )
      )}
    </ul>
  );
}
