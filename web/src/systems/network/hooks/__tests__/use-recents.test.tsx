// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { useNetworkRecents } from "../use-recents";

vi.mock("../../adapters/network-api", () => ({
  listNetworkThreads: vi.fn(async (channel: string) => {
    if (channel === "ops") {
      return [
        {
          channel: "ops",
          last_activity_at: "2026-04-17T18:16:00Z",
          last_message_preview: "thread preview ops",
          message_count: 6,
          open_work_count: 0,
          opened_at: "2026-04-17T17:50:00Z",
          opened_by_peer_id: "peer-a",
          opened_session_id: "sess-a",
          participant_count: 4,
          root_message_id: "msg_root_ops",
          thread_id: "thread_ops_one",
          title: "Ops thread",
        },
      ];
    }
    if (channel === "design") {
      return [
        {
          channel: "design",
          last_activity_at: "2026-04-17T17:00:00Z",
          last_message_preview: "design thread preview",
          message_count: 3,
          open_work_count: 0,
          opened_at: "2026-04-17T16:00:00Z",
          opened_by_peer_id: "peer-b",
          opened_session_id: "sess-b",
          participant_count: 2,
          root_message_id: "msg_root_design",
          thread_id: "thread_design_one",
        },
      ];
    }
    return [];
  }),
  listNetworkDirectRooms: vi.fn(async (channel: string) => {
    if (channel === "ops") {
      return [
        {
          channel: "ops",
          direct_id: "direct_ops_one",
          last_activity_at: "2026-04-17T18:30:00Z",
          last_message_preview: "ops dm preview",
          message_count: 2,
          open_work_count: 0,
          opened_at: "2026-04-17T18:00:00Z",
          peer_a: "peer-a",
          peer_b: "peer-b",
        },
        {
          channel: "ops",
          direct_id: "direct_ops_two",
          last_activity_at: "2026-04-17T15:00:00Z",
          last_message_preview: "ops dm older",
          message_count: 1,
          open_work_count: 0,
          opened_at: "2026-04-17T14:00:00Z",
          peer_a: "peer-c",
          peer_b: "peer-d",
        },
      ];
    }
    if (channel === "design") {
      return [
        {
          channel: "design",
          direct_id: "direct_design_one",
          last_activity_at: "2026-04-17T16:30:00Z",
          last_message_preview: "design dm preview",
          message_count: 2,
          open_work_count: 0,
          opened_at: "2026-04-17T16:00:00Z",
          peer_a: "peer-x",
          peer_b: "peer-y",
        },
      ];
    }
    return [];
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

const channels = [
  { channel: "ops", workspace_id: "w1", created_at: "", created_by: "", peer_count: 2 },
  { channel: "design", workspace_id: "w1", created_at: "", created_by: "", peer_count: 2 },
];

describe("useNetworkRecents", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("merges thread + direct summaries across channels and caps at the limit", async () => {
    const { result } = renderHook(() => useNetworkRecents(channels, { limit: 5 }), {
      wrapper: makeWrapper(),
    });

    await waitFor(() => {
      expect(result.current.recents.length).toBeGreaterThan(0);
    });
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.recents.length).toBeLessThanOrEqual(5);
    expect(result.current.recents.map(entry => entry.containerId)).toEqual([
      "direct_ops_one",
      "thread_ops_one",
      "thread_design_one",
      "direct_design_one",
      "direct_ops_two",
    ]);
  });

  it("caps at the supplied max and returns at most N entries even with more available", async () => {
    const { result } = renderHook(() => useNetworkRecents(channels, { limit: 2 }), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => {
      expect(result.current.recents.length).toBeGreaterThan(0);
    });
    expect(result.current.recents.length).toBe(2);
  });

  it("flags unread when last_activity is fresher than the stored last-read marker", async () => {
    window.localStorage.setItem(
      "network:last-read",
      JSON.stringify({
        "ops:thread:thread_ops_one": "2026-04-17T18:00:00Z",
      })
    );
    const { result } = renderHook(() => useNetworkRecents(channels, { limit: 5 }), {
      wrapper: makeWrapper(),
    });
    await waitFor(() => {
      const opsThread = result.current.recents.find(
        entry => entry.containerId === "thread_ops_one"
      );
      expect(opsThread).toBeDefined();
      // last_activity 18:16 > stored 18:00 → unread true
      expect(opsThread?.hasUnread).toBe(true);
    });
  });
});
