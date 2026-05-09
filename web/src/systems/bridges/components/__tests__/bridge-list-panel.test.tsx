import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BridgeListPanel } from "@/systems/bridges/components/bridge-list-panel";
import type { BridgeHealthMap, BridgeSummary } from "@/systems/bridges/types";

function makeBridge(overrides: Partial<BridgeSummary> = {}): BridgeSummary {
  return {
    created_at: "2026-04-13T12:00:00Z",
    delivery_defaults: {},
    display_name: "Support",
    dm_policy: "open",
    enabled: true,
    extension_name: "ext-telegram",
    id: "brg_support",
    platform: "telegram",
    provider_config: {},
    routing_policy: { include_group: true, include_peer: true, include_thread: true },
    scope: "workspace",
    status: "ready",
    updated_at: "2026-04-13T12:30:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeHealthMap(entries: Record<string, BridgeHealthMap[string]>): BridgeHealthMap {
  return entries;
}

describe("BridgeListPanel", () => {
  it("groups bridges by provider and renders one row per bridge", () => {
    const bridges = [
      makeBridge({ id: "brg_support", display_name: "Support", platform: "telegram" }),
      makeBridge({
        id: "brg_ops_email",
        display_name: "Ops email",
        extension_name: "ext-email",
        platform: "email",
      }),
      makeBridge({
        id: "brg_ops_slack",
        display_name: "Ops slack",
        extension_name: "ext-slack",
        platform: "slack",
      }),
    ];

    render(
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={bridges}
        onSearchChange={vi.fn()}
        onSelectBridge={vi.fn()}
        searchQuery=""
        selectedBridgeId="brg_support"
        summary="3 bridges visible"
      />
    );

    expect(screen.getByTestId("bridge-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-item-brg_support")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-item-brg_ops_email")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-item-brg_ops_slack")).toBeInTheDocument();

    const groupHeaders = screen.getAllByTestId(/^bridge-list-group-header-/);
    expect(groupHeaders).toHaveLength(3);
    expect(
      screen.getByTestId("bridge-list-group-header-ext-telegram-telegram")
    ).toBeInTheDocument();
    expect(screen.getByTestId("bridge-list-group-header-ext-email-email")).toBeInTheDocument();

    expect(screen.getByTestId("bridge-item-brg_support")).toHaveAttribute("aria-pressed", "true");
    expect(
      screen
        .getByTestId("bridge-item-brg_support")
        .querySelector('[data-slot="item-selection-indicator"][data-indicator="rail"]')
    ).toBeInTheDocument();
  });

  it("renders the filtered-empty state with no-results copy when search has no matches", () => {
    render(
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        onSearchChange={vi.fn()}
        onSelectBridge={vi.fn()}
        searchQuery="zzzz"
        selectedBridgeId={null}
        summary="0 bridges visible"
      />
    );

    const empty = screen.getByTestId("bridge-list-empty");
    expect(empty).toBeInTheDocument();
    expect(within(empty).getByText(/Try a different search term/i)).toBeInTheDocument();
  });

  it("renders the default empty state when no bridges and no search term", () => {
    render(
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        onSearchChange={vi.fn()}
        onSelectBridge={vi.fn()}
        searchQuery=""
        selectedBridgeId={null}
        summary="0 bridges visible"
      />
    );

    const empty = screen.getByTestId("bridge-list-empty");
    expect(within(empty).getByText(/No bridges match the current filters/i)).toBeInTheDocument();
  });

  it("renders loading and error fallbacks", () => {
    const { rerender } = render(
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        isLoading
        onSearchChange={vi.fn()}
        onSelectBridge={vi.fn()}
        searchQuery=""
        selectedBridgeId={null}
        summary=""
      />
    );

    expect(screen.getByTestId("bridge-list-loading")).toBeInTheDocument();

    rerender(
      <BridgeListPanel
        bridgeHealth={{}}
        bridges={[]}
        errorMessage="boom"
        onSearchChange={vi.fn()}
        onSelectBridge={vi.fn()}
        searchQuery=""
        selectedBridgeId={null}
        summary=""
      />
    );

    expect(screen.getByTestId("bridge-list-error")).toHaveTextContent("boom");
  });

  it("calls onSelectBridge when a row is clicked", async () => {
    const user = userEvent.setup();
    const onSelectBridge = vi.fn();
    const onSearchChange = vi.fn();

    render(
      <BridgeListPanel
        bridgeHealth={makeHealthMap({
          brg_support: {
            auth_failures_total: 0,
            bridge_instance_id: "brg_support",
            delivery_backlog: 1,
            delivery_dropped_total: 0,
            delivery_failures_total: 0,
            last_success_at: "2026-04-13T12:20:00Z",
            route_count: 2,
            status: "ready",
          },
        })}
        bridges={[makeBridge({ id: "brg_support" })]}
        onSearchChange={onSearchChange}
        onSelectBridge={onSelectBridge}
        searchQuery=""
        selectedBridgeId={null}
        summary="1 bridges visible"
      />
    );

    await user.click(screen.getByTestId("bridge-item-brg_support"));
    expect(onSelectBridge).toHaveBeenCalledWith("brg_support");

    fireEvent.change(screen.getByTestId("bridge-search-input"), {
      target: { value: "support" },
    });
    expect(onSearchChange).toHaveBeenCalledWith("support");

    expect(screen.getByText(/2 routes/i)).toBeInTheDocument();
  });
});
