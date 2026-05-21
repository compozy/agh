import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type {
  BridgeHealth,
  BridgeProvider,
  BridgeResolveTargetResponse,
  BridgeRoute,
  BridgeSummary,
  BridgeTarget,
  BridgeTargetsResponse,
} from "@/systems/bridges";
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
    notification_suppress: false,
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

let nextRouteIndex = 1;

function makeRoute(overrides: Partial<BridgeRoute> = {}): BridgeRoute {
  const routeId = String(nextRouteIndex++).padStart(3, "0");

  return {
    agent_name: "support-agent",
    bridge_instance_id: "brg_support",
    created_at: "2026-04-13T12:00:00Z",
    last_activity_at: "2026-04-13T12:15:00Z",
    peer_id: "peer_123",
    routing_key_hash: `route_hash_${routeId}`,
    scope: "workspace",
    session_id: `sess_${routeId}`,
    updated_at: "2026-04-13T12:15:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeTarget(overrides: Partial<BridgeTarget> = {}): BridgeTarget {
  return {
    bridge_id: "brg_support",
    canonical_route: "telegram:channel:support",
    capabilities: ["direct-send", "reply"],
    display_name: "Support room",
    last_seen_at: "2026-04-13T12:20:00Z",
    normalized: "support room",
    qualifier: "telegram",
    target_type: "channel",
    updated_at: "2026-04-13T12:20:00Z",
    ...overrides,
  };
}

function makeTargetsResponse(targets: BridgeTarget[] = [makeTarget()]): BridgeTargetsResponse {
  return {
    bridge_id: "brg_support",
    cache_stale: false,
    generated_at: "2026-04-13T12:20:00Z",
    last_successful_refresh_at: "2026-04-13T12:20:00Z",
    targets,
    total: targets.length,
  };
}

describe("BridgeDetailPanel", () => {
  it("renders loading, error, and empty states", () => {
    const { rerender } = render(
      <BridgeDetailPanel
        bridge={undefined}
        error={null}
        health={undefined}
        state={{ isLoading: true, isRoutesLoading: false }}
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
        state={{ isLoading: false, isRoutesLoading: false }}
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
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-detail-empty")).toBeInTheDocument();
  });

  it("Should render the 24 px DetailHeader anatomy and surface the bridge display name as H1", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth()}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    const header = screen.getByTestId("bridge-detail-header");
    expect(header).toHaveAttribute("data-slot", "detail-header");
    const heading = within(header).getByRole("heading", { level: 1 });
    expect(heading).toHaveTextContent("Support");
  });

  it("renders provider-runtime fallbacks when metadata is absent", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge({ dm_policy: undefined, provider_config: undefined })}
        error={null}
        health={makeHealth()}
        state={{ isLoading: false, isRoutesLoading: false }}
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

  it("renders exactly four metric tiles with the required labels", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth({ delivery_backlog: 4, route_count: 2 })}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-metric-events-24h")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-metric-success-rate")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-metric-last-delivery")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-metric-active-routes")).toBeInTheDocument();

    const tiles = document.querySelectorAll('[data-slot="metric"]');
    expect(tiles).toHaveLength(4);
  });

  it("renders the Empty event stream when no routes are present", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth()}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("bridge-routes-empty")).toHaveTextContent("No routes");
  });

  it("renders bridge route session ids for operator traceability", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth({ route_count: 1 })}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[makeRoute({ session_id: "sess_trace_123" })]}
      />
    );

    const routeRow = screen.getByTestId("bridge-route-sess_trace_123");

    expect(routeRow).toHaveTextContent("sess_trace_123");
    expect(within(routeRow).getByText("Session")).toHaveAttribute("data-slot", "eyebrow");
  });

  it("renders target directory rows and submits resolve requests", async () => {
    const user = userEvent.setup();
    const onQueryChange = vi.fn();
    const onResolveInputChange = vi.fn();
    const onResolveSubmit = vi.fn();

    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth()}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
        targetDirectory={{
          error: null,
          isLoading: false,
          isResolving: false,
          onQueryChange,
          onResolveInputChange,
          onResolveSubmit,
          query: "",
          resolveInput: "Support room",
          resolveResult: null,
          response: makeTargetsResponse(),
        }}
      />
    );

    expect(screen.getByTestId("bridge-target-directory")).toHaveTextContent("Support room");
    expect(screen.getByTestId("bridge-target-telegram:channel:support")).toHaveTextContent(
      "telegram:channel:support"
    );

    await user.type(screen.getByTestId("bridge-target-search"), "ops");
    expect(onQueryChange).toHaveBeenCalled();

    await user.click(screen.getByTestId("bridge-target-resolve-submit"));
    expect(onResolveSubmit).toHaveBeenCalledTimes(1);
  });

  it("renders ambiguous target resolution candidates without choosing one", () => {
    const candidates = [
      makeTarget({ canonical_route: "telegram:channel:support", display_name: "Support room" }),
      makeTarget({ canonical_route: "telegram:channel:support-ops", display_name: "Support ops" }),
    ];
    const resolveResult: BridgeResolveTargetResponse = {
      diagnostic: {
        category: "bridge",
        code: "target_ambiguous",
        data_freshness: "live",
        id: "bridge_target_resolve:brg_support",
        message: 'Bridge target "support" matched 2 candidates',
        severity: "warn",
        title: "Bridge target is ambiguous",
      },
      result: {
        ambiguous: true,
        candidates,
        step: 4,
      },
    };

    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth()}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
        targetDirectory={{
          error: null,
          isLoading: false,
          isResolving: false,
          onQueryChange: vi.fn(),
          onResolveInputChange: vi.fn(),
          onResolveSubmit: vi.fn(),
          query: "",
          resolveInput: "support",
          resolveResult,
          response: makeTargetsResponse(candidates),
        }}
      />
    );

    expect(screen.getByTestId("bridge-target-resolve-ambiguous")).toHaveTextContent(
      "Bridge target is ambiguous"
    );
    expect(screen.getByTestId("bridge-target-resolve-ambiguous")).toHaveTextContent("2 candidates");
    expect(
      screen.getByTestId("bridge-target-resolve-candidate-telegram:channel:support")
    ).toHaveTextContent("Support room");
    expect(
      screen.getByTestId("bridge-target-resolve-candidate-telegram:channel:support-ops")
    ).toHaveTextContent("Support ops");
  });

  it("uses unique default route identities when rendering multiple route fixtures", () => {
    const routes = [makeRoute(), makeRoute()];

    expect(new Set(routes.map(route => `${route.session_id}:${route.routing_key_hash}`)).size).toBe(
      routes.length
    );

    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth({ route_count: routes.length })}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={routes}
      />
    );

    expect(screen.getByTestId(`bridge-route-${routes[0].session_id}`)).toBeInTheDocument();
    expect(screen.getByTestId(`bridge-route-${routes[1].session_id}`)).toBeInTheDocument();
  });

  it("renders disabled status with danger StatusDot and disables Send Test", () => {
    render(
      <BridgeDetailPanel
        bridge={makeBridge({ enabled: false, status: "disabled" })}
        error={null}
        health={makeHealth({ status: "disabled" })}
        state={{ isLoading: false, isRoutesLoading: false }}
        onOpenTestDelivery={vi.fn()}
        routes={[]}
      />
    );

    expect(screen.getByTestId("open-test-delivery-btn")).toBeDisabled();
    const dangerDot = document.querySelector('[data-slot="pill-dot"][data-tone="danger"]');
    expect(dangerDot).not.toBeNull();
  });

  it("renders lifecycle actions and secret binding controls", async () => {
    const user = userEvent.setup();
    const onOpenEdit = vi.fn();
    const onRestartBridge = vi.fn();
    const onDisableBridge = vi.fn();
    const onSaveSecretBinding = vi.fn();
    const onDeleteSecretBinding = vi.fn();
    const onSecretDraftChange = vi.fn();

    render(
      <BridgeDetailPanel
        bridge={makeBridge()}
        error={null}
        health={makeHealth({ status: "degraded" })}
        state={{ isLoading: false, isRoutesLoading: false }}
        onDeleteSecretBinding={onDeleteSecretBinding}
        onDisableBridge={onDisableBridge}
        onOpenEdit={onOpenEdit}
        onOpenTestDelivery={vi.fn()}
        onRestartBridge={onRestartBridge}
        onSaveSecretBinding={onSaveSecretBinding}
        onSecretDraftChange={onSecretDraftChange}
        provider={makeProvider()}
        restartRequired
        routes={[]}
        secretBindings={[
          {
            binding_name: "bot_token",
            bridge_instance_id: "brg_support",
            created_at: "2026-04-13T12:00:00Z",
            kind: "bot_token",
            updated_at: "2026-04-13T12:10:00Z",
            secret_ref: "vault:bridges/brg_support/bot_token",
          },
        ]}
        secretInputValues={{ bot_token: "telegram-token" }}
      />
    );

    expect(screen.getByTestId("bridge-restart-required")).toBeInTheDocument();
    expect(screen.getByTestId("disable-bridge-btn")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-secret-binding-bot_token")).toHaveTextContent("BOUND");

    await user.click(screen.getByTestId("edit-bridge-btn"));
    await user.click(screen.getByTestId("restart-bridge-btn"));
    await user.type(screen.getByTestId("bridge-secret-env-input-bot_token"), "X");
    await user.click(screen.getByTestId("save-bridge-secret-bot_token"));
    await user.click(screen.getByTestId("delete-bridge-secret-bot_token"));
    await user.click(screen.getByTestId("confirm-delete-bridge-secret-bot_token"));

    expect(onOpenEdit).toHaveBeenCalledTimes(1);
    expect(onRestartBridge).toHaveBeenCalledTimes(1);
    expect(onDisableBridge).not.toHaveBeenCalled();
    expect(onSecretDraftChange).toHaveBeenCalled();
    expect(onSaveSecretBinding).toHaveBeenCalledWith("bot_token");
    expect(onDeleteSecretBinding).toHaveBeenCalledWith("bot_token");
  });
});
