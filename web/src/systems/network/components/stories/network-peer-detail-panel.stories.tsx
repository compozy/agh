import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkPeerFixture } from "@/systems/network/mocks";

import { NetworkPeerDetailPanel } from "../network-peer-detail-panel";

const meta: Meta<typeof NetworkPeerDetailPanel> = {
  title: "systems/network/NetworkPeerDetailPanel",
  component: NetworkPeerDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <NetworkPeerDetailPanel error={null} isLoading={false} peer={networkPeerFixture} />
    </PanelSurface>
  ),
};

export const NotFound: Story = {
  render: () => (
    <PanelSurface>
      <NetworkPeerDetailPanel
        error={new Error("Network peer not found")}
        isLoading={false}
        peer={undefined}
      />
    </PanelSurface>
  ),
};
