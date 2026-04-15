import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { BridgeHealth, BridgeProvider, BridgeRoute, BridgeSummary } from "@/systems/bridges";
import { BridgeDetailPanel } from "@/systems/bridges/components/bridge-detail-panel";

function makeBridge(overrides: Partial<BridgeSummary> = {}): BridgeSummary {
  return {
    created_at: "2026-04-13T12:00:00Z",
    delivery_defaults: {
      mode: "reply",
      peer_id: "peer_123",
    },
    display_name: "Support",
    dm_policy: "allowlist",
    enabled: true,
    extension_name: "ext-telegram",
    id: "brg_support",
    platform: "telegram",
    provider_config: {
      mode: "bot",
    },
    routing_policy: {
      include_group: true,
      include_peer: true,
      include_thread: true,
    },
    scope: "workspace",
    status: "ready",
    updated_at: "2026-04-13T12:30:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeHealth(overrides: Partial<BridgeHealth> = {}): BridgeHealth {
  return {
    auth_failures_total: 0,
    bridge_instance_id: "brg_support",
    delivery_backlog: 0,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    route_count: 0,
    status: "ready",
    ...overrides,
  };
}

function makeProvider(overrides: Partial<BridgeProvider> = {}): BridgeProvider {
  return {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-15",
    },
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    platform: "telegram",
    secret_slots: [
      {
        description: "Bot token",
        name: "bot_token",
        required: true,
      },
    ],
    state: "active",
    ...overrides,
  };
}

describe("BridgeDetailPanel", () => {
  it("renders loading, error, and empty states", () => {
    const { rerender } = render(
      <BridgeDetailPanel
        bridge={undefined}
        error={null}
        health={undefined}
        isLoading
        isRoutesLoading={false}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-detail-loading")).toBeInTheDocument();

    rerender(
      <BridgeDetailPanel
        bridge={undefined}
        error={new Error("boom")}
        health={undefined}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-detail-error")).toHaveTextContent("boom");

    rerender(
      <BridgeDetailPanel
        bridge={undefined}
        error={null}
        health={undefined}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-detail-empty")).toBeInTheDocument();
  });

  it("renders provider-runtime fallbacks when metadata is absent", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge({ dm_policy: undefined, provider_config: undefined })}
        error={null}
        health={makeHealth()}
        isLoading={false}
        isRoutesLoading={false}
        onOpenTestDelivery={vi.fn()}
        provider={makeProvider({ config_schema: undefined, secret_slots: undefined })}
        routes={[] satisfies BridgeRoute[]}
      />
    );

    expect(screen.getByText("Provider default")).toBeInTheDocument();
    expect(screen.getByText("No structured config schema published")).toBeInTheDocument();
    expect(
      screen.getByText("No provider runtime config stored for this bridge.")
    ).toBeInTheDocument();
  });
});
