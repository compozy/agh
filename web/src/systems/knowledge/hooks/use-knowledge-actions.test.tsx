import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useConsolidateMemory, useDeleteMemory } from "./use-knowledge-actions";

vi.mock("../adapters/knowledge-api", () => ({
  listMemories: vi.fn(),
  readMemory: vi.fn(),
  deleteMemory: vi.fn(),
  writeMemory: vi.fn(),
  consolidateMemory: vi.fn(),
}));

import { consolidateMemory, deleteMemory } from "../adapters/knowledge-api";

describe("useDeleteMemory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls deleteMemory and invalidates memory list cache on settle", async () => {
    vi.mocked(deleteMemory).mockResolvedValue({ ok: true });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);

    const { result } = renderHook(() => useDeleteMemory(), { wrapper });

    act(() => {
      result.current.mutate({ scope: "global", filename: "old.md" });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(deleteMemory).toHaveBeenCalledWith("global", "old.md", undefined);
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["knowledge"],
    });
  });
});

describe("useConsolidateMemory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls consolidateMemory and invalidates cache on settle", async () => {
    vi.mocked(consolidateMemory).mockResolvedValue({ triggered: true });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children);

    const { result } = renderHook(() => useConsolidateMemory(), { wrapper });

    act(() => {
      result.current.mutate({ workspace: "/home/user/project" });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(consolidateMemory).toHaveBeenCalledWith("/home/user/project");
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["knowledge"],
    });
  });
});
