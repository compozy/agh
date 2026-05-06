// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";

import { networkKeys } from "../../lib/query-keys";
import type { NetworkConversationMessage } from "../../types";

const sendNetworkMessageMock = vi.fn();
const resolveNetworkDirectRoomMock = vi.fn();
const toastErrorMock = vi.fn();

vi.mock("../../adapters/network-api", async () => {
  const actual = await vi.importActual<typeof import("../../adapters/network-api")>(
    "../../adapters/network-api"
  );
  return {
    ...actual,
    sendNetworkMessage: (...args: unknown[]) => sendNetworkMessageMock(...args),
    resolveNetworkDirectRoom: (...args: unknown[]) => resolveNetworkDirectRoomMock(...args),
  };
});

vi.mock("@agh/ui", async () => {
  const actual = await vi.importActual<typeof import("@agh/ui")>("@agh/ui");
  return {
    ...actual,
    toast: {
      error: (...args: unknown[]) => toastErrorMock(...args),
    },
  };
});

import {
  THREAD_COLLISION_TOAST,
  useCreateNetworkThread,
  useResolveNetworkDirectRoom,
  useSendNetworkMessage,
} from "../use-network-actions";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  return { queryClient, wrapper };
}

describe("useSendNetworkMessage", () => {
  beforeEach(() => {
    sendNetworkMessageMock.mockReset();
    toastErrorMock.mockReset();
  });

  it("Should append an optimistic message immediately and replace it on success", async () => {
    const { queryClient, wrapper } = createWrapper();
    queryClient.setQueryData(
      networkKeys.threadMessages("ops", "thread-1"),
      [] as NetworkConversationMessage[]
    );
    sendNetworkMessageMock.mockResolvedValue({
      message: {
        id: "client-id",
        kind: "say",
        channel: "ops",
        session_id: "sess-1",
      },
    });

    const { result } = renderHook(() => useSendNetworkMessage(), { wrapper });

    let promise: Promise<unknown> | null = null;
    await act(async () => {
      promise = result.current.send({
        surface: "thread",
        channel: "ops",
        threadId: "thread-1",
        sessionId: "sess-1",
        peerFrom: "peer-self",
        text: "Hello world",
      });
      // Allow the mutationFn microtask to run so the optimistic cache update
      // becomes observable before we assert against it.
      await Promise.resolve();
    });

    const cacheAfterMutate = queryClient.getQueryData<NetworkConversationMessage[]>(
      networkKeys.threadMessages("ops", "thread-1")
    );
    expect(cacheAfterMutate).toBeDefined();
    expect(cacheAfterMutate?.length).toBe(1);
    expect(cacheAfterMutate?.[0]?.text).toBe("Hello world");

    await act(async () => {
      await promise;
    });

    const cacheAfterSuccess = queryClient.getQueryData<NetworkConversationMessage[]>(
      networkKeys.threadMessages("ops", "thread-1")
    );
    expect(cacheAfterSuccess?.length).toBe(1);
    const replaced = cacheAfterSuccess?.[0] as NetworkConversationMessage & {
      optimistic?: string;
    };
    expect(replaced?.optimistic).toBeUndefined();
  });

  it("Should mark the optimistic message as failed when the send rejects", async () => {
    const { queryClient, wrapper } = createWrapper();
    queryClient.setQueryData(
      networkKeys.threadMessages("ops", "thread-1"),
      [] as NetworkConversationMessage[]
    );
    sendNetworkMessageMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useSendNetworkMessage(), { wrapper });

    await expect(
      act(() =>
        result.current.send({
          surface: "thread",
          channel: "ops",
          threadId: "thread-1",
          sessionId: "sess-1",
          peerFrom: "peer-self",
          text: "Hello world",
        })
      )
    ).rejects.toThrow("boom");

    const cache = queryClient.getQueryData<NetworkConversationMessage[]>(
      networkKeys.threadMessages("ops", "thread-1")
    );
    const failed = cache?.[0] as NetworkConversationMessage & { optimistic?: string };
    expect(failed?.optimistic).toBe("failed");
  });

  it("Should never construct a request body containing kind:'direct'", async () => {
    const { queryClient, wrapper } = createWrapper();
    queryClient.setQueryData(
      networkKeys.directMessages("ops", "direct-1"),
      [] as NetworkConversationMessage[]
    );
    sendNetworkMessageMock.mockResolvedValue({
      message: { id: "x", kind: "say", channel: "ops", session_id: "sess-1" },
    });

    const { result } = renderHook(() => useSendNetworkMessage(), { wrapper });
    await act(async () => {
      await result.current.send({
        surface: "direct",
        channel: "ops",
        directId: "direct-1",
        sessionId: "sess-1",
        peerFrom: "peer-self",
        text: "secret",
      });
    });

    expect(sendNetworkMessageMock).toHaveBeenCalledTimes(1);
    const sent = sendNetworkMessageMock.mock.calls[0]?.[0] as Record<string, unknown>;
    expect(sent.kind).toBe("say");
    expect(sent.surface).toBe("direct");
    expect(sent.direct_id).toBe("direct-1");
    expect(sent).not.toHaveProperty("interaction_id");
  });

  it("Should drop the optimistic message when discard is invoked", async () => {
    const { queryClient, wrapper } = createWrapper();
    queryClient.setQueryData(
      networkKeys.threadMessages("ops", "thread-1"),
      [] as NetworkConversationMessage[]
    );
    sendNetworkMessageMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useSendNetworkMessage(), { wrapper });
    let failedId = "";
    await act(async () => {
      try {
        await result.current.send({
          surface: "thread",
          channel: "ops",
          threadId: "thread-1",
          sessionId: "sess-1",
          peerFrom: "peer-self",
          text: "Hello",
        });
      } catch {
        // expected
      }
    });

    const cache = queryClient.getQueryData<NetworkConversationMessage[]>(
      networkKeys.threadMessages("ops", "thread-1")
    );
    failedId = cache?.[0]?.message_id ?? "";
    expect(failedId).toBeTruthy();

    act(() => {
      result.current.discard(
        {
          surface: "thread",
          channel: "ops",
          threadId: "thread-1",
          sessionId: "sess-1",
          peerFrom: "peer-self",
          text: "Hello",
        },
        failedId
      );
    });

    const after = queryClient.getQueryData<NetworkConversationMessage[]>(
      networkKeys.threadMessages("ops", "thread-1")
    );
    expect(after?.length).toBe(0);
  });
});

