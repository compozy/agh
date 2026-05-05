// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";

import type { NetworkConversationMessage } from "../types";

const useNetworkMessagesMock = vi.fn();

vi.mock("./use-messages", () => ({
  useNetworkMessages: (...args: unknown[]) => useNetworkMessagesMock(...args),
}));

import { useOpenWork } from "./use-work";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

function buildMessage(overrides: Partial<NetworkConversationMessage>): NetworkConversationMessage {
  return {
    body: { text: "Hi" },
    channel: "ops",
    direction: "sent",
    display_name: "Codex",
    kind: "say",
    message_id: "msg",
    peer_from: "peer-self",
    timestamp: "2026-04-17T18:00:00Z",
    ...overrides,
  } as NetworkConversationMessage;
}

describe("useOpenWork", () => {
  it("Should aggregate work entries from messages, ignoring terminal states", () => {
    useNetworkMessagesMock.mockReturnValue({
      messages: [
        buildMessage({
          message_id: "m1",
          work_id: "work-a",
          body: { state: "working" },
          timestamp: "2026-04-17T18:00:00Z",
        }),
        buildMessage({
          message_id: "m2",
          work_id: "work-a",
          body: { state: "needs_input" },
          peer_to: "peer-remote",
          timestamp: "2026-04-17T18:01:00Z",
        }),
        buildMessage({
          message_id: "m3",
          work_id: "work-b",
          body: { state: "completed" },
          timestamp: "2026-04-17T18:02:00Z",
        }),
        buildMessage({
          message_id: "m4",
          work_id: "work-c",
          body: { state: "working" },
          timestamp: "2026-04-17T18:03:00Z",
        }),
      ],
      isLoading: false,
      isFetching: false,
      error: null,
    });

    const { result } = renderHook(
      () => useOpenWork({ channel: "ops", surface: "thread", containerId: "thread-1" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.openCount).toBe(2);
    expect(result.current.entries.map(entry => entry.workId)).toEqual(["work-a", "work-c"]);
    expect(result.current.hasNeedsInput).toBe(true);
    const workA = result.current.entries.find(entry => entry.workId === "work-a");
    expect(workA?.state).toBe("needs_input");
    expect(workA?.targetPeerId).toBe("peer-remote");
  });

  it("Should return empty when disabled", () => {
    useNetworkMessagesMock.mockReturnValue({
      messages: [],
      isLoading: false,
      isFetching: false,
      error: null,
    });
    const { result } = renderHook(
      () =>
        useOpenWork({ channel: "ops", surface: "thread", containerId: "thread-1", enabled: false }),
      { wrapper: createWrapper() }
    );
    expect(result.current.openCount).toBe(0);
    expect(result.current.entries).toEqual([]);
    expect(result.current.hasNeedsInput).toBe(false);
  });
});
