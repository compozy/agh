import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { KnowledgeMemoryItem, MemoryHeader } from "@/systems/knowledge/types";

const useMemoriesMock = vi.fn();
const useMemoryMock = vi.fn();
const deleteMutateAsync = vi.fn();
const deleteReset = vi.fn();

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: {
      id: "ws_signalforge",
      root_dir: "/workspaces/signalforge",
      add_dirs: [],
      name: "signalforge",
      created_at: "2026-04-25T12:00:00Z",
      updated_at: "2026-04-25T12:00:00Z",
    },
    activeWorkspaceId: "ws_signalforge",
  }),
}));

vi.mock("@/systems/knowledge", async () => {
  const actual = await vi.importActual("@/systems/knowledge");
  return {
    ...actual,
    useMemories: (...args: unknown[]) => useMemoriesMock(...args),
    useMemory: (...args: unknown[]) => useMemoryMock(...args),
    useDeleteMemory: () => ({
      error: null,
      isPending: false,
      mutateAsync: deleteMutateAsync,
      reset: deleteReset,
    }),
  };
});

import { useKnowledgePage } from "./use-knowledge-page";

const GLOBAL_MEMORY: MemoryHeader = {
  filename: "operator-playbook-0425.md",
  mod_time: "2026-04-25T21:00:00Z",
  name: "Operator Playbook 0425",
  description: "Reusable operator checklist",
  type: "reference",
};

const WORKSPACE_MEMORY: MemoryHeader = {
  filename: "launch-brief-0425.md",
  mod_time: "2026-04-25T21:01:00Z",
  name: "Launch Brief 0425",
  description: "Workspace launch brief",
  type: "project",
};

describe("useKnowledgePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useMemoriesMock.mockImplementation(
      (scope?: string, workspace?: string, options?: { enabled?: boolean }) => {
        if (scope === "global") {
          return { data: [GLOBAL_MEMORY], isLoading: false, error: null, options };
        }
        if (scope === "workspace") {
          return {
            data: workspace ? [WORKSPACE_MEMORY] : [],
            isLoading: false,
            error: null,
            options,
          };
        }
        return { data: [], isLoading: false, error: null, options };
      }
    );
    useMemoryMock.mockReturnValue({
      data: "# Memory content",
      isLoading: false,
      error: null,
    });
    deleteMutateAsync.mockResolvedValue({ ok: true });
  });

  it("uses the active workspace path when loading and reading workspace knowledge", async () => {
    const { result } = renderHook(() => useKnowledgePage());

    await waitFor(() => {
      expect(result.current.memories).toHaveLength(2);
    });

    expect(useMemoriesMock).toHaveBeenNthCalledWith(1, "global");
    expect(useMemoriesMock).toHaveBeenCalledWith(
      "workspace",
      "/workspaces/signalforge",
      expect.objectContaining({ enabled: true })
    );

    act(() => {
      result.current.setActiveTab("workspace");
    });

    await waitFor(() => {
      expect(result.current.selectedScope).toBe("workspace");
    });

    expect(useMemoryMock).toHaveBeenLastCalledWith(
      "workspace",
      "launch-brief-0425.md",
      "/workspaces/signalforge"
    );
  });

  it("deletes workspace knowledge using the active workspace path", async () => {
    const { result } = renderHook(() => useKnowledgePage());

    act(() => {
      result.current.setActiveTab("workspace");
    });

    await waitFor(() => {
      expect(result.current.memories).toHaveLength(1);
    });

    await act(async () => {
      await result.current.handleDelete(result.current.memories[0] as KnowledgeMemoryItem);
    });

    expect(deleteMutateAsync).toHaveBeenCalledWith({
      scope: "workspace",
      filename: "launch-brief-0425.md",
      workspace: "/workspaces/signalforge",
    });
  });
});
