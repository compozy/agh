import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { bridgesListFixture } from "@/systems/bridges/mocks";
import type { BridgeSummary } from "@/systems/bridges/types";

import { BridgeListPanel } from "../bridge-list-panel";

const meta: Meta<typeof BridgeListPanel> = {
  title: "systems/bridges/BridgeListPanel",
  component: BridgeListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const defaultBridges: BridgeSummary[] = [
  ...bridgesListFixture.bridges,
  {
    ...bridgesListFixture.bridges[0],
    display_name: "Ops slack",
    extension_name: "ext-slack",
    id: "brg_ops_slack",
    platform: "slack",
    scope: "global",
    status: "ready",
  },
];

export const Default: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <BridgeListPanel
        bridgeHealth={
          bridgesListFixture.bridge_health ? { ...bridgesListFixture.bridge_health } : {}
        }
        bridges={defaultBridges}
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery=""
        selectedBridgeId={defaultBridges[0]?.id ?? null}
        summary="2 bridges visible"
      />
    </PanelSurface>
  ),
};

export const Empty: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery=""
        selectedBridgeId={null}
        summary="0 bridges visible"
      />
    </PanelSurface>
  ),
};

export const FilteredEmpty: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery="zzzzzz"
        selectedBridgeId={null}
        summary="0 bridges match the filter"
      />
    </PanelSurface>
  ),
};

export const Error: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        errorMessage="Failed to load bridges"
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery=""
        selectedBridgeId={null}
        summary=""
      />
    </PanelSurface>
  ),
};

export const RowSelect: Story = {
  tags: ["play-fn"],
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={defaultBridges}
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery=""
        selectedBridgeId={defaultBridges[0]?.id ?? null}
        summary="2 bridges visible"
      />
    </PanelSurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const row = await canvas.findByTestId(`bridge-item-${defaultBridges[1].id}`);
    await userEvent.click(row);
    await expect(row).toBeVisible();
  },
};
