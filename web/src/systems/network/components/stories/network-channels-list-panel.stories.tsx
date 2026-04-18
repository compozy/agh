import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { NetworkChannelsListPanel } from "@/systems/network/components/network-channels-list-panel";
import { networkChannelsFixture } from "@/systems/network/mocks";

const meta: Meta<typeof NetworkChannelsListPanel> = {
  title: "systems/network/NetworkChannelsListPanel",
  component: NetworkChannelsListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function NetworkChannelsListPanelFrame(props: ComponentProps<typeof NetworkChannelsListPanel>) {
  return (
    <PanelSurface className="max-w-[280px]">
      <NetworkChannelsListPanel {...props} />
    </PanelSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => (
    <NetworkChannelsListPanelFrame
      channels={networkChannelsFixture.channels}
      onSearchChange={() => undefined}
      onSelectChannel={() => undefined}
      searchQuery=""
      selectedChannel={networkChannelsFixture.channels[0]?.channel ?? null}
    />
  ),
};

export const Loading: Story = {
  args: {},
  render: () => (
    <NetworkChannelsListPanelFrame
      channels={[]}
      isLoading
      onSearchChange={() => undefined}
      onSelectChannel={() => undefined}
      searchQuery=""
      selectedChannel={null}
    />
  ),
};

export const Empty: Story = {
  args: {},
  render: () => (
    <NetworkChannelsListPanelFrame
      channels={[]}
      onSearchChange={() => undefined}
      onSelectChannel={() => undefined}
      searchQuery="missing"
      selectedChannel={null}
    />
  ),
};
