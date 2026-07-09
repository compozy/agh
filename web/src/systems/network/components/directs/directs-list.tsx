import { Link } from "@tanstack/react-router";

import { ListingRow, Eyebrow, Skeleton, SkeletonRows } from "@agh/ui";

import type { ChannelMember, ChannelMemberRole } from "../../hooks/use-channel-members";
import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkDirectRoomSummary } from "../../types";
import { DirectsEmpty } from "../empty-states/directs-empty";
import { MessageAvatar } from "../timeline/message-avatar";

export interface DirectsListProps {
  workspaceId: string;
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
  workspaceId: string;
  channel: string;
  direct: NetworkDirectRoomSummary;
  active: boolean;
  selfPeerId?: string;
  role?: ChannelMemberRole;
}

function DirectsListRow({
  workspaceId,
  channel,
  direct,
  active,
  selfPeerId,
  role,
}: DirectsListRowProps) {
  const otherPeerId = pickOtherPeerId(direct, selfPeerId);
  const lastActivity = formatNetworkRelativeTime(direct.last_activity_at ?? null);
  const avatarRole = role === "human" ? "human" : "agent";
  const preview = direct.last_message_preview ?? "No messages yet.";

  return (
    <ListingRow
      aria-current={active ? "page" : undefined}
      data-testid={`network-direct-list-row-${direct.direct_id}`}
      selected={active}
    >
      <ListingRow.Link
        render={
          <Link
            params={{ workspaceId, channel, directId: direct.direct_id }}
            to="/network/$workspaceId/$channel/directs/$directId"
            aria-label={`Open @${otherPeerId}`}
          />
        }
      >
        <ListingRow.Icon className="overflow-hidden p-0">
          <MessageAvatar
            initialFrom={otherPeerId}
            name={otherPeerId}
            ownerRole={avatarRole}
            seed={otherPeerId}
            sizePx={32}
          />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name>
            <ListingRow.Title>@{otherPeerId}</ListingRow.Title>
            {role ? (
              <Eyebrow data-testid={`network-direct-list-row-role-${direct.direct_id}`}>
                {role === "agent" ? "AGENT" : "HUMAN"}
              </Eyebrow>
            ) : null}
          </ListingRow.Name>
          <ListingRow.Description>{preview}</ListingRow.Description>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        <span
          className="shrink-0 text-eyebrow text-faint"
          data-testid={`network-direct-list-row-time-${direct.direct_id}`}
        >
          {lastActivity}
        </span>
      </ListingRow.Trail>
    </ListingRow>
  );
}

function DirectsListSkeleton() {
  return (
    <SkeletonRows
      count={3}
      data-testid="network-direct-list-skeleton"
      rowClassName="flex-row gap-3 border-b border-line px-4 py-3"
    >
      <Skeleton className="size-9 rounded-chip" />
      <div className="flex flex-1 flex-col gap-1.5">
        <Skeleton className="h-3 w-1/3" />
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-2/3" />
      </div>
    </SkeletonRows>
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
  workspaceId,
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
            workspaceId={workspaceId}
          />
        );
      })}
    </div>
  );
}
