import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  applyBridgeHealthSnapshot,
  useBridgeHealthStream,
} from "@/systems/bridges/hooks/use-bridge-health-stream";

class FakeEventSource {
  public close = vi.fn();
  public onerror: ((event: Event) => void) | null = null;
  private readonly listeners = new Map<string, EventListenerOrEventListenerObject[]>();

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const current = this.listeners.get(type) ?? [];
    current.push(listener);
    this.listeners.set(type, current);
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const current = this.listeners.get(type) ?? [];
    this.listeners.set(
      type,
      current.filter(candidate => candidate !== listener)
    );
  }

  emit(type: string, data: unknown) {
    const event = new MessageEvent(type, {
      data: JSON.stringify(data),
    });

    for (const listener of this.listeners.get(type) ?? []) {
      if (typeof listener === "function") {
        listener(event);
        continue;
      }
      listener.handleEvent(event);
    }
  }
}

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("applyBridgeHealthSnapshot", () => {
  it("updates the bridges list and matching detail query", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    queryClient.setQueryData(["bridges", "list", "all", "ws_test", ""], {
      bridge_health: {},
      bridges: [
        {
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
      ],
    });
    queryClient.setQueryData(["bridges", "detail", "brg_support"], {
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

    applyBridgeHealthSnapshot(queryClient, {
      bridge_health: {
        brg_other: {
          auth_failures_total: 0,
          bridge_instance_id: "brg_other",
          delivery_backlog: 9,
          delivery_dropped_total: 0,
          delivery_failures_total: 0,
          route_count: 1,
          status: "ready",
        },
        brg_support: {
          auth_failures_total: 1,
          bridge_instance_id: "brg_support",
          delivery_backlog: 2,
          delivery_dropped_total: 0,
          delivery_failures_total: 0,
          route_count: 3,
          status: "ready",
        },
      },
      generated_at: "2026-04-15T12:00:00Z",
    });

    expect(
      queryClient.getQueryData<{
        bridge_health: Record<string, { status: string }>;
      }>(["bridges", "list", "all", "ws_test", ""])?.bridge_health.brg_support.status
    ).toBe("ready");
    expect(
      queryClient.getQueryData<{
        bridge_health: Record<string, { status: string }>;
      }>(["bridges", "list", "all", "ws_test", ""])?.bridge_health.brg_other
    ).toBeUndefined();
    expect(
      queryClient.getQueryData<{
        health: { route_count: number; status: string };
      }>(["bridges", "detail", "brg_support"])?.health
    ).toEqual(
      expect.objectContaining({
        route_count: 3,
        status: "ready",
      })
    );
  });

  it("invalidates cached bridge routes when the live route count changes", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    queryClient.setQueryData(["bridges", "routes", "brg_support"], []);

    applyBridgeHealthSnapshot(queryClient, {
      bridge_health: {
        brg_support: {
          auth_failures_total: 0,
          bridge_instance_id: "brg_support",
          delivery_backlog: 0,
          delivery_dropped_total: 0,
          delivery_failures_total: 0,
          route_count: 1,
          status: "ready",
        },
      },
      generated_at: "2026-04-15T12:00:00Z",
    });

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["bridges", "routes", "brg_support"],
    });
  });
});

describe("useBridgeHealthStream", () => {
  it("subscribes to the stream and closes the event source on unmount", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const eventSource = new FakeEventSource();
    const eventSourceFactory = vi.fn((_url: string) => eventSource);

    queryClient.setQueryData(["bridges", "list", "all", "ws_test", ""], {
      bridge_health: {},
      bridges: [
        {
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
      ],
    });

    const { unmount } = renderHook(
      () =>
        useBridgeHealthStream({
          eventSourceFactory,
          filters: { scope: "all", workspace_id: "ws_test" },
        }),
      {
        wrapper: createWrapper(queryClient),
      }
    );

    expect(eventSourceFactory).toHaveBeenCalledWith(
      "/api/bridges/health/stream?scope=all&workspace_id=ws_test"
    );

    act(() => {
      eventSource.emit("snapshot", {
        bridge_health: {
          brg_support: {
            auth_failures_total: 0,
            bridge_instance_id: "brg_support",
            delivery_backlog: 1,
            delivery_dropped_total: 0,
            delivery_failures_total: 0,
            route_count: 1,
            status: "ready",
          },
        },
        generated_at: "2026-04-15T12:00:00Z",
      });
    });

    expect(
      queryClient.getQueryData<{
        bridge_health: Record<string, { delivery_backlog: number }>;
      }>(["bridges", "list", "all", "ws_test", ""])?.bridge_health.brg_support.delivery_backlog
    ).toBe(1);

    unmount();

    expect(eventSource.close).toHaveBeenCalledTimes(1);
  });
});
