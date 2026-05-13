import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  useBridge,
  useBridgeProviders,
  useBridgeRoutes,
  useBridgeSecretBindings,
  useBridges,
} from "@/systems/bridges/hooks/use-bridges";

vi.mock("@/systems/bridges/adapters/bridges-api", () => ({
  createBridge: vi.fn(),
  deleteBridgeSecretBinding: vi.fn(),
  disableBridge: vi.fn(),
  enableBridge: vi.fn(),
  getBridge: vi.fn(),
  listBridgeSecretBindings: vi.fn(),
  listBridgeProviders: vi.fn(),
  listBridgeRoutes: vi.fn(),
  listBridges: vi.fn(),
  putBridgeSecretBinding: vi.fn(),
  restartBridge: vi.fn(),
  testBridgeDelivery: vi.fn(),
  updateBridge: vi.fn(),
}));

import {
  getBridge,
  listBridgeSecretBindings,
  listBridgeProviders,
  listBridgeRoutes,
  listBridges,
} from "@/systems/bridges/adapters/bridges-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useBridges", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads the bridges list", async () => {
    vi.mocked(listBridges).mockResolvedValue({
      bridges: [],
    });

    const { result } = renderHook(() => useBridges(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.bridges).toEqual([]);
    });

    expect(listBridges).toHaveBeenCalledWith({}, expect.any(AbortSignal));
  });

  it("passes bridge list filters to the adapter", async () => {
    vi.mocked(listBridges).mockResolvedValue({
      bridges: [],
    });

    const { result } = renderHook(() => useBridges({ scope: "all", workspace_id: "ws_alpha" }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.bridges).toEqual([]);
    });

    expect(listBridges).toHaveBeenCalledWith(
      { scope: "all", workspace_id: "ws_alpha" },
      expect.any(AbortSignal)
    );
  });
});

describe("useBridgeProviders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads installed bridge providers", async () => {
    vi.mocked(listBridgeProviders).mockResolvedValue([
      {
        display_name: "Telegram",
        enabled: true,
        extension_name: "ext-telegram",
        health: "healthy",
        platform: "telegram",
        state: "active",
      },
    ]);

    const { result } = renderHook(() => useBridgeProviders(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listBridgeProviders).toHaveBeenCalledWith(expect.any(AbortSignal));
  });
});

describe("useBridge", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads a selected bridge detail", async () => {
    vi.mocked(getBridge).mockResolvedValue({
      bridge: {
        created_at: "2026-04-13T12:00:00Z",
        display_name: "Support",
        enabled: true,
        extension_name: "ext-telegram",
        id: "brg_support",
        platform: "telegram",
        routing_policy: { include_group: true, include_peer: true, include_thread: true },
        scope: "workspace",
        status: "ready",
        updated_at: "2026-04-13T12:30:00Z",
        workspace_id: "ws_test",
      },
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

    const { result } = renderHook(() => useBridge("brg_support"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.bridge.id).toBe("brg_support");
    });

    expect(getBridge).toHaveBeenCalledWith("brg_support", expect.any(AbortSignal));
  });

  it("does not fetch when bridge id is empty", () => {
    renderHook(() => useBridge(""), {
      wrapper: createWrapper(),
    });

    expect(getBridge).not.toHaveBeenCalled();
  });
});

describe("useBridgeRoutes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads routes for the selected bridge", async () => {
    vi.mocked(listBridgeRoutes).mockResolvedValue([
      {
        agent_name: "support-agent",
        bridge_instance_id: "brg_support",
        created_at: "2026-04-13T12:00:00Z",
        last_activity_at: "2026-04-13T12:10:00Z",
        peer_id: "peer_123",
        routing_key_hash: "abc123",
        scope: "workspace",
        session_id: "sess_123",
        updated_at: "2026-04-13T12:10:00Z",
        workspace_id: "ws_test",
      },
    ]);

    const { result } = renderHook(() => useBridgeRoutes("brg_support"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listBridgeRoutes).toHaveBeenCalledWith("brg_support", expect.any(AbortSignal));
  });
});

describe("useBridgeSecretBindings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads secret bindings for the selected bridge", async () => {
    vi.mocked(listBridgeSecretBindings).mockResolvedValue([
      {
        binding_name: "bot_token",
        bridge_instance_id: "brg_support",
        created_at: "2026-04-13T12:00:00Z",
        kind: "bot_token",
        updated_at: "2026-04-13T12:10:00Z",
        secret_ref: "vault:bridges/brg_support/bot_token",
      },
    ]);

    const { result } = renderHook(() => useBridgeSecretBindings("brg_support"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(listBridgeSecretBindings).toHaveBeenCalledWith("brg_support", expect.any(AbortSignal));
  });
});
