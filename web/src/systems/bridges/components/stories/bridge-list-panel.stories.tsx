import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { storyWorkspaceIds } from "@/storybook/fintech-scenario";
import { bridgesListFixture } from "@/systems/bridges/mocks";
import type { BridgeSummary } from "@/systems/bridges/types";

import { BridgeListPanel } from "../bridge-list-panel";

const meta: Meta<typeof BridgeListPanel> = {
  title: "systems/bridges/components/BridgeListPanel",
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
    display_name: "Fraud alerts",
    extension_name: "ext-slack",
    id: "brg_fraud_alerts",
    platform: "slack",
    scope: "global",
    status: "ready",
    workspace_id: undefined,
  },
];

const defaultProps = {
  activeWorkspaceId: storyWorkspaceIds.hq as string | null,
  bridgeHealth: bridgesListFixture.bridge_health ? { ...bridgesListFixture.bridge_health } : {},
  bridges: defaultBridges,
  onClearFilters: () => undefined,
  platformFilter: null,
  scopeFilter: "all" as const,
  searchQuery: "",
  statusFilter: null,
  view: "rows" as const,
};

export const Default: Story = {
  render: () => (
    <PanelSurface className="max-w-3xl">
      <BridgeListPanel {...defaultProps} />
    </PanelSurface>
  ),
};

export const Cards: Story = {
  render: () => (
    <PanelSurface className="max-w-5xl">
      <BridgeListPanel {...defaultProps} view="cards" />
    </PanelSurface>
  ),
};

export const Empty: Story = {
  render: () => (
    <PanelSurface className="max-w-3xl">
      <BridgeListPanel {...defaultProps} bridges={[]} />
    </PanelSurface>
  ),
};

export const FilteredEmpty: Story = {
  render: () => (
    <PanelSurface className="max-w-3xl">
      <BridgeListPanel {...defaultProps} bridges={[]} searchQuery="zzzzzz" />
    </PanelSurface>
  ),
};

export const Error: Story = {
  render: () => (
    <PanelSurface className="max-w-3xl">
      <BridgeListPanel {...defaultProps} bridges={[]} errorMessage="Failed to load bridges" />
    </PanelSurface>
  ),
};

export const RowSelect: Story = {
  tags: ["play-fn"],
  render: () => (
    <PanelSurface className="max-w-3xl">
      <BridgeListPanel {...defaultProps} />
    </PanelSurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const row = await canvas.findByTestId(`bridge-item-${defaultBridges[1].id}`);
    await userEvent.click(row);
    await expect(row).toBeVisible();
  },
};
