import type { ReactNode } from "react";

import type {
  NetworkChannel,
  NetworkChannelSummary,
  NetworkDirectRoomSummary,
  NetworkRecentEntry,
} from "../../types";
import { RightRail, type RightRailMode } from "@agh/ui";

import { ChannelHeader } from "./channel-header";
import { ChannelRail } from "./channel-rail";
import type { ChannelTab } from "./channel-tabs-types";
import { ChannelToolbar } from "./channel-toolbar";

export interface NetworkShellProps {
  pinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  unpinnedChannels: ReadonlyArray<NetworkChannelSummary>;
  recents: ReadonlyArray<NetworkRecentEntry>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  loading: {
    channels: boolean;
    recents: boolean;
    directs: boolean;
  };
  activeChannel: NetworkChannelSummary | null;
  activeChannelDetail: NetworkChannel | null;
  activeTab: ChannelTab;
  activeDirectId: string | null;
  selfPeerId: string | null;
  threadCount: number | null;
  directCount: number | null;
  openWorkCount: number;
  rightRailOpen: boolean;
  rightRailMode: RightRailMode;
  rightRailContent?: ReactNode;
  inspectorOpen: boolean;
  onInspectorToggle: () => void;
  isPinned: (channel: string) => boolean;
  onTogglePinned: (channel: string) => void;
  hasUnread: (channel: string) => boolean;
  children: ReactNode;
}

export function NetworkShell({
  pinnedChannels,
  unpinnedChannels,
  recents,
  directs,
  loading,
  activeChannel,
  activeChannelDetail,
  activeTab,
  activeDirectId,
  selfPeerId,
  threadCount,
  directCount,
  openWorkCount,
  rightRailOpen,
  rightRailMode,
  rightRailContent,
  inspectorOpen,
  onInspectorToggle,
  isPinned,
  onTogglePinned,
  hasUnread,
  children,
}: NetworkShellProps) {
  return (
    <div className="flex min-h-0 flex-1 bg-(--canvas)" data-testid="network-shell">
      <ChannelRail
        activeChannel={activeChannel?.channel ?? null}
        activeDirectId={activeDirectId}
        directs={directs}
        hasUnread={hasUnread}
        loading={loading}
        isPinned={isPinned}
        onTogglePinned={onTogglePinned}
        pinnedChannels={pinnedChannels}
        recents={recents}
        selfPeerId={selfPeerId}
        unpinnedChannels={unpinnedChannels}
      />

      <main className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid="network-main-pane">
        {activeChannel ? (
          <>
            <ChannelHeader
              channel={activeChannel}
              detail={activeChannelDetail}
              inspectorOpen={inspectorOpen}
              onInspectorToggle={onInspectorToggle}
              openWorkCount={openWorkCount}
            />
            <ChannelToolbar
              activeTab={activeTab}
              channel={activeChannel.channel}
              directCount={directCount}
              threadCount={threadCount}
            />
          </>
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
