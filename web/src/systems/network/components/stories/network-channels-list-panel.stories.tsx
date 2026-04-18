import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkChannelsFixture } from "@/systems/network/mocks";

import { NetworkChannelsListPanel } from "../network-channels-list-panel";

const meta: Meta<typeof NetworkChannelsListPanel> = {
  title: "systems/network/NetworkChannelsListPanel",
  component: NetworkChannelsListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function NetworkChannelsListPanelFrame(
  props: React.ComponentProps<typeof NetworkChannelsListPanel>
) {
  return (
    <PanelSurface className="max-w-[280px]">
      <NetworkChannelsListPanel {...props} />
    </PanelSurface>
  );
}

export const Default: Story = {
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
