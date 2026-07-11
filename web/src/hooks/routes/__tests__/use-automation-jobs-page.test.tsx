import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const pageState = vi.hoisted(() => ({
  current: {
    activeWorkspace: null,
    activeWorkspaceId: null as string | null,
    automationRuntime: null,
    deferredSearchQuery: "",
    listFilters: { limit: 50 },
    scopeFilter: "all" as const,
    searchQuery: "",
    selectedId: null,
    setScopeFilter: vi.fn(),
    setSearchQuery: vi.fn(),
    setSelectedId: vi.fn(),
    workspaces: [],
  },
}));

vi.mock("../use-automation-page-base", async importOriginal => {
  const actual = await importOriginal<typeof import("../use-automation-page-base")>();
  return {
    ...actual,
    useAutomationPageBase: () => pageState.current,
  };
});

vi.mock("@/systems/automation", async importOriginal => {
  const actual = await importOriginal<typeof import("@/systems/automation")>();
  return {
    ...actual,
    useAutomationJobs: () => ({
      error: null,
      fetchNextPage: vi.fn(),
      hasNextPage: false,
      isFetchingNextPage: false,
      isLoading: false,
      jobs: [],
      total: 0,
    }),
    useAutomationJob: () => ({ data: null, error: null, isLoading: false }),
    useAutomationJobRuns: () => ({ data: [], error: null, isLoading: false }),
    useCreateAutomationJob: () => ({ isPending: false, mutateAsync: vi.fn() }),
    useDeleteAutomationJob: () => ({ isPending: false, mutateAsync: vi.fn() }),
    useTriggerAutomationJob: () => ({ isPending: false, mutateAsync: vi.fn() }),
    useUpdateAutomationJob: () => ({ isPending: false, mutateAsync: vi.fn() }),
  };
});

const { useAutomationJobsPage } = await import("../use-automation-jobs-page");

describe("useAutomationJobsPage", () => {
  beforeEach(() => {
    pageState.current = {
      activeWorkspace: null,
      activeWorkspaceId: null,
      automationRuntime: null,
      deferredSearchQuery: "",
      listFilters: { limit: 50 },
      scopeFilter: "all",
      searchQuery: "",
      selectedId: null,
      setScopeFilter: vi.fn(),
      setSearchQuery: vi.fn(),
      setSelectedId: vi.fn(),
      workspaces: [],
    };
  });

  it("Should wait for an active workspace before consuming a loop-target job seed", async () => {
    const { result, rerender } = renderHook(() =>
      useAutomationJobsPage({ loop: "software-delivery" })
    );

    await act(async () => {});
    expect(result.current.editorDialogProps.editor).toBeNull();

    pageState.current = {
      ...pageState.current,
      activeWorkspaceId: "ws_default",
    };
    rerender();

    await waitFor(() =>
      expect(result.current.editorDialogProps.editor?.draft.loop_target).toMatchObject({
        loop_name: "software-delivery",
        workspace_id: "ws_default",
      })
    );
    expect(result.current.editorDialogProps.editor?.draft.workspace_id).toBe("ws_default");
  });
});
