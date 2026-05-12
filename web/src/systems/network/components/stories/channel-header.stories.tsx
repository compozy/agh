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

export const Default: Story = {
  render: () => (
    <PanelSurface className="min-h-[120px]">
      <ChannelHeader
        channel={heroChannel}
        detail={networkChannelFixture}
        inspectorOpen={false}
        onInspectorToggle={() => undefined}
        openWorkCount={2}
      />
    </PanelSurface>
  ),
};

export const NoPeers: Story = {
  render: () => (
    <PanelSurface className="min-h-[120px]">
      <ChannelHeader
        channel={heroChannel}
        detail={null}
        inspectorOpen={false}
        onInspectorToggle={() => undefined}
        openWorkCount={0}
      />
    </PanelSurface>
  ),
};

export const InspectorOpen: Story = {
  render: () => (
    <PanelSurface className="min-h-[120px]">
      <ChannelHeader
        channel={heroChannel}
        detail={networkChannelFixture}
        inspectorOpen
        onInspectorToggle={() => undefined}
        openWorkCount={2}
      />
    </PanelSurface>
  ),
};
