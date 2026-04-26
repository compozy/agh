import { QueryClient, type QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listNetworkChannelMessages: vi.fn().mockResolvedValue([]),
  listNetworkPeerMessages: vi.fn().mockResolvedValue([]),
}));

vi.mock("../adapters/network-api", async () => {
  const actual = await vi.importActual("../adapters/network-api");

  return {
    ...actual,
    listNetworkChannelMessages: mocks.listNetworkChannelMessages,
    listNetworkPeerMessages: mocks.listNetworkPeerMessages,
  };
});

import { networkChannelMessagesOptions, networkPeerMessagesOptions } from "./query-options";

function makeQueryContext<TQueryKey extends readonly unknown[]>(queryKey: TQueryKey) {
  return {
    client: new QueryClient(),
    meta: undefined,
    queryKey,
    signal: new AbortController().signal,
  } satisfies QueryFunctionContext<TQueryKey>;
}

function requireQueryFn<TQueryKey extends readonly unknown[]>(
  queryFn: ((context: QueryFunctionContext<TQueryKey>) => unknown) | undefined
) {
  if (!queryFn) {
    throw new Error("Expected queryFn to be defined");
  }

  return queryFn;
}

describe("network query options", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("normalizes channel timeline defaults for both the query key and fetch payload", async () => {
    const withoutQuery = networkChannelMessagesOptions("storybook");
    const emptyQuery = networkChannelMessagesOptions("storybook", {});
    const undefinedLimit = networkChannelMessagesOptions("storybook", { limit: undefined });

    expect(withoutQuery.queryKey).toEqual(emptyQuery.queryKey);
    expect(emptyQuery.queryKey).toEqual(undefinedLimit.queryKey);
    expect(withoutQuery.queryKey).toEqual([
      "network",
      "channels",
      "messages",
      "storybook",
      "",
      "",
      0,
      120,
    ]);

    await requireQueryFn(withoutQuery.queryFn)(makeQueryContext(withoutQuery.queryKey));
    await requireQueryFn(emptyQuery.queryFn)(makeQueryContext(emptyQuery.queryKey));
    await requireQueryFn(undefinedLimit.queryFn)(makeQueryContext(undefinedLimit.queryKey));

    expect(mocks.listNetworkChannelMessages).toHaveBeenNthCalledWith(
      1,
      "storybook",
      { limit: 120 },
      expect.any(AbortSignal)
    );
    expect(mocks.listNetworkChannelMessages).toHaveBeenNthCalledWith(
      2,
      "storybook",
      { limit: 120 },
      expect.any(AbortSignal)
    );
    expect(mocks.listNetworkChannelMessages).toHaveBeenNthCalledWith(
      3,
      "storybook",
      { limit: 120 },
      expect.any(AbortSignal)
    );
  });

  it("normalizes peer timeline defaults without dropping explicit cursors", async () => {
    const withCursor = networkPeerMessagesOptions("peer_storybook_remote", {
      after: "cursor_123",
      limit: undefined,
    });
    const normalized = networkPeerMessagesOptions("peer_storybook_remote", {
      after: "cursor_123",
    });

    expect(withCursor.queryKey).toEqual(normalized.queryKey);
    expect(withCursor.queryKey).toEqual([
      "network",
      "peers",
      "messages",
      "peer_storybook_remote",
      "",
      "cursor_123",
      0,
      120,
    ]);

    await requireQueryFn(withCursor.queryFn)(makeQueryContext(withCursor.queryKey));

    expect(mocks.listNetworkPeerMessages).toHaveBeenCalledWith(
      "peer_storybook_remote",
      { after: "cursor_123", limit: 120 },
      expect.any(AbortSignal)
    );
  });

  it("keeps presence toggles isolated in both the query key and fetch payload", async () => {
    const hiddenPresence = networkChannelMessagesOptions("storybook", { limit: 20 });
    const shownPresence = networkChannelMessagesOptions("storybook", {
      include_presence: true,
      limit: 20,
    });

    expect(hiddenPresence.queryKey).not.toEqual(shownPresence.queryKey);
    expect(shownPresence.queryKey).toEqual([
      "network",
      "channels",
      "messages",
      "storybook",
      "",
      "",
      1,
      20,
    ]);

    await requireQueryFn(shownPresence.queryFn)(makeQueryContext(shownPresence.queryKey));

    expect(mocks.listNetworkChannelMessages).toHaveBeenCalledWith(
      "storybook",
      { include_presence: true, limit: 20 },
      expect.any(AbortSignal)
    );
  });
});
