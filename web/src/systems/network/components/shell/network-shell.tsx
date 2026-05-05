import type { ReactNode } from "react";

import type { NetworkChannel, NetworkChannelSummary, NetworkRecentEntry } from "../../types";
import { ChannelHeader } from "./channel-header";
import { ChannelRail } from "./channel-rail";
import { RightRail, type RightRailMode } from "./right-rail";
import type { ChannelTab } from "./channel-tabs";

export interface NetworkShellProps {
  pinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  unpinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  recents: ReadonlyArray<NetworkRecentEntry>;
  isChannelsLoading: boolean;
  isRecentsLoading: boolean;
  activeChannel: NetworkChannelSummary | null;
  activeChannelDetail: NetworkChannel | null;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
  openWorkCount: number;
  rightRailOpen: boolean;
  rightRailMode: RightRailMode;
  rightRailContent?: ReactNode;
  isPinned: (channel: string) => boolean;
  onTogglePinned: (channel: string) => void;
  hasUnread: (channel: string) => boolean;
  children: ReactNode;
}

export function NetworkShell({
  pinnedChannels,
  unpinnedChannels,
  recents,
  isChannelsLoading,
  isRecentsLoading,
  activeChannel,
  activeChannelDetail,
  activeTab,
  threadCount,
  directCount,
  openWorkCount,
  rightRailOpen,
  rightRailMode,
  rightRailContent,
  isPinned,
  onTogglePinned,
  hasUnread,
  children,
}: NetworkShellProps) {
  return (
    <div className="flex min-h-0 flex-1 bg-[color:var(--color-canvas)]" data-testid="network-shell">
      <ChannelRail
        activeChannel={activeChannel?.channel ?? null}
        hasUnread={hasUnread}
        isChannelsLoading={isChannelsLoading}
        isPinned={isPinned}
        isRecentsLoading={isRecentsLoading}
        onTogglePinned={onTogglePinned}
        pinnedChannels={pinnedChannels}
        recents={recents}
        unpinnedChannels={unpinnedChannels}
      />

      <main className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid="network-main-pane">
        {activeChannel ? (
          <ChannelHeader
            activeTab={activeTab}
            channel={activeChannel}
            detail={activeChannelDetail}
            directCount={directCount}
            openWorkCount={openWorkCount}
            threadCount={threadCount}
          />
        ) : null}

        <div className="flex min-h-0 flex-1 flex-col" data-testid="network-tab-panel">
          {children}
        </div>
      </main>

      <RightRail mode={rightRailMode} open={rightRailOpen}>
        {rightRailContent}
      </RightRail>
    </div>
  );
}
