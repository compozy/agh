import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "../../types";
import { agentKeys } from "../../lib/query-keys";
import { AgentDigestConflictError } from "../../adapters/agent-api";

const {
  mockUpdateAgent,
  mockDeleteAgent,
  mockDuplicateAgent,
  mockCreateAgent,
  mockFetchAgents,
  mockFetchAgent,
  mockNavRefresh,
  mockGetNavCountsStore,
} = vi.hoisted(() => {
  const mockNavRefresh = vi.fn();
  return {
    mockUpdateAgent: vi.fn(),
    mockDeleteAgent: vi.fn(),
    mockDuplicateAgent: vi.fn(),
    mockCreateAgent: vi.fn(),
    mockFetchAgents: vi.fn(),
    mockFetchAgent: vi.fn(),
    mockNavRefresh,
    mockGetNavCountsStore: vi.fn(() => ({
      getState: () => ({ refresh: mockNavRefresh }),
    })),
  };
});

vi.mock("../../adapters/agent-api", async () => {
  const actual = await vi.importActual<typeof import("../../adapters/agent-api")>(
    "../../adapters/agent-api"
  );
  return {
    ...actual,
    updateAgent: mockUpdateAgent,
    deleteAgent: mockDeleteAgent,
    duplicateAgent: mockDuplicateAgent,
    createAgent: mockCreateAgent,
    fetchAgents: mockFetchAgents,
    fetchAgent: mockFetchAgent,
  };
});

vi.mock("@/systems/runtime", () => ({
  getNavCountsStore: mockGetNavCountsStore,
}));

import {
  useAgent,
  useAgents,
  useCreateAgent,
  useDeleteAgent,
  useDuplicateAgent,
  useUpdateAgent,
} from "../use-agents";

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

function makeAgent(overrides: Partial<AgentPayload> = {}): AgentPayload {
  return {
    name: "coder",
    provider: "claude",
    prompt: "Ship carefully.",
    definition_digest: "d1",
    origin: "workspace",
    ...overrides,
  };
}

describe("use-agent-mutations", () => {
  beforeEach(() => {
    mockUpdateAgent.mockReset();
    mockDeleteAgent.mockReset();
    mockDuplicateAgent.mockReset();
    mockCreateAgent.mockReset();
    mockFetchAgents.mockReset();
    mockFetchAgent.mockReset();
    mockNavRefresh.mockReset();
    mockGetNavCountsStore.mockClear();
  });

  it("Should load agents via useAgents", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    mockFetchAgents.mockResolvedValue([makeAgent()]);
    const { result } = renderHook(() => useAgents("ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetchAgents).toHaveBeenCalledWith("ws_alpha", expect.any(AbortSignal));

    mockFetchAgent.mockResolvedValue(makeAgent());
    const detail = renderHook(() => useAgent("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => expect(detail.result.current.isSuccess).toBe(true));
    expect(mockFetchAgent).toHaveBeenCalledWith("coder", "ws_alpha", expect.any(AbortSignal));
  });

  it("Should cache created agents on create success", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const created = makeAgent({ name: "new-agent" });
    mockCreateAgent.mockResolvedValue(created);
    const { result } = renderHook(() => useCreateAgent(), {
      wrapper: createWrapper(queryClient),
    });
    await act(async () => {
      await result.current.mutateAsync({
        scope: "workspace",
        workspace: "ws_alpha",
        agent: { name: "new-agent", provider: "claude", prompt: "Ship carefully." },
      });
    });
    expect(queryClient.getQueryData(agentKeys.detail("new-agent", "ws_alpha"))).toEqual(created);
  });

  it("Should set detail with fresh digest and invalidate lists on update", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const updated = makeAgent({ prompt: "Updated", definition_digest: "d2" });
    mockUpdateAgent.mockResolvedValue(updated);
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useUpdateAgent(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "coder",
        params: {
          expected_digest: "d1",
          agent: { name: "coder", provider: "claude", prompt: "Updated" },
          workspace: "ws_alpha",
        },
      });
    });

    expect(queryClient.getQueryData(agentKeys.detail("coder", "ws_alpha"))).toEqual(updated);
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: agentKeys.lists() });
  });

  it("Should not poison detail cache on digest conflict", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const stale = makeAgent();
    queryClient.setQueryData(agentKeys.detail("coder", "ws_alpha"), stale);
    mockUpdateAgent.mockRejectedValue(new AgentDigestConflictError("definition digest conflict"));

    const { result } = renderHook(() => useUpdateAgent(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "coder",
          params: {
            expected_digest: "stale",
            agent: { name: "coder", provider: "claude", prompt: "Nope" },
            workspace: "ws_alpha",
          },
        })
      ).rejects.toBeInstanceOf(AgentDigestConflictError);
    });

    expect(queryClient.getQueryData(agentKeys.detail("coder", "ws_alpha"))).toEqual(stale);
  });

  it("Should remove detail, refresh nav counts, and return unshadowed_origin on delete", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(agentKeys.detail("coder", "ws_alpha"), makeAgent());
    mockDeleteAgent.mockResolvedValue({
      name: "coder",
      origin: "workspace",
      unshadowed_origin: "global",
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useDeleteAgent(), {
      wrapper: createWrapper(queryClient),
    });

    let response: unknown;
    await act(async () => {
      response = await result.current.mutateAsync({
        name: "coder",
        workspace: "ws_alpha",
      });
    });

    expect(response).toEqual({
      name: "coder",
      origin: "workspace",
      unshadowed_origin: "global",
    });
    expect(queryClient.getQueryData(agentKeys.detail("coder", "ws_alpha"))).toBeUndefined();
    expect(mockGetNavCountsStore).toHaveBeenCalledWith("ws_alpha");
    expect(mockNavRefresh).toHaveBeenCalledTimes(1);
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: agentKeys.lists() });
  });

  it("Should cache the duplicated agent detail", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const copy = makeAgent({ name: "coder-copy", definition_digest: "d3" });
    mockDuplicateAgent.mockResolvedValue(copy);

    const { result } = renderHook(() => useDuplicateAgent(), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        sourceName: "coder",
        params: {
          name: "coder-copy",
          scope: "workspace",
          workspace: "ws_alpha",
        },
      });
    });

    await waitFor(() => {
      expect(queryClient.getQueryData(agentKeys.detail("coder-copy", "ws_alpha"))).toEqual(copy);
    });
  });
});
