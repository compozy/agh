import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useResolveWorkspace, useWorkspaces } from "./use-workspaces";
import { workspaceKeys } from "../lib/query-keys";

vi.mock("../adapters/workspace-api", () => ({
  fetchWorkspaces: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

import { fetchWorkspaces, resolveWorkspace } from "../adapters/workspace-api";

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("workspace hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads the workspace registry", async () => {
    vi.mocked(fetchWorkspaces).mockResolvedValue([
      {
        id: "ws_alpha",
        root_dir: "/workspace/alpha",
        add_dirs: [],
        name: "alpha",
        created_at: "2026-04-06T10:00:00Z",
        updated_at: "2026-04-06T10:00:00Z",
      },
    ]);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const { result } = renderHook(() => useWorkspaces(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(fetchWorkspaces).toHaveBeenCalledOnce();
  });

  it("invalidates the workspace list after resolving a workspace", async () => {
    vi.mocked(resolveWorkspace).mockResolvedValue({
      id: "ws_alpha",
      root_dir: "/workspace/alpha",
      add_dirs: [],
      name: "alpha",
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useResolveWorkspace(), {
      wrapper: createWrapper(queryClient),
    });

    result.current.mutate({ path: "/workspace/alpha" });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(resolveWorkspace).toHaveBeenCalledWith({ path: "/workspace/alpha" });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: workspaceKeys.lists() });
  });
});
