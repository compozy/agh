import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useResolveWorkspace, useWorkspace, useWorkspaces } from "../use-workspaces";
import { workspaceKeys } from "../../lib/query-keys";

vi.mock("@/systems/workspace/adapters/workspace-api", () => ({
  fetchWorkspace: vi.fn(),
  fetchWorkspaces: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

import {
  fetchWorkspace,
  fetchWorkspaces,
  resolveWorkspace,
} from "@/systems/workspace/adapters/workspace-api";

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

  it("loads one resolved workspace detail", async () => {
    vi.mocked(fetchWorkspace).mockResolvedValue({
      agents: [{ name: "coder", prompt: "code", provider: "openai" }],
      sessions: [],
      skills: [],
      workspace: {
        id: "ws_alpha",
        root_dir: "/workspace/alpha",
        add_dirs: [],
        name: "alpha",
        created_at: "2026-04-06T10:00:00Z",
        updated_at: "2026-04-06T10:00:00Z",
      },
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const { result } = renderHook(() => useWorkspace("ws_alpha"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.workspace.id).toBe("ws_alpha");
    });

    expect(fetchWorkspace).toHaveBeenCalledWith("ws_alpha", expect.any(AbortSignal));
  });

  it("allows callers to disable the workspace detail query", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    renderHook(() => useWorkspace("ws_alpha", { enabled: false }), {
      wrapper: createWrapper(queryClient),
    });

    expect(fetchWorkspace).not.toHaveBeenCalled();
  });

  it("invalidates the workspace list after resolving a workspace", async () => {
    const resolvedWorkspace = {
      id: "ws_alpha",
      root_dir: "/workspace/alpha",
      add_dirs: [],
      name: "alpha",
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    };
    vi.mocked(resolveWorkspace).mockResolvedValue(resolvedWorkspace);

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
    expect(queryClient.getQueryData(workspaceKeys.list())).toEqual([resolvedWorkspace]);
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: workspaceKeys.lists() });
  });
});
