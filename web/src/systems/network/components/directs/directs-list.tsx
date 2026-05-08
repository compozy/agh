import { Link } from "@tanstack/react-router";

import { Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";

import type { ChannelMember, ChannelMemberRole } from "../../hooks/use-channel-members";
import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary } from "../../types";
import { DirectsEmpty } from "../empty-states/directs-empty";
import { MessageAvatar } from "../timeline/message-avatar";

export interface DirectsListProps {
  channel: string;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  activeDirectId: string | null;
  isLoading: boolean;
  /** Local peer id used to identify which side of `peer_a/peer_b` is "the other peer". */
  selfPeerId?: string;
  members?: ReadonlyArray<ChannelMember>;
  onNewDirect?: () => void;
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
  role?: ChannelMemberRole;
}

function DirectsListRow({ channel, direct, active, selfPeerId, role }: DirectsListRowProps) {
  const otherPeerId = pickOtherPeerId(direct, selfPeerId);
  const lastActivity = formatNetworkRelativeTime(direct.last_activity_at ?? null);

  return (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(
        "group flex items-start gap-3 border-b border-(--color-divider) px-5 py-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent",
        active ? "bg-(--color-accent-tint)" : "hover:bg-(--color-hover)"
      )}
      data-testid={`network-direct-list-row-${direct.direct_id}`}
      params={{ channel, directId: direct.direct_id }}
      to="/network/$channel/directs/$directId"
    >
      <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={36} />

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-semibold text-(--color-text-primary)">
            @{otherPeerId}
          </p>
          {role ? (
            <span
              className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)"
              data-testid={`network-direct-list-row-role-${direct.direct_id}`}
            >
              {role === "agent" ? "AGENT" : "HUMAN"}
            </span>
          ) : null}
        </div>
        <p className="line-clamp-2 text-small-body text-(--color-text-secondary)">
          {direct.last_message_preview ?? "No messages yet."}
        </p>
      </div>

      <span
        className="shrink-0 self-start font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)"
        data-testid={`network-direct-list-row-time-${direct.direct_id}`}
      >
        {lastActivity}
      </span>
    </Link>
  );
}

function DirectsListSkeleton() {
  return (
    <div className="space-y-0" data-testid="network-direct-list-skeleton">
      {[0, 1, 2].map(index => (
        <div className="flex gap-3 border-b border-(--color-divider) px-5 py-3" key={index}>
          <Skeleton className="size-9 rounded-chip" />
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

function buildRoleLookup(
  members: ReadonlyArray<ChannelMember> | undefined
): Map<string, ChannelMemberRole> {
  const map = new Map<string, ChannelMemberRole>();
  if (!members) {
    return map;
  }
  for (const member of members) {
    map.set(member.peerId, member.role);
  }
  return map;
}

export function DirectsList({
  channel,
  directs,
  activeDirectId,
  isLoading,
  selfPeerId,
  members,
  onNewDirect,
}: DirectsListProps) {
  if (isLoading && directs.length === 0) {
    return <DirectsListSkeleton />;
  }

  if (directs.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <DirectsEmpty className="max-w-md" onNewDirect={onNewDirect} />
      </div>
    );
  }

  const roleByPeerId = buildRoleLookup(members);

  return (
    <div
      aria-label={`Direct rooms in #${channel}`}
      aria-live="polite"
      className="flex flex-1 flex-col overflow-y-auto"
      data-testid="network-direct-list"
    >
      {directs.map(direct => {
        const otherPeerId = pickOtherPeerId(direct, selfPeerId);
        return (
          <DirectsListRow
            active={direct.direct_id === activeDirectId}
            channel={channel}
            direct={direct}
            key={direct.direct_id}
            role={roleByPeerId.get(otherPeerId)}
            selfPeerId={selfPeerId}
          />
        );
      })}
    </div>
  );
}
