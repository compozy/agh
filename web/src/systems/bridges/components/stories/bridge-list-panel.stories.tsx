import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { bridgesListFixture } from "@/systems/bridges/mocks";

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

export const Default: Story = {
  render: () => (
    <PanelSurface className="max-w-[300px]">
      <BridgeListPanel
        bridgeHealth={
          bridgesListFixture.bridge_health ? { ...bridgesListFixture.bridge_health } : {}
        }
        bridges={bridgesListFixture.bridges}
        onSearchChange={() => undefined}
        onSelectBridge={() => undefined}
        searchQuery=""
        selectedBridgeId={bridgesListFixture.bridges[0]?.id ?? null}
        summary="1 bridges visible"
      />
    </PanelSurface>
  ),
};
