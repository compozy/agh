import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";
import {
  BridgesApiError,
  createBridge,
  deleteBridgeSecretBinding,
  disableBridge,
  enableBridge,
  getBridge,
  listBridgeSecretBindings,
  listBridgeProviders,
  listBridgeRoutes,
  listBridges,
  putBridgeSecretBinding,
  restartBridge,
  testBridgeDelivery,
  updateBridge,
} from "@/systems/bridges/adapters/bridges-api";

const bridgeFixture = {
  created_at: "2026-04-13T12:00:00Z",
  dm_policy: "open",
  display_name: "Support",
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
  scope: "workspace" as const,
  status: "ready" as const,
  updated_at: "2026-04-13T12:30:00Z",
  workspace_id: "ws_test",
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("listBridges", () => {
  it("calls GET /api/bridges and returns the typed payload", async () => {
    mockJsonResponse({
      bridge_health: {
        brg_support: {
          auth_failures_total: 0,
          bridge_instance_id: "brg_support",
          delivery_backlog: 1,
          delivery_dropped_total: 0,
          delivery_failures_total: 0,
          route_count: 2,
          status: "ready",
        },
      },
      bridges: [bridgeFixture],
    });

    const result = await listBridges();

    expect(result.bridges).toEqual([bridgeFixture]);
    await expectFetchRequest({ path: "/api/bridges" });
  });

  it("sends bridge list scope filters", async () => {
    mockJsonResponse({ bridges: [] });

    await listBridges({ scope: "all", workspace_id: " ws_alpha " });

    await expectFetchRequest({ path: "/api/bridges?scope=all&workspace_id=ws_alpha" });
  });

  it("passes abort signal through to fetch", async () => {
    mockJsonResponse({ bridges: [] });

    const controller = new AbortController();
    await listBridges({}, controller.signal);

    await expectFetchRequest({
      path: "/api/bridges",
      signal: controller.signal,
    });
  });

  it("throws BridgesApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(listBridges()).rejects.toThrow(BridgesApiError);
    await expect(listBridges()).rejects.toThrow("Failed to fetch bridges: 503");
  });
});

describe("listBridgeProviders", () => {
  it("calls GET /api/bridges/providers", async () => {
    mockJsonResponse({
      providers: [
        {
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
        },
      ],
    });

    const result = await listBridgeProviders();

    expect(result).toEqual([
      expect.objectContaining({
        display_name: "Telegram",
        extension_name: "ext-telegram",
        secret_slots: [
          {
            description: "Bot token",
            name: "bot_token",
            required: true,
          },
        ],
      }),
    ]);
    await expectFetchRequest({ path: "/api/bridges/providers" });
  });

  it("throws BridgesApiError when provider lookup fails", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(listBridgeProviders()).rejects.toThrow(BridgesApiError);
    await expect(listBridgeProviders()).rejects.toThrow("Failed to fetch bridge providers: 503");
  });
});

describe("getBridge", () => {
  it("calls GET /api/bridges/:id", async () => {
    mockJsonResponse({
      bridge: bridgeFixture,
      health: {
        auth_failures_total: 0,
        bridge_instance_id: "brg_support",
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        route_count: 1,
        status: "ready",
      },
    });

    const result = await getBridge("brg_support");

    expect(result.bridge).toEqual(bridgeFixture);
    await expectFetchRequest({ path: "/api/bridges/brg_support" });
  });

  it("throws a not found error for unknown bridges", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(getBridge("missing")).rejects.toThrow("Bridge not found: missing");
  });

  it("throws a typed error for non-404 bridge fetch failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(getBridge("brg_support")).rejects.toThrow(
      'Failed to load bridge "brg_support": 500'
    );
  });
});

describe("listBridgeRoutes", () => {
  it("calls GET /api/bridges/:id/routes", async () => {
    mockJsonResponse({
      routes: [
        {
          agent_name: "support-agent",
          bridge_instance_id: "brg_support",
          created_at: "2026-04-13T12:00:00Z",
          last_activity_at: "2026-04-13T12:15:00Z",
          peer_id: "peer_123",
          routing_key_hash: "abc123",
          scope: "workspace",
          session_id: "sess_123",
          updated_at: "2026-04-13T12:15:00Z",
          workspace_id: "ws_test",
        },
      ],
    });

    const result = await listBridgeRoutes("brg_support");

    expect(result).toHaveLength(1);
    await expectFetchRequest({ path: "/api/bridges/brg_support/routes" });
  });

  it("throws a not found error for missing route sets", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(listBridgeRoutes("missing")).rejects.toThrow("Bridge not found: missing");
  });
});

describe("createBridge", () => {
  it("calls POST /api/bridges with the create payload", async () => {
    mockJsonResponse(
      {
        bridge: bridgeFixture,
        health: {
          auth_failures_total: 0,
          bridge_instance_id: "brg_support",
          delivery_backlog: 0,
          delivery_dropped_total: 0,
          delivery_failures_total: 0,
          route_count: 0,
          status: "starting",
        },
      },
      { status: 201 }
    );

    const payload = {
      dm_policy: "open" as const,
      display_name: "Support",
      enabled: true,
      extension_name: "ext-telegram",
      platform: "telegram",
      provider_config: {
        mode: "bot",
      },
      routing_policy: {
        include_group: true,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace" as const,
      workspace_id: "ws_test",
    };

    const result = await createBridge(payload);

    expect(result.bridge).toEqual(bridgeFixture);
    await expectFetchRequest({
      body: payload,
      method: "POST",
      path: "/api/bridges",
    });
  });

  it("throws BridgesApiError when bridge creation fails", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 400 }));

    await expect(
      createBridge({
        display_name: "Support",
        enabled: true,
        extension_name: "ext-telegram",
        platform: "telegram",
        routing_policy: {
          include_group: true,
          include_peer: true,
          include_thread: true,
        },
        scope: "workspace",
        workspace_id: "ws_test",
      })
    ).rejects.toThrow("Failed to create bridge: 400");
  });
});

