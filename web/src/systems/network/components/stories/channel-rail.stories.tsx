import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import {
  networkChannelsFixture,
  networkDirectRoomsFixture,
  networkThreadsFixture,
} from "@/systems/network/mocks";
import {
  ChannelRail,
  ChannelRailRecents,
  ChannelRailRow,
  ChannelTabs,
} from "@/systems/network/components/shell";
import type { NetworkChannelSummary, NetworkRecentEntry, NetworkSurface } from "@/systems/network";

const allChannels: NetworkChannelSummary[] = [...networkChannelsFixture.channels];

const recents: NetworkRecentEntry[] = [
  {
    surface: "thread" satisfies NetworkSurface,
    channel: networkThreadsFixture[0]?.channel ?? "builders",
    containerId: networkThreadsFixture[0]?.thread_id ?? "thread_one",
    preview: "Open both corridors at 18:30 UTC.",
    lastActivityAt: networkThreadsFixture[0]?.last_activity_at ?? null,
    hasUnread: true,
    participantLabel: "6 peers",
  },
  {
    surface: "direct" satisfies NetworkSurface,
    channel: networkDirectRoomsFixture[0]?.channel ?? "builders",
    containerId: networkDirectRoomsFixture[0]?.direct_id ?? "direct_one",
    preview: "Replay finished. BR timeout copy is clear.",
    lastActivityAt: networkDirectRoomsFixture[0]?.last_activity_at ?? null,
    hasUnread: false,
    participantLabel: "two-party",
  },
];

const meta: Meta<typeof ChannelRail> = {
  title: "systems/network/ChannelRail",
  component: ChannelRail,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" as const },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const pinnedSet = new Set([allChannels[0]?.channel ?? ""]);

export const Default: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px]">
      <ChannelRail
        activeChannel={allChannels[0]?.channel ?? null}
        activeDirectId={null}
        directs={networkDirectRoomsFixture}
        hasUnread={() => true}
        isChannelsLoading={false}
        isDirectsLoading={false}
        isPinned={channel => pinnedSet.has(channel)}
        isRecentsLoading={false}
        onTogglePinned={() => undefined}
        pinnedChannels={allChannels.filter(channel => pinnedSet.has(channel.channel))}
        recents={recents}
        selfPeerId={null}
        unpinnedChannels={allChannels.filter(channel => !pinnedSet.has(channel.channel))}
      />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px]">
      <ChannelRail
        activeChannel={null}
        activeDirectId={null}
        directs={[]}
        hasUnread={() => false}
        isChannelsLoading
        isDirectsLoading
        isPinned={() => false}
        isRecentsLoading
        onTogglePinned={() => undefined}
        pinnedChannels={[]}
        recents={[]}
        selfPeerId={null}
        unpinnedChannels={[]}
      />
    </PanelSurface>
  ),
};

export const Empty: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px]">
      <ChannelRail
        activeChannel={null}
        activeDirectId={null}
        directs={[]}
        hasUnread={() => false}
        isChannelsLoading={false}
        isDirectsLoading={false}
        isPinned={() => false}
        isRecentsLoading={false}
        onTogglePinned={() => undefined}
        pinnedChannels={[]}
        recents={[]}
        selfPeerId={null}
        unpinnedChannels={[]}
      />
    </PanelSurface>
  ),
};

/**
 * Individual rail row, recents, and tab primitives used inside the channel rail
 * and network header.
 */
export const RailPrimitives: StoryObj<typeof ChannelRailRow> = {
  render: () => (
    <PanelSurface className="min-h-[360px] p-4">
      <div className="grid max-w-sm gap-5">
        <div className="space-y-1">
          <ChannelRailRow
            active
            channel={allChannels[0]!}
            hasUnread
            isPinned
            onTogglePinned={() => undefined}
          />
          <ChannelRailRow
            active={false}
            channel={allChannels[1]!}
            hasUnread={false}
            isPinned={false}
            onTogglePinned={() => undefined}
          />
        </div>
        <ChannelRailRecents recents={recents} isLoading={false} />
        <ChannelTabs
          channel={allChannels[0]?.channel ?? "launch-war-room"}
          activeTab="threads"
          threadCount={12}
          directCount={3}
        />
      </div>
    </PanelSurface>
  ),
};
