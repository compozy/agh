import { SidebarSectionLabel, Skeleton } from "@agh/ui";

import type { NetworkChannelSummary, NetworkRecentEntry } from "../../types";
import { ChannelRailRecents } from "./channel-rail-recents";
import { ChannelRailRow } from "./channel-rail-row";

const CHANNELS_HEADING = "Channels";

export interface ChannelRailProps {
  pinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  unpinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  recents: ReadonlyArray<NetworkRecentEntry>;
  isChannelsLoading: boolean;
  isRecentsLoading: boolean;
  activeChannel: string | null;
  isPinned: (channel: string) => boolean;
  onTogglePinned: (channel: string) => void;
  hasUnread: (channel: string) => boolean;
}

export function ChannelRail({
  pinnedChannels,
  unpinnedChannels,
  recents,
  isChannelsLoading,
  isRecentsLoading,
  activeChannel,
  isPinned,
  onTogglePinned,
  hasUnread,
}: ChannelRailProps) {
  const hasAnyChannel = pinnedChannels.length + unpinnedChannels.length > 0;

  return (
    <aside
      aria-label="Network channels"
      className="flex min-h-0 w-[260px] shrink-0 flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-canvas-deep)]"
      data-testid="network-channel-rail"
    >
      <div className="flex-1 space-y-5 overflow-y-auto px-3 py-4">
        <ChannelRailRecents recents={recents} isLoading={isRecentsLoading} />

        <section aria-label="Channels" className="space-y-1">
          <SidebarSectionLabel>{CHANNELS_HEADING}</SidebarSectionLabel>

          {isChannelsLoading && !hasAnyChannel ? (
            <div className="space-y-1.5 px-2 py-1" data-testid="network-channels-loading">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-4 w-4/5" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-3/5" />
            </div>
          ) : !hasAnyChannel ? (
            <p
              className="px-2 py-1 text-[11px] text-[color:var(--color-text-tertiary)]"
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
      </div>
    </aside>
  );
}
