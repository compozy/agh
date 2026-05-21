import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import {
  bridgeDetailFixture,
  bridgeProvidersFixture,
  bridgeResolveTargetFixture,
  bridgeRoutesFixture,
  bridgeSecretBindingsFixture,
  bridgeTargetsFixture,
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
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={bridgeRoutesFixture}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        targetDirectory={{
          error: null,
          isLoading: false,
          isResolving: false,
          onQueryChange: () => undefined,
          onResolveInputChange: () => undefined,
          onResolveSubmit: () => undefined,
          query: "",
          resolveInput: "Launch room",
          resolveResult: bridgeResolveTargetFixture,
          response: bridgeTargetsFixture,
        }}
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
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={[]}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        targetDirectory={{
          error: null,
          isLoading: false,
          isResolving: false,
          onQueryChange: () => undefined,
          onResolveInputChange: () => undefined,
          onResolveSubmit: () => undefined,
          query: "",
          resolveInput: "",
          resolveResult: null,
          response: { ...bridgeTargetsFixture, cache_stale: true },
        }}
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
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={() => undefined}
        provider={bridgeProvidersFixture[0]}
        routes={[]}
        secretBindings={bridgeSecretBindingsFixture}
        secretInputValues={{ bot_token: "telegram-token" }}
        targetDirectory={{
          error: null,
          isLoading: false,
          isResolving: false,
          onQueryChange: () => undefined,
          onResolveInputChange: () => undefined,
          onResolveSubmit: () => undefined,
          query: "",
          resolveInput: "merchant",
          resolveResult: {
            diagnostic: {
              category: "bridge",
              code: "target_ambiguous",
              data_freshness: "live",
              id: "bridge_target_resolve:brg_launch_room",
              message: "Bridge target matched multiple candidates.",
              severity: "warn",
              title: "Bridge target is ambiguous",
            },
            result: {
              ambiguous: true,
              candidates: bridgeTargetsFixture.targets,
              step: 4,
            },
          },
          response: bridgeTargetsFixture,
        }}
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
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={() => undefined}
        routes={[]}
      />
    </PanelSurface>
  ),
};
