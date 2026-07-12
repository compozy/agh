import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { agentKeys } from "../../lib/query-keys";

const {
  mockFetchSoul,
  mockPutSoul,
  mockDeleteSoul,
  mockValidateSoul,
  mockRollbackSoul,
  mockFetchSoulHistory,
} = vi.hoisted(() => ({
  mockFetchSoul: vi.fn(),
  mockPutSoul: vi.fn(),
  mockDeleteSoul: vi.fn(),
  mockValidateSoul: vi.fn(),
  mockRollbackSoul: vi.fn(),
  mockFetchSoulHistory: vi.fn(),
}));

vi.mock("../../adapters/agent-soul-api", () => ({
  fetchAgentSoul: mockFetchSoul,
  putAgentSoul: mockPutSoul,
  deleteAgentSoul: mockDeleteSoul,
  validateAgentSoul: mockValidateSoul,
  rollbackAgentSoul: mockRollbackSoul,
  fetchAgentSoulHistory: mockFetchSoulHistory,
}));

import {
  useAgentSoul,
  useAgentSoulHistory,
  useDeleteAgentSoul,
  usePutAgentSoul,
  useRollbackAgentSoul,
  useValidateAgentSoul,
} from "../use-agent-soul";

const soul = {
  active: true,
  present: true,
  enabled: true,
  body: "Be helpful.",
  digest: "a".repeat(64),
  valid: true,
  validation_status: "valid" as const,
  frontmatter: {},
  limits: { max_body_bytes: 65536 },
  config_provenance: {
    digest: "a".repeat(64),
    enabled: true,
    context_projection_bytes: 0,
    max_body_bytes: 65536,
  },
};

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("use-agent-soul", () => {
  beforeEach(() => {
    mockFetchSoul.mockReset();
    mockPutSoul.mockReset();
    mockDeleteSoul.mockReset();
    mockValidateSoul.mockReset();
    mockRollbackSoul.mockReset();
    mockFetchSoulHistory.mockReset();
    mockFetchSoul.mockResolvedValue(soul);
    mockPutSoul.mockResolvedValue({ soul, revision: { id: "r1" } });
    mockDeleteSoul.mockResolvedValue({ soul: { ...soul, active: false }, revision: { id: "r2" } });
    mockValidateSoul.mockResolvedValue(soul);
    mockRollbackSoul.mockResolvedValue({ soul, revision: { id: "r3" } });
    mockFetchSoulHistory.mockResolvedValue({ revisions: [] });
  });

  it("Should load soul and cache put results", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const { result } = renderHook(() => useAgentSoul("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetchSoul).toHaveBeenCalledWith("coder", "ws_alpha", expect.any(AbortSignal));

    const history = renderHook(() => useAgentSoulHistory("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => expect(history.result.current.isSuccess).toBe(true));
    expect(mockFetchSoulHistory).toHaveBeenCalled();

    const put = renderHook(() => usePutAgentSoul("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await act(async () => {
      await put.result.current.mutateAsync({
        body: "Be helpful.",
        expected_digest: "a".repeat(64),
      });
    });
    expect(queryClient.getQueryData(agentKeys.soul("coder", "ws_alpha"))).toEqual(soul);
  });

  it("Should validate, delete, and rollback through mutation hooks", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const validate = renderHook(() => useValidateAgentSoul("coder"), {
      wrapper: createWrapper(queryClient),
    });
    await act(async () => {
      await validate.result.current.mutateAsync({ body: "Be helpful." });
    });
    expect(mockValidateSoul).toHaveBeenCalledWith("coder", { body: "Be helpful." });

    const del = renderHook(() => useDeleteAgentSoul("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await act(async () => {
      await del.result.current.mutateAsync({ expected_digest: "a".repeat(64) });
    });
    expect(mockDeleteSoul).toHaveBeenCalled();

    const rollback = renderHook(() => useRollbackAgentSoul("coder", "ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });
    await act(async () => {
      await rollback.result.current.mutateAsync({
        revision_id: "r1",
        expected_digest: "a".repeat(64),
      });
    });
    expect(queryClient.getQueryData(agentKeys.soul("coder", "ws_alpha"))).toEqual(soul);
  });
});
