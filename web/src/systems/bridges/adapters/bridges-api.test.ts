import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";
import {
  BridgesApiError,
  createBridge,
  getBridge,
  listBridgeProviders,
  listBridgeRoutes,
  listBridges,
  testBridgeDelivery,
} from "@/systems/bridges/adapters/bridges-api";

const bridgeFixture = {
  created_at: "2026-04-13T12:00:00Z",
  display_name: "Support",
  enabled: true,
  extension_name: "ext-telegram",
  id: "brg_support",
  platform: "telegram",
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

  it("passes abort signal through to fetch", async () => {
    mockJsonResponse({ bridges: [] });

    const controller = new AbortController();
    await listBridges(controller.signal);

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
          display_name: "Telegram",
          enabled: true,
          extension_name: "ext-telegram",
          health: "healthy",
          platform: "telegram",
          state: "active",
        },
      ],
    });

    const result = await listBridgeProviders();

    expect(result).toEqual([
      expect.objectContaining({
        display_name: "Telegram",
        extension_name: "ext-telegram",
      }),
    ]);
    await expectFetchRequest({ path: "/api/bridges/providers" });
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
      display_name: "Support",
      enabled: true,
      extension_name: "ext-telegram",
      platform: "telegram",
      routing_policy: {
        include_group: true,
        include_peer: true,
        include_thread: true,
      },
      scope: "workspace" as const,
      status: "starting" as const,
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
});
