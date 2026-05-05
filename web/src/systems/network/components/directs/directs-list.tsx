import { MessageCircle } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Empty, Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary } from "../../types";
import { MessageAvatar } from "../timeline/message-avatar";

export interface DirectsListProps {
  channel: string;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  activeDirectId: string | null;
  isLoading: boolean;
  /** Local peer id used to identify which side of `peer_a/peer_b` is "the other peer". */
  selfPeerId?: string;
}

function pickOtherPeerId(direct: NetworkDirectRoomSummary, selfPeerId?: string): string {
  if (!selfPeerId) {
    return direct.peer_a;
  }
  if (direct.peer_a === selfPeerId) {
    return direct.peer_b;
  }
  return direct.peer_a;
}

interface DirectsListRowProps {
  channel: string;
  direct: NetworkDirectRoomSummary;
  active: boolean;
  selfPeerId?: string;
}

function DirectsListRow({ channel, direct, active, selfPeerId }: DirectsListRowProps) {
  const otherPeerId = pickOtherPeerId(direct, selfPeerId);
  const lastActivity = formatNetworkRelativeTime(direct.last_activity_at ?? null);
  const messageCount = direct.message_count ?? 0;

  return (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(
        "group flex items-start gap-3 border-b border-[color:var(--color-divider)] px-5 py-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]",
        active ? "bg-[color:var(--color-accent-tint)]" : "hover:bg-[color:var(--color-hover)]"
      )}
      data-testid={`network-direct-list-row-${direct.direct_id}`}
      params={{ channel, directId: direct.direct_id }}
      to="/network/$channel/directs/$directId"
    >
      <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={36} />

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <p className="truncate text-[14px] font-semibold text-[color:var(--color-text-primary)]">
          @{otherPeerId}
        </p>
        <p className="line-clamp-2 text-[13px] text-[color:var(--color-text-secondary)]">
          {direct.last_message_preview ?? "No messages yet."}
        </p>
        <div className="flex items-center gap-3 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          <span data-testid="network-direct-list-row-meta-messages">
            {messageCount} {messageCount === 1 ? "msg" : "msgs"}
          </span>
          <span aria-hidden="true">·</span>
          <span data-testid="network-direct-list-row-meta-time">{lastActivity}</span>
        </div>
      </div>
    </Link>
  );
}

function DirectsListSkeleton() {
  return (
    <div className="space-y-0" data-testid="network-direct-list-skeleton">
      {[0, 1, 2].map(index => (
        <div
          className="flex gap-3 border-b border-[color:var(--color-divider)] px-5 py-3"
          key={index}
        >
          <Skeleton className="size-9 rounded-[4px]" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-1/3" />
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function DirectsList({
  channel,
  directs,
  activeDirectId,
  isLoading,
  selfPeerId,
}: DirectsListProps) {
  if (isLoading && directs.length === 0) {
    return <DirectsListSkeleton />;
  }

  if (directs.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <Empty
          className="max-w-md"
          description="Open one to talk privately with a peer in this channel."
          icon={MessageCircle}
          title="No direct rooms yet."
        />
      </div>
    );
  }

  return (
    <div
      aria-label={`Direct rooms in #${channel}`}
      aria-live="polite"
      className="flex flex-1 flex-col overflow-y-auto"
      data-testid="network-direct-list"
    >
      {directs.map(direct => (
        <DirectsListRow
          active={direct.direct_id === activeDirectId}
          channel={channel}
          direct={direct}
          key={direct.direct_id}
          selfPeerId={selfPeerId}
        />
      ))}
    </div>
  );
}
