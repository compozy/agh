// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { PINNED_CHANNELS_STORAGE_KEY_FOR_TESTS, useNetworkChannels } from "../use-channels";

vi.mock("../../adapters/network-api", () => ({
  listNetworkChannels: vi.fn().mockResolvedValue({
    channels: [
      { channel: "ops", workspace_id: "w1", created_at: "2026-04-17T14:00:00Z", created_by: "ops" },
      {
        channel: "alpha",
        workspace_id: "w1",
        created_at: "2026-04-17T14:00:00Z",
        created_by: "ops",
      },
      {
        channel: "design",
        workspace_id: "w1",
        created_at: "2026-04-17T14:00:00Z",
        created_by: "ops",
      },
    ],
  }),
}));

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe("useNetworkChannels", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("sorts channels alphabetically and surfaces pinned channels separately", async () => {
    const { result } = renderHook(() => useNetworkChannels({ enabled: true }), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => {
      expect(result.current.channels.map(channel => channel.channel)).toEqual([
        "alpha",
        "design",
        "ops",
      ]);
    });

    expect(result.current.pinned).toEqual([]);
    expect(result.current.unpinned.map(channel => channel.channel)).toEqual([
      "alpha",
      "design",
      "ops",
    ]);
  });

  it("toggles pinned channels through localStorage", async () => {
    const { result } = renderHook(() => useNetworkChannels({ enabled: true }), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => {
      expect(result.current.channels.length).toBe(3);
    });

    act(() => {
      result.current.togglePinned("ops");
    });

    await waitFor(() => {
      expect(result.current.isPinned("ops")).toBe(true);
    });
    expect(result.current.pinned.map(channel => channel.channel)).toEqual(["ops"]);

    const stored = JSON.parse(
      window.localStorage.getItem(PINNED_CHANNELS_STORAGE_KEY_FOR_TESTS) ?? "[]"
    ) as string[];
    expect(stored).toEqual(["ops"]);

    act(() => {
      result.current.togglePinned("ops");
    });
    await waitFor(() => {
      expect(result.current.isPinned("ops")).toBe(false);
    });
  });
});
