import { QueryClient, type QueryFunctionContext } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listNetworkThreadMessages: vi.fn().mockResolvedValue([]),
  listNetworkDirectRoomMessages: vi.fn().mockResolvedValue([]),
}));

vi.mock("../../adapters/network-api", async () => {
  const actual = await vi.importActual("../../adapters/network-api");

  return {
    ...actual,
    listNetworkThreadMessages: mocks.listNetworkThreadMessages,
    listNetworkDirectRoomMessages: mocks.listNetworkDirectRoomMessages,
  };
});

import { NetworkApiError } from "../../adapters/network-api";
import {
  networkDirectDetailOptions,
  networkDirectMessagesOptions,
  networkThreadDetailOptions,
  networkThreadMessagesOptions,
} from "../query-options";
import { networkKeys } from "../query-keys";

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

function requireRetry(
  retry: boolean | number | ((failureCount: number, error: Error) => boolean) | undefined
) {
  if (typeof retry !== "function") {
    throw new Error("Expected retry to be a function");
  }

  return retry;
}

describe("network query options — surface isolation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("threads tab queries are namespaced with channel + surface + thread id", async () => {
    const options = networkThreadMessagesOptions("builders", "thread_one");
    expect(options.queryKey).toEqual([
      "network",
      "channel",
      "builders",
      "thread",
      "messages",
      "thread_one",
      "",
      "",
      "",
      "",
      120,
    ]);
    await requireQueryFn(options.queryFn)(makeQueryContext(options.queryKey));
    expect(mocks.listNetworkThreadMessages).toHaveBeenCalledWith(
      "builders",
      "thread_one",
      { limit: 120 },
      expect.any(AbortSignal)
    );
  });

  it("directs tab queries are namespaced with channel + surface + direct id", async () => {
    const options = networkDirectMessagesOptions("builders", "direct_one");
    expect(options.queryKey).toEqual([
      "network",
      "channel",
      "builders",
      "direct",
      "messages",
      "direct_one",
      "",
      "",
      "",
      "",
      120,
    ]);
    await requireQueryFn(options.queryFn)(makeQueryContext(options.queryKey));
    expect(mocks.listNetworkDirectRoomMessages).toHaveBeenCalledWith(
      "builders",
      "direct_one",
      { limit: 120 },
      expect.any(AbortSignal)
    );
  });

  it("a directs query never shares the same key as a threads query in the same channel", () => {
    const threadsOpts = networkThreadMessagesOptions("builders", "shared_id");
    const directsOpts = networkDirectMessagesOptions("builders", "shared_id");
    expect(threadsOpts.queryKey).not.toEqual(directsOpts.queryKey);

    // Hierarchical key check using factories.
    expect(networkKeys.threadsList("builders").slice(0, 5)).toEqual([
      "network",
      "channel",
      "builders",
      "thread",
      "list",
    ]);
    expect(networkKeys.directsList("builders").slice(0, 5)).toEqual([
      "network",
      "channel",
      "builders",
      "direct",
      "list",
    ]);
  });

  it("a directs query never matches threads query under TanStack predicate", () => {
    const client = new QueryClient();
    const threadOpts = networkThreadMessagesOptions("builders", "container_x");
    const directOpts = networkDirectMessagesOptions("builders", "container_x");
    client.setQueryData(threadOpts.queryKey, [
      {
        message_id: "thread-msg",
        body: {},
        channel: "builders",
        direction: "sent",
        kind: "say",
        peer_from: "p",
        timestamp: "",
      },
    ]);
    client.setQueryData(directOpts.queryKey, [
      {
        message_id: "direct-msg",
        body: {},
        channel: "builders",
        direction: "sent",
        kind: "say",
        peer_from: "p",
        timestamp: "",
      },
    ]);

    const threadCacheValue = client.getQueryData(threadOpts.queryKey) as Array<{
      message_id: string;
    }>;
    const directCacheValue = client.getQueryData(directOpts.queryKey) as Array<{
      message_id: string;
    }>;
    expect(threadCacheValue?.[0]?.message_id).toBe("thread-msg");
    expect(directCacheValue?.[0]?.message_id).toBe("direct-msg");

    const threadEntry = client
      .getQueryCache()
      .findAll({ queryKey: networkKeys.threadsList("builders").slice(0, 4), exact: false });
    const directEntry = client
      .getQueryCache()
      .findAll({ queryKey: networkKeys.directsList("builders").slice(0, 4), exact: false });

    expect(threadEntry.flatMap(query => query.queryKey)).toContain("thread");
    expect(threadEntry.flatMap(query => query.queryKey)).not.toContain("direct");
    expect(directEntry.flatMap(query => query.queryKey)).toContain("direct");
    expect(directEntry.flatMap(query => query.queryKey)).not.toContain("thread");
  });

  it("normalizes message limit defaults inside both surfaces", async () => {
    const noLimit = networkThreadMessagesOptions("builders", "thread_one");
    const explicit = networkThreadMessagesOptions("builders", "thread_one", { limit: 120 });
    expect(noLimit.queryKey).toEqual(explicit.queryKey);
    await requireQueryFn(noLimit.queryFn)(makeQueryContext(noLimit.queryKey));
    expect(mocks.listNetworkThreadMessages).toHaveBeenLastCalledWith(
      "builders",
      "thread_one",
      { limit: 120 },
      expect.any(AbortSignal)
    );
  });

  it("does not retry 4xx conversation detail failures", () => {
    const threadRetry = requireRetry(networkThreadDetailOptions("builders", "missing").retry);
    const directRetry = requireRetry(networkDirectDetailOptions("builders", "missing").retry);

    expect(threadRetry(0, new NetworkApiError("Thread not found", 404))).toBe(false);
    expect(directRetry(0, new NetworkApiError("Invalid direct id", 400))).toBe(false);
  });

  it("retries transient conversation detail failures within the detail retry budget", () => {
    const retry = requireRetry(networkThreadDetailOptions("builders", "thread_one").retry);

    expect(retry(0, new Error("temporary network failure"))).toBe(true);
    expect(retry(1, new Error("temporary network failure"))).toBe(true);
    expect(retry(2, new Error("temporary network failure"))).toBe(false);
  });
});
