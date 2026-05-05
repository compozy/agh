import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkChannelFixture, networkChannelsFixture } from "@/systems/network/mocks";
import { ChannelHeader } from "@/systems/network/components/shell";
import type { NetworkChannelSummary } from "@/systems/network";

const heroChannel: NetworkChannelSummary | undefined = networkChannelsFixture.channels[0];

const meta: Meta<typeof ChannelHeader> = {
  title: "systems/network/ChannelHeader",
  component: ChannelHeader,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" as const },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

if (!heroChannel) {
  throw new Error("networkChannelsFixture must include at least one channel for stories");
}

export const ThreadsTabActive: Story = {
  render: () => (
    <PanelSurface className="min-h-[160px]">
      <ChannelHeader
        activeTab="threads"
        channel={heroChannel}
        detail={networkChannelFixture}
        directCount={4}
        openWorkCount={2}
        threadCount={12}
      />
    </PanelSurface>
  ),
};

export const DirectsTabActive: Story = {
  render: () => (
    <PanelSurface className="min-h-[160px]">
      <ChannelHeader
        activeTab="directs"
        channel={heroChannel}
        detail={networkChannelFixture}
        directCount={4}
        openWorkCount={0}
        threadCount={12}
      />
    </PanelSurface>
  ),
};

export const ActivityTabActive: Story = {
  render: () => (
    <PanelSurface className="min-h-[160px]">
      <ChannelHeader
        activeTab="activity"
        channel={heroChannel}
        detail={null}
        directCount={null}
        openWorkCount={0}
        threadCount={null}
      />
    </PanelSurface>
  ),
};
