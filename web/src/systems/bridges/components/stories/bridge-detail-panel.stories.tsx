import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import {
  bridgeDetailFixture,
  bridgeProvidersFixture,
  bridgeRoutesFixture,
  bridgeSecretBindingsFixture,
} from "@/systems/bridges/mocks";

import { BridgeDetailPanel } from "../bridge-detail-panel";

const meta: Meta<typeof BridgeDetailPanel> = {
  title: "systems/bridges/BridgeDetailPanel",
  component: BridgeDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <BridgeDetailPanel
        bridge={bridgeDetailFixture.bridge}
        error={null}
        health={bridgeDetailFixture.health}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={bridgeRoutesFixture}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        workspaceName={storyDefaultWorkspaceName}
      />
    </PanelSurface>
  ),
};

export const Disabled: Story = {
  render: () => (
    <PanelSurface>
      <BridgeDetailPanel
        bridge={{
          ...bridgeDetailFixture.bridge,
          enabled: false,
          status: "disabled",
        }}
        error={null}
        health={{ ...bridgeDetailFixture.health, status: "disabled" }}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={[]}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        workspaceName={storyDefaultWorkspaceName}
      />
    </PanelSurface>
  ),
};

export const NoRoutes: Story = {
  render: () => (
    <PanelSurface>
      <BridgeDetailPanel
        bridge={bridgeDetailFixture.bridge}
        error={null}
        health={bridgeDetailFixture.health}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={[]}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        workspaceName={storyDefaultWorkspaceName}
      />
    </PanelSurface>
  ),
};

export const Error: Story = {
  render: () => (
    <PanelSurface>
      <BridgeDetailPanel
        bridge={undefined}
        error={new globalThis.Error("Failed to load bridge details")}
        health={undefined}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={() => undefined}
        routes={[]}
      />
    </PanelSurface>
  ),
};
