import { Link } from "@tanstack/react-router";

import { SidebarSectionLabel, Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type {
  NetworkChannelSummary,
  NetworkDirectRoomSummary,
  NetworkRecentEntry,
} from "../../types";
import { MessageAvatar } from "../timeline/message-avatar";
import { ChannelRailRecents } from "./channel-rail-recents";
import { ChannelRailRow } from "./channel-rail-row";

const CHANNELS_HEADING = "Channels";
const DIRECT_ROOMS_HEADING = "Direct Rooms";

export interface ChannelRailProps {
  pinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  unpinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  recents: ReadonlyArray<NetworkRecentEntry>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  loading: {
    channels: boolean;
    recents: boolean;
    directs: boolean;
  };
  activeChannel: string | null;
  activeDirectId: string | null;
  selfPeerId: string | null;
  isPinned: (channel: string) => boolean;
  onTogglePinned: (channel: string) => void;
  hasUnread: (channel: string) => boolean;
}

function pickOtherPeerId(
  direct: NetworkDirectRoomSummary,
  selfPeerId: string | null | undefined
): string {
  if (!selfPeerId) {
    return direct.peer_a;
  }
  if (direct.peer_a === selfPeerId) {
    return direct.peer_b;
  }
  return direct.peer_a;
}

interface DirectRoomRailRowProps {
  channel: string;
  direct: NetworkDirectRoomSummary;
  active: boolean;
  selfPeerId: string | null;
}

function DirectRoomRailRow({ channel, direct, active, selfPeerId }: DirectRoomRailRowProps) {
  const otherPeerId = pickOtherPeerId(direct, selfPeerId);
  const lastActivity = direct.last_activity_at
    ? formatNetworkRelativeTime(direct.last_activity_at)
    : null;
  return (
    <Link
      aria-current={active ? "page" : undefined}
      className={cn(NAV_ROW_CLASS, active && ACTIVE_NAV_ROW_CLASS)}
      data-active={active}
      data-testid={`network-direct-rail-row-${direct.direct_id}`}
      params={{ channel, directId: direct.direct_id }}
      to="/network/$channel/directs/$directId"
    >
      {active ? <span aria-hidden="true" className={ACTIVE_NAV_INDICATOR_CLASS} /> : null}
      <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={20} />
      <span className="min-w-0 flex-1 truncate">@{otherPeerId}</span>
      {lastActivity ? (
        <span className="shrink-0 font-mono text-badge text-(--subtle)">{lastActivity}</span>
      ) : null}
    </Link>
  );
}

export function ChannelRail({
  pinnedChannels,
  unpinnedChannels,
  recents,
  directs,
  loading,
  activeChannel,
  activeDirectId,
  selfPeerId,
  isPinned,
  onTogglePinned,
  hasUnread,
}: ChannelRailProps) {
  const {
    channels: isChannelsLoading,
    recents: isRecentsLoading,
    directs: isDirectsLoading,
  } = loading;
  const hasAnyChannel = pinnedChannels.length + unpinnedChannels.length > 0;
  const hasAnyDirect = directs.length > 0;

  return (
    <aside
      aria-label="Network channels"
      className="flex min-h-0 w-[260px] shrink-0 flex-col border-r border-(--line) bg-(--canvas)"
      data-testid="network-channel-rail"
    >
      <div className="flex-1 space-y-5 overflow-y-auto px-3 py-4">
        <section aria-label="Channels" className="space-y-1">
          <SidebarSectionLabel>{CHANNELS_HEADING}</SidebarSectionLabel>

          {isChannelsLoading && !hasAnyChannel ? (
            <div className="space-y-1.5 px-2 py-1" data-testid="network-channels-loading">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="size-4/5" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-3/5" />
            </div>
          ) : !hasAnyChannel ? (
            <p
              className="px-2 py-1 text-eyebrow text-(--subtle)"
              data-testid="network-channels-empty"
            >
              No channels yet.
            </p>
          ) : (
            <div className="space-y-0.5">
              {pinnedChannels.map(channel => (
                <ChannelRailRow
                  active={channel.channel === activeChannel}
                  channel={channel}
                  hasUnread={hasUnread(channel.channel)}
                  isPinned={true}
                  key={channel.channel}
                  onTogglePinned={onTogglePinned}
                />
              ))}
              {unpinnedChannels.map(channel => (
                <ChannelRailRow
                  active={channel.channel === activeChannel}
                  channel={channel}
                  hasUnread={hasUnread(channel.channel)}
                  isPinned={isPinned(channel.channel)}
                  key={channel.channel}
                  onTogglePinned={onTogglePinned}
                />
              ))}
            </div>
          )}
        </section>

        <section aria-label="Direct rooms" className="space-y-1" data-testid="network-rail-directs">
          <SidebarSectionLabel>{DIRECT_ROOMS_HEADING}</SidebarSectionLabel>
          {!activeChannel ? (
            <p
              className="px-2 py-1 text-eyebrow text-(--subtle)"
              data-testid="network-rail-directs-empty"
            >
              Select a channel to see direct rooms.
            </p>
          ) : isDirectsLoading && !hasAnyDirect ? (
            <div className="space-y-1.5 px-2 py-1" data-testid="network-rail-directs-loading">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="size-4/5" />
            </div>
          ) : !hasAnyDirect ? (
            <p
              className="px-2 py-1 text-eyebrow text-(--subtle)"
              data-testid="network-rail-directs-empty"
            >
              No direct rooms yet.
            </p>
          ) : (
            <div className="space-y-0.5">
              {directs.map(direct => (
                <DirectRoomRailRow
                  active={direct.direct_id === activeDirectId}
                  channel={activeChannel}
                  direct={direct}
                  key={direct.direct_id}
                  selfPeerId={selfPeerId}
                />
              ))}
            </div>
          )}
        </section>

        <ChannelRailRecents recents={recents} isLoading={isRecentsLoading} />
      </div>
    </aside>
  );
}
