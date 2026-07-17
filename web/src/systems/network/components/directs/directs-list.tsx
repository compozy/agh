import { Link } from "@tanstack/react-router";

import { Button, ListingRow, Eyebrow, Skeleton, SkeletonRows } from "@agh/ui";

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
  total?: number;
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void | Promise<void>;
  /** Local session id used to identify which side of the direct room is remote. */
  selfSessionId?: string;
  members?: ReadonlyArray<ChannelMember>;
  onNewDirect?: () => void;
}

function pickOtherSessionId(direct: NetworkDirectRoomSummary, selfSessionId?: string): string {
  if (!selfSessionId) {
    return direct.session_a;
  }
  if (direct.session_a === selfSessionId) {
    return direct.session_b;
  }
  return direct.session_a;
}

interface DirectsListRowProps {
  workspaceId: string;
  channel: string;
  direct: NetworkDirectRoomSummary;
  active: boolean;
  selfSessionId?: string;
  member?: ChannelMember;
}

function DirectsListRow({
  workspaceId,
  channel,
  direct,
  active,
  selfSessionId,
  member,
}: DirectsListRowProps) {
  const otherSessionId = pickOtherSessionId(direct, selfSessionId);
  const otherPeerId = member?.peerId ?? otherSessionId;
  const role: ChannelMemberRole | undefined = member?.role;
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

function buildMemberLookup(
  members: ReadonlyArray<ChannelMember> | undefined
): Map<string, ChannelMember> {
  const map = new Map<string, ChannelMember>();
  if (!members) {
    return map;
  }
  for (const member of members) {
    if (member.sessionId) {
      map.set(member.sessionId, member);
    }
  }
  return map;
}

export function DirectsList({
  workspaceId,
  channel,
  directs,
  activeDirectId,
  isLoading,
  total,
  hasMore = false,
  isLoadingMore = false,
  onLoadMore,
  selfSessionId,
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

  const memberBySessionId = buildMemberLookup(members);

  return (
    <div
      aria-label={`Direct rooms in #${channel}`}
      aria-live="polite"
      className="flex flex-1 flex-col overflow-y-auto"
      data-testid="network-direct-list"
    >
      {directs.map(direct => {
        const otherSessionId = pickOtherSessionId(direct, selfSessionId);
        return (
          <DirectsListRow
            active={direct.direct_id === activeDirectId}
            channel={channel}
            direct={direct}
            key={direct.direct_id}
            member={memberBySessionId.get(otherSessionId)}
            selfSessionId={selfSessionId}
            workspaceId={workspaceId}
          />
        );
      })}
      {hasMore && onLoadMore ? (
        <div className="flex items-center justify-between gap-3 border-t border-line px-4 py-3">
          <span className="text-small-body text-muted">
            {directs.length} of {total ?? directs.length} direct rooms loaded
          </span>
          <Button
            aria-busy={isLoadingMore}
            aria-label="Load more direct rooms"
            disabled={isLoadingMore}
            onClick={() => void onLoadMore()}
            size="sm"
            variant="outline"
          >
            {isLoadingMore ? "Loading rooms…" : "Load more direct rooms"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
