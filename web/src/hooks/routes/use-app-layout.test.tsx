import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockNavigate, mockMutateAsync, mockToastError } = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockMutateAsync: vi.fn(),
  mockToastError: vi.fn(),
}));

let mockActiveWorkspaceId: string | null = "ws_alpha";

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("sonner", () => ({
  toast: {
    error: mockToastError,
  },
}));

vi.mock("@/hooks/use-sidebar-store", () => ({
  useSidebarStore: (
    selector: (state: { collapsed: boolean; setCollapsed: (next: boolean) => void }) => unknown
  ) => selector({ collapsed: false, setCollapsed: vi.fn() }),
}));

vi.mock("@/systems/daemon", () => ({
  useDaemonHealth: () => ({
    health: { version: "0.1.0" },
    connectionStatus: "connected",
  }),
}));

vi.mock("@/systems/agent", () => ({
  useAgents: () => ({
    data: [],
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/session", () => ({
  useCreateSession: () => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useSessions: () => ({
    data: [],
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces:
      mockActiveWorkspaceId === null
        ? []
        : [
            {
              id: "ws_alpha",
              root_dir: "/workspace/alpha",
              add_dirs: [],
              name: "alpha",
              created_at: "2026-04-20T10:00:00Z",
              updated_at: "2026-04-20T10:00:00Z",
            },
          ],
    hasWorkspaces: mockActiveWorkspaceId !== null,
    activeWorkspace:
      mockActiveWorkspaceId === null
        ? undefined
        : {
            id: "ws_alpha",
            root_dir: "/workspace/alpha",
            add_dirs: [],
            name: "alpha",
            created_at: "2026-04-20T10:00:00Z",
            updated_at: "2026-04-20T10:00:00Z",
          },
    activeWorkspaceId: mockActiveWorkspaceId,
    setActiveWorkspaceId: vi.fn(),
    isLoading: false,
    isError: false,
  }),
}));

import { useAppLayout } from "./use-app-layout";

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;

  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

describe("useAppLayout", () => {
  beforeEach(() => {
    mockActiveWorkspaceId = "ws_alpha";
    mockNavigate.mockReset();
    mockMutateAsync.mockReset();
    mockToastError.mockReset();
  });

  it("keeps the pending sidebar state until navigation settles", async () => {
    const mutation = createDeferred<{ id: string }>();
    const navigation = createDeferred<void>();
    mockMutateAsync.mockReturnValue(mutation.promise);
    mockNavigate.mockReturnValue(navigation.promise);

    const { result } = renderHook(() => useAppLayout());

    act(() => {
      void result.current.handleNewSession("claude-agent");
    });

    expect(result.current.isCreatingSession).toBe(true);
    expect(result.current.pendingSessionAgentName).toBe("claude-agent");
    expect(result.current.pendingSessionWorkspaceId).toBe("ws_alpha");
    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "claude-agent",
      workspace: "ws_alpha",
    });

    await act(async () => {
      mutation.resolve({ id: "sess-created" });
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({
        to: "/session/$id",
        params: { id: "sess-created" },
      });
    });

    expect(result.current.isCreatingSession).toBe(true);
    expect(result.current.pendingSessionAgentName).toBe("claude-agent");

    await act(async () => {
      navigation.resolve();
      await navigation.promise;
    });

    expect(result.current.isCreatingSession).toBe(false);
    expect(result.current.pendingSessionAgentName).toBeNull();
    expect(result.current.pendingSessionWorkspaceId).toBeNull();
  });

  it("clears pending state and shows a toast when creation fails", async () => {
    mockMutateAsync.mockRejectedValue(new Error("Failed to create"));

    const { result } = renderHook(() => useAppLayout());

    await act(async () => {
      await result.current.handleNewSession("claude-agent");
    });

    expect(mockToastError).toHaveBeenCalledWith("Failed to create");
    expect(mockNavigate).not.toHaveBeenCalled();
    expect(result.current.isCreatingSession).toBe(false);
    expect(result.current.pendingSessionAgentName).toBeNull();
  });
});
