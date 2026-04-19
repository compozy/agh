import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fireEvent, userEvent, within } from "storybook/test";

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
    <PanelSurface className="max-w-[320px]">
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

export const Error: Story = {
  render: () => (
    <NetworkChannelsListPanelFrame
      channels={[]}
      errorMessage="Network status unavailable"
      onSearchChange={() => undefined}
      onSelectChannel={() => undefined}
      searchQuery=""
      selectedChannel={null}
    />
  ),
};

export const SearchFilter: Story = {
  tags: ["play-fn"],
  render: Default.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const search = await canvas.findByTestId("network-channel-search-input");
    fireEvent.change(search, { target: { value: "release" } });
    await expect(search).toHaveValue("release");
  },
};

export const RowSelect: Story = {
  tags: ["play-fn"],
  render: Default.render,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const target = networkChannelsFixture.channels[1]!;
    await userEvent.click(await canvas.findByTestId(`network-channel-item-${target.channel}`));
  },
};
