import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fireEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { networkPeersFixture } from "@/systems/network/mocks";

import { NetworkPeersListPanel } from "../network-peers-list-panel";

const meta: Meta<typeof NetworkPeersListPanel> = {
  title: "systems/network/NetworkPeersListPanel",
  component: NetworkPeersListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function NetworkPeersListPanelFrame(props: ComponentProps<typeof NetworkPeersListPanel>) {
  return (
    <PanelSurface className="max-w-[320px]">
      <NetworkPeersListPanel {...props} />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => (
    <NetworkPeersListPanelFrame
      onSearchChange={() => undefined}
      onSelectPeer={() => undefined}
      peers={networkPeersFixture}
      searchQuery=""
      selectedPeerId={networkPeersFixture[0]?.peer_id ?? null}
    />
  ),
};

export const Loading: Story = {
  render: () => (
    <NetworkPeersListPanelFrame
      isLoading
      onSearchChange={() => undefined}
      onSelectPeer={() => undefined}
      peers={[]}
      searchQuery=""
      selectedPeerId={null}
    />
  ),
};

export const Empty: Story = {
  render: () => (
    <NetworkPeersListPanelFrame
      onSearchChange={() => undefined}
      onSelectPeer={() => undefined}
      peers={[]}
      searchQuery="nobody"
      selectedPeerId={null}
    />
  ),
};

export const Error: Story = {
  render: () => (
    <NetworkPeersListPanelFrame
      errorMessage="Peer discovery failed"
      onSearchChange={() => undefined}
      onSelectPeer={() => undefined}
      peers={[]}
      searchQuery=""
      selectedPeerId={null}
    />
  ),
};

export const SearchFilter: Story = {
  tags: ["play-fn"],
  render: Default.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const search = await canvas.findByTestId("network-peer-search-input");
    fireEvent.change(search, { target: { value: "remote" } });
    await expect(search).toHaveValue("remote");
  },
};
