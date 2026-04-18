import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { bridgeProvidersFixture } from "@/systems/bridges/mocks";

import { BridgeEmptyState } from "../bridge-empty-state";

const meta: Meta<typeof BridgeEmptyState> = {
  title: "systems/bridges/BridgeEmptyState",
  component: BridgeEmptyState,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <BridgeEmptyState onCreate={() => undefined} providers={bridgeProvidersFixture} />
    </PanelSurface>
  ),
};