describe("useCreateNetworkThread", () => {
  beforeEach(() => {
    sendNetworkMessageMock.mockReset();
    toastErrorMock.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("Should retry exactly once when the first attempt fails", async () => {
    const { wrapper } = createWrapper();
    sendNetworkMessageMock.mockRejectedValueOnce(new Error("collision"));
    sendNetworkMessageMock.mockResolvedValueOnce({
      message: { id: "second", kind: "say", channel: "ops", session_id: "sess-1" },
    });

    const { result } = renderHook(() => useCreateNetworkThread(), { wrapper });

    let outcome: { threadId: string; rootMessageId: string } | null = null;
    await act(async () => {
      outcome = await result.current.createThread({
        channel: "ops",
        sessionId: "sess-1",
        peerFrom: "peer-self",
        text: "Open this thread",
      });
    });

    expect(sendNetworkMessageMock).toHaveBeenCalledTimes(2);
    const finalOutcome = outcome as { threadId: string; rootMessageId: string } | null;
    expect(finalOutcome?.threadId.startsWith("thread_")).toBe(true);
    const firstCall = sendNetworkMessageMock.mock.calls[0] ?? [];
    const secondCall = sendNetworkMessageMock.mock.calls[1] ?? [];
    const firstThreadId = (firstCall[0] as { thread_id: string }).thread_id;
    const secondThreadId = (secondCall[0] as { thread_id: string }).thread_id;
    expect(firstThreadId).not.toBe(secondThreadId);
    expect(toastErrorMock).not.toHaveBeenCalled();
  });

  it("Should surface a single Sonner toast after the second collision", async () => {
    const { wrapper } = createWrapper();
    sendNetworkMessageMock.mockRejectedValue(new Error("collision again"));

    const { result } = renderHook(() => useCreateNetworkThread(), { wrapper });

    await expect(
      act(() =>
        result.current.createThread({
          channel: "ops",
          sessionId: "sess-1",
          peerFrom: "peer-self",
          text: "Open this thread",
        })
      )
    ).rejects.toBeTruthy();

    expect(sendNetworkMessageMock).toHaveBeenCalledTimes(2);
    expect(toastErrorMock).toHaveBeenCalledTimes(1);
    expect(toastErrorMock).toHaveBeenCalledWith(THREAD_COLLISION_TOAST);
  });
});

describe("useResolveNetworkDirectRoom", () => {
  beforeEach(() => {
    resolveNetworkDirectRoomMock.mockReset();
  });

  it("Should call the resolve adapter with the channel and body", async () => {
    const { wrapper } = createWrapper();
    resolveNetworkDirectRoomMock.mockResolvedValue({
      channel: "ops",
      direct_id: "direct-x",
      peer_a: "a",
      peer_b: "b",
      message_count: 0,
      open_work_count: 0,
    });

    const { result } = renderHook(() => useResolveNetworkDirectRoom(), { wrapper });
    type Resolved = { direct_id: string };
    let resolved: Resolved | null = null;
    await act(async () => {
      resolved = (await result.current.resolveRoom({
        channel: "ops",
        body: { peer_id: "a", session_id: "sess-1" },
      })) as Resolved;
    });

    await waitFor(() => expect(result.current.isResolving).toBe(false));
    expect(resolveNetworkDirectRoomMock).toHaveBeenCalledWith("ops", {
      peer_id: "a",
      session_id: "sess-1",
    });
    expect((resolved as Resolved | null)?.direct_id).toBe("direct-x");
  });
});
