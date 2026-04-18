import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { bridgeProvidersFixture } from "@/systems/bridges/mocks";

import { BridgeProviderCard } from "../bridge-provider-card";

const meta: Meta<typeof BridgeProviderCard> = {
  title: "systems/bridges/BridgeProviderCard",
  component: BridgeProviderCard,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <CenteredSurface>
      <div className="w-[28rem]">
        <BridgeProviderCard onSelect={() => undefined} provider={bridgeProvidersFixture[0]} />
      </div>
    </CenteredSurface>
  ),
};

export const Disabled: Story = {
  render: () => (
    <CenteredSurface>
      <div className="w-[28rem]">
        <BridgeProviderCard
          onSelect={() => undefined}
          provider={{
            ...bridgeProvidersFixture[0],
            enabled: false,
            health: "unhealthy",
            health_message: "Runtime health checks are failing for this provider.",
          }}
        />
      </div>
    </CenteredSurface>
  ),
};
