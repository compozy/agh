import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useMemories, useMemory } from "@/systems/knowledge/hooks/use-knowledge";

vi.mock("@/systems/knowledge/adapters/knowledge-api", () => ({
  listMemories: vi.fn(),
  readMemory: vi.fn(),
  deleteMemory: vi.fn(),
  writeMemory: vi.fn(),
  consolidateMemory: vi.fn(),
}));

import { listMemories, readMemory } from "@/systems/knowledge/adapters/knowledge-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

const validHeader = {
  filename: "user_role.md",
  mod_time: "2026-04-01T12:00:00Z",
  name: "User Role",
  type: "user" as const,
};

describe("useMemories", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns loading state then data", async () => {
    vi.mocked(listMemories).mockResolvedValue([validHeader]);

    const { result } = renderHook(() => useMemories("global", "/ws"), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(result.current.data).toEqual([validHeader]);
    expect(listMemories).toHaveBeenCalledWith("global", "/ws", expect.any(AbortSignal));
  });
});

describe("useMemory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads a single memory content", async () => {
    vi.mocked(readMemory).mockResolvedValue("# Memory content");

    const { result } = renderHook(() => useMemory("global", "test.md", "/ws"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toBe("# Memory content");
    });

    expect(readMemory).toHaveBeenCalledWith("global", "test.md", "/ws", expect.any(AbortSignal));
  });

  it("does not fetch when scope is omitted", () => {
    renderHook(() => useMemory(undefined, "test.md"), {
      wrapper: createWrapper(),
    });

    expect(readMemory).not.toHaveBeenCalled();
  });

  it("does not fetch when filename is empty", () => {
    renderHook(() => useMemory("global", ""), {
      wrapper: createWrapper(),
    });

    expect(readMemory).not.toHaveBeenCalled();
  });
});
