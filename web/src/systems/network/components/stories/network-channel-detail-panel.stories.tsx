import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkChannelFixture, networkChannelMessagesFixture } from "@/systems/network/mocks";

import { NetworkChannelDetailPanel } from "../network-channel-detail-panel";

const meta: Meta<typeof NetworkChannelDetailPanel> = {
  title: "systems/network/NetworkChannelDetailPanel",
  component: NetworkChannelDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <NetworkChannelDetailPanel
        channel={networkChannelFixture}
        error={null}
        isLoading={false}
        isMessagesLoading={false}
        messages={networkChannelMessagesFixture}
      />
    </PanelSurface>
  ),
};

export const NotFound: Story = {
  render: () => (
    <PanelSurface>
      <NetworkChannelDetailPanel
        channel={undefined}
        error={new Error("Network channel not found")}
        isLoading={false}
        isMessagesLoading={false}
        messages={[]}
      />
    </PanelSurface>
  ),
};