describe("updateBridge", () => {
  it("calls PATCH /api/bridges/:id with the update payload", async () => {
    mockJsonResponse({
      bridge: {
        ...bridgeFixture,
        display_name: "Support Ops",
      },
      health: {
        auth_failures_total: 0,
        bridge_instance_id: "brg_support",
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        route_count: 0,
        status: "ready",
      },
    });

    const payload = {
      delivery_defaults: {
        peer_id: "peer_default",
      },
      display_name: "Support Ops",
      dm_policy: "allowlist" as const,
      provider_config: {
        mode: "bot",
      },
      routing_policy: {
        include_group: true,
        include_peer: false,
        include_thread: true,
      },
    };

    const result = await updateBridge("brg_support", payload);

    expect(result.bridge.display_name).toBe("Support Ops");
    await expectFetchRequest({
      body: payload,
      method: "PATCH",
      path: "/api/bridges/brg_support",
    });
  });
});

describe("listBridgeSecretBindings", () => {
  it("calls GET /api/bridges/:id/secret-bindings", async () => {
    mockJsonResponse({
      bindings: [
        {
          binding_name: "bot_token",
          bridge_instance_id: "brg_support",
          created_at: "2026-04-13T12:00:00Z",
          kind: "bot_token",
          updated_at: "2026-04-13T12:00:00Z",
          secret_ref: "vault:bridges/brg_support/bot_token",
        },
      ],
    });

    const result = await listBridgeSecretBindings("brg_support");

    expect(result).toHaveLength(1);
    await expectFetchRequest({ path: "/api/bridges/brg_support/secret-bindings" });
  });
});

describe("putBridgeSecretBinding", () => {
  it("calls PUT /api/bridges/:id/secret-bindings/:binding_name", async () => {
    mockJsonResponse({
      binding: {
        binding_name: "bot_token",
        bridge_instance_id: "brg_support",
        created_at: "2026-04-13T12:00:00Z",
        kind: "bot_token",
        updated_at: "2026-04-13T12:30:00Z",
        secret_ref: "vault:bridges/brg_support/bot_token",
      },
    });

    const payload = {
      kind: "bot_token",
      secret_ref: "vault:bridges/brg_support/bot_token",
      secret_value: "telegram-token",
    };

    const result = await putBridgeSecretBinding("brg_support", "bot_token", payload);

    expect(result.secret_ref).toBe("vault:bridges/brg_support/bot_token");
    await expectFetchRequest({
      body: payload,
      method: "PUT",
      path: "/api/bridges/brg_support/secret-bindings/bot_token",
    });
  });
});

describe("deleteBridgeSecretBinding", () => {
  it("calls DELETE /api/bridges/:id/secret-bindings/:binding_name", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 204 }));

    await deleteBridgeSecretBinding("brg_support", "bot_token");

    await expectFetchRequest({
      method: "DELETE",
      path: "/api/bridges/brg_support/secret-bindings/bot_token",
    });
  });
});

describe("bridge lifecycle", () => {
  it("calls the enable, disable, and restart endpoints", async () => {
    mockJsonResponse({
      bridge: bridgeFixture,
      health: {
        auth_failures_total: 0,
        bridge_instance_id: "brg_support",
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        route_count: 0,
        status: "starting",
      },
    });
    await enableBridge("brg_support");
    await expectFetchRequest({
      callIndex: 0,
      method: "POST",
      path: "/api/bridges/brg_support/enable",
    });

    mockJsonResponse({
      bridge: bridgeFixture,
      health: {
        auth_failures_total: 0,
        bridge_instance_id: "brg_support",
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        route_count: 0,
        status: "disabled",
      },
    });
    await disableBridge("brg_support");
    await expectFetchRequest({
      callIndex: 1,
      method: "POST",
      path: "/api/bridges/brg_support/disable",
    });

    mockJsonResponse({
      bridge: bridgeFixture,
      health: {
        auth_failures_total: 0,
        bridge_instance_id: "brg_support",
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        route_count: 0,
        status: "starting",
      },
    });
    await restartBridge("brg_support");
    await expectFetchRequest({
      callIndex: 2,
      method: "POST",
      path: "/api/bridges/brg_support/restart",
    });
  });
});

describe("testBridgeDelivery", () => {
  it("calls POST /api/bridges/:id/test-delivery with the typed target payload", async () => {
    mockJsonResponse({
      delivery_target: {
        bridge_instance_id: "brg_support",
        mode: "reply",
        peer_id: "peer_123",
      },
      message: "Ping",
      status: "resolved",
    });

    const payload = {
      message: "Ping",
      target: {
        bridge_instance_id: "brg_support",
        mode: "reply" as const,
        peer_id: "peer_123",
      },
    };

    const result = await testBridgeDelivery("brg_support", payload);

    expect(result.status).toBe("resolved");
    await expectFetchRequest({
      body: payload,
      method: "POST",
      path: "/api/bridges/brg_support/test-delivery",
    });
  });

  it("throws a bridge unavailable error on 409", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 409 }));

    await expect(
      testBridgeDelivery("brg_support", {
        target: {
          bridge_instance_id: "brg_support",
          peer_id: "peer_123",
        },
      })
    ).rejects.toThrow('Bridge "brg_support" is unavailable: 409');
  });

  it("throws a generic typed error for other delivery failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(
      testBridgeDelivery("brg_support", {
        target: {
          bridge_instance_id: "brg_support",
        },
      })
    ).rejects.toThrow('Failed to test delivery for bridge "brg_support": 500');
  });
});
