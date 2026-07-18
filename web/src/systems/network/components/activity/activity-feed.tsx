import { ActivitySquare } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { toast } from "sonner";

import { Button, Empty, Eyebrow, Pill, Skeleton, SkeletonRows } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary, NetworkThreadSummary } from "../../types";

export interface ActivityFeedProps {
  workspaceId: string;
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  status: "loading" | "ready";
  threadTotal?: number;
  directTotal?: number;
  pagination?: {
    threads?: ActivityFeedPagination;
    directs?: ActivityFeedPagination;
  };
  isFiltered?: boolean;
  sort?: "recent_activity" | "created" | "alphabetical";
}

interface ActivityFeedPagination {
  status: "available" | "loading";
  onLoadMore: () => void | Promise<void>;
}

async function loadMore(
  pagination: ActivityFeedPagination,
  fallbackMessage: string
): Promise<void> {
  try {
    await pagination.onLoadMore();
  } catch (error) {
    toast.error(error instanceof Error && error.message ? error.message : fallbackMessage);
  }
}

type ThreadEntry = {
  kind: "thread";
  id: string;
  preview: string;
  title: string;
  timestamp: string | null;
  openedAt: string | null;
  to: "/network/$workspaceId/$channel/threads/$threadId";
  params: { workspaceId: string; channel: string; threadId: string };
};

type DirectEntry = {
  kind: "direct";
  id: string;
  preview: string;
  title: string;
  timestamp: string | null;
  openedAt: string | null;
  to: "/network/$workspaceId/$channel/directs/$directId";
  params: { workspaceId: string; channel: string; directId: string };
};

type ActivityEntry = ThreadEntry | DirectEntry;

function buildEntries(
  workspaceId: string,
  channel: string,
  threads: ReadonlyArray<NetworkThreadSummary>,
  directs: ReadonlyArray<NetworkDirectRoomSummary>,
  sort: "recent_activity" | "created" | "alphabetical"
): ActivityEntry[] {
  const entries: ActivityEntry[] = [];
  for (const thread of threads) {
    entries.push({
      id: `thread:${thread.thread_id}`,
      kind: "thread",
      params: { workspaceId, channel, threadId: thread.thread_id },
      preview: thread.last_message_preview ?? "No messages yet.",
      timestamp: thread.last_activity_at ?? null,
      openedAt: thread.opened_at ?? null,
      title: thread.title ?? "Untitled thread",
      to: "/network/$workspaceId/$channel/threads/$threadId",
    });
  }
  for (const direct of directs) {
    entries.push({
      id: `direct:${direct.direct_id}`,
      kind: "direct",
      params: { workspaceId, channel, directId: direct.direct_id },
      preview: direct.last_message_preview ?? "No messages yet.",
      timestamp: direct.last_activity_at ?? null,
      openedAt: direct.opened_at ?? null,
      title: `${direct.session_a} ↔ ${direct.session_b}`,
      to: "/network/$workspaceId/$channel/directs/$directId",
    });
  }

  return entries.sort((left, right) => {
    if (sort === "alphabetical") {
      const byTitle = left.title.localeCompare(right.title, undefined, { sensitivity: "base" });
      return byTitle === 0 ? left.id.localeCompare(right.id) : byTitle;
    }
    if (sort === "created") {
      const leftCreated = left.openedAt ? new Date(left.openedAt).getTime() : 0;
      const rightCreated = right.openedAt ? new Date(right.openedAt).getTime() : 0;
      const byCreated = leftCreated - rightCreated;
      return byCreated === 0 ? left.id.localeCompare(right.id) : byCreated;
    }
    const leftTs = left.timestamp ? new Date(left.timestamp).getTime() : 0;
    const rightTs = right.timestamp ? new Date(right.timestamp).getTime() : 0;
    const byActivity = rightTs - leftTs;
    return byActivity === 0 ? left.id.localeCompare(right.id) : byActivity;
  });
}

function ActivityFeedSkeleton() {
  return (
    <SkeletonRows
      count={4}
      data-testid="network-activity-feed-skeleton"
      rowClassName="border-b border-line px-5 py-3"
    >
      <Skeleton className="h-3 w-24" />
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-3 w-full" />
    </SkeletonRows>
  );
}

export function ActivityFeed({
  workspaceId,
  channel,
  threads,
  directs,
  status,
  threadTotal,
  directTotal,
  pagination,
  isFiltered = false,
  sort = "recent_activity",
}: ActivityFeedProps) {
  const resolvedThreadTotal = threadTotal === undefined ? threads.length : threadTotal;
  const resolvedDirectTotal = directTotal === undefined ? directs.length : directTotal;
  const entries = buildEntries(workspaceId, channel, threads, directs, sort);
  const threadPagination = pagination?.threads;
  const directPagination = pagination?.directs;

  if (status === "loading" && entries.length === 0) {
    return <ActivityFeedSkeleton />;
  }

  if (entries.length === 0 && !threadPagination && !directPagination) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <Empty
          className="max-w-md"
          description={
            isFiltered
              ? "Try another search or remove a filter."
              : "No activity yet across threads or direct rooms."
          }
          icon={ActivitySquare}
          title={isFiltered ? "No matching activity." : "Quiet across the channel."}
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
      <div className="border-b border-line px-5 py-2" data-testid="network-activity-subheader">
        <Eyebrow>
          Recent activity / {entries.length} loaded / {resolvedThreadTotal + resolvedDirectTotal}{" "}
          total
        </Eyebrow>
      </div>
      {entries.map(entry => {
        const linkClass = cn(
          "flex flex-col gap-1 border-b border-line px-5 py-3 text-left transition-colors hover:bg-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
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
            <p className="truncate text-sm font-medium text-fg">{entry.title}</p>
            <p className="line-clamp-2 text-small-body text-muted">{entry.preview}</p>
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
      {threadPagination || directPagination ? (
        <div className="flex flex-wrap items-center justify-end gap-2 border-t border-line px-5 py-3">
          {threadPagination ? (
            <Button
              aria-busy={threadPagination.status === "loading"}
              disabled={threadPagination.status === "loading"}
              onClick={() => void loadMore(threadPagination, "Failed to load more threads.")}
              size="sm"
              variant="outline"
            >
              {threadPagination.status === "loading" ? "Loading threads…" : "Load more threads"}
            </Button>
          ) : null}
          {directPagination ? (
            <Button
              aria-busy={directPagination.status === "loading"}
              disabled={directPagination.status === "loading"}
              onClick={() => void loadMore(directPagination, "Failed to load more direct rooms.")}
              size="sm"
              variant="outline"
            >
              {directPagination.status === "loading" ? "Loading rooms…" : "Load more direct rooms"}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
