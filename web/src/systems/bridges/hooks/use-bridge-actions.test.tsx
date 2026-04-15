import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useCreateBridge, useTestBridgeDelivery } from "@/systems/bridges/hooks/use-bridge-actions";

vi.mock("@/systems/bridges/adapters/bridges-api", () => ({
  createBridge: vi.fn(),
  getBridge: vi.fn(),
  listBridgeProviders: vi.fn(),
  listBridgeRoutes: vi.fn(),
  listBridges: vi.fn(),
  testBridgeDelivery: vi.fn(),
}));

import { createBridge, testBridgeDelivery } from "@/systems/bridges/adapters/bridges-api";

describe("useCreateBridge", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("creates a bridge and invalidates the bridge root query", async () => {
    vi.mocked(createBridge).mockResolvedValue({
      bridge: {
        created_at: "2026-04-13T12:00:00Z",
        display_name: "Support",
        enabled: true,
        extension_name: "ext-telegram",
        id: "brg_support",
        platform: "telegram",
        routing_policy: { include_group: true, include_peer: true, include_thread: true },
        scope: "workspace",
        status: "starting",
        updated_at: "2026-04-13T12:00:00Z",
        workspace_id: "ws_test",
      },
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

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);

    const { result } = renderHook(() => useCreateBridge(), { wrapper });

    act(() => {
      result.current.mutate({
        dm_policy: "allowlist",
        display_name: "Support",
        enabled: true,
        extension_name: "ext-telegram",
        platform: "telegram",
        provider_config: {
          mode: "bot",
        },
        routing_policy: { include_group: true, include_peer: true, include_thread: true },
        scope: "workspace",
        status: "starting",
        workspace_id: "ws_test",
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(createBridge).toHaveBeenCalledWith({
      dm_policy: "allowlist",
      display_name: "Support",
      enabled: true,
      extension_name: "ext-telegram",
      platform: "telegram",
      provider_config: {
        mode: "bot",
      },
      routing_policy: { include_group: true, include_peer: true, include_thread: true },
      scope: "workspace",
      status: "starting",
      workspace_id: "ws_test",
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["bridges"],
    });
  });
});

describe("useTestBridgeDelivery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("tests delivery and invalidates detail plus routes for the selected bridge", async () => {
    vi.mocked(testBridgeDelivery).mockResolvedValue({
      delivery_target: {
        bridge_instance_id: "brg_support",
        mode: "reply",
        peer_id: "peer_123",
      },
      status: "resolved",
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);

    const { result } = renderHook(() => useTestBridgeDelivery(), { wrapper });

    act(() => {
      result.current.mutate({
        data: {
          target: {
            bridge_instance_id: "brg_support",
            peer_id: "peer_123",
          },
        },
        id: "brg_support",
      });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(testBridgeDelivery).toHaveBeenCalledWith("brg_support", {
      target: {
        bridge_instance_id: "brg_support",
        peer_id: "peer_123",
      },
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["bridges"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["bridges", "detail", "brg_support"],
    });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["bridges", "routes", "brg_support"],
    });
  });
});
