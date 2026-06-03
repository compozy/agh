import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { WorkspacePayload } from "@/systems/workspace";

import { useOnboardingDraftStore } from "../../stores/use-onboarding-draft-store";
import { useOnboardingWorkspaces } from "../use-onboarding-workspaces";

const mocks = vi.hoisted(() => ({
  deleteWorkspace: vi.fn(),
  registeredWorkspaces: [] as WorkspacePayload[],
  resolveWorkspace: vi.fn(),
}));

vi.mock("@/systems/workspace", () => ({
  useDeleteWorkspace: () => ({
    isPending: false,
    mutateAsync: mocks.deleteWorkspace,
  }),
  useResolveWorkspace: () => ({
    isPending: false,
    mutateAsync: mocks.resolveWorkspace,
  }),
  useWorkspaces: () => ({
    data: mocks.registeredWorkspaces,
    error: null,
    isFetching: false,
    isLoading: false,
  }),
}));

vi.mock("../use-directory-browser", () => ({
  useDirectoryBrowser: () => ({
    data: {
      path: "/Users/operator",
      parent: null,
      home: "/Users/operator",
      entries: [],
    },
    error: null,
    isFetching: false,
    isLoading: false,
  }),
}));

const now = "2026-05-27T00:00:00Z";

function workspace(overrides: Partial<WorkspacePayload> = {}): WorkspacePayload {
  return {
    id: "ws_home",
    root_dir: "/Users/operator",
    add_dirs: [],
    name: "operator",
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

describe("useOnboardingWorkspaces", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    mocks.registeredWorkspaces = [];
    useOnboardingDraftStore.getState().reset();
  });

  it("hydrates an empty onboarding draft from daemon-registered workspaces", async () => {
    mocks.registeredWorkspaces = [
      workspace(),
      workspace({
        id: "ws_project",
        root_dir: "/Users/operator/project",
        name: "project",
      }),
    ];

    const { result } = renderHook(() => useOnboardingWorkspaces());

    await waitFor(() => {
      expect(result.current.workspaces).toEqual([
        { path: "/Users/operator", name: "operator", workspaceId: "ws_home" },
        { path: "/Users/operator/project", name: "project", workspaceId: "ws_project" },
      ]);
    });
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([
      { path: "/Users/operator", name: "operator", workspaceId: "ws_home" },
      { path: "/Users/operator/project", name: "project", workspaceId: "ws_project" },
    ]);
  });

  it("does not overwrite an existing onboarding draft with daemon workspaces", async () => {
    act(() => {
      useOnboardingDraftStore
        .getState()
        .addWorkspace({ path: "/Users/operator/manual", name: "manual" });
    });
    mocks.registeredWorkspaces = [workspace()];

    const { result } = renderHook(() => useOnboardingWorkspaces());

    await waitFor(() => {
      expect(result.current.workspaces).toEqual([
        { path: "/Users/operator/manual", name: "manual" },
      ]);
    });
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([
      { path: "/Users/operator/manual", name: "manual" },
    ]);
  });

  it("deletes the registered workspace before removing a resolved folder from the draft", async () => {
    mocks.resolveWorkspace.mockResolvedValue(
      workspace({
        id: "ws_project",
        root_dir: "/Users/operator/project",
        name: "project",
      })
    );
    mocks.deleteWorkspace.mockResolvedValue(undefined);

    const { result } = renderHook(() => useOnboardingWorkspaces());

    await act(async () => {
      await result.current.addWorkspace("/Users/operator/project");
    });
    await waitFor(() => {
      expect(result.current.workspaces).toEqual([
        { path: "/Users/operator/project", name: "project", workspaceId: "ws_project" },
      ]);
    });

    await act(async () => {
      await result.current.removeWorkspace("/Users/operator/project");
    });

    expect(mocks.deleteWorkspace).toHaveBeenCalledWith("ws_project");
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([]);
  });

  it("keeps the folder in the draft when removing its registered workspace fails", async () => {
    act(() => {
      useOnboardingDraftStore.getState().addWorkspace({
        path: "/Users/operator/project",
        name: "project",
        workspaceId: "ws_project",
      });
    });
    mocks.deleteWorkspace.mockRejectedValue(new Error("workspace has active sessions"));

    const { result } = renderHook(() => useOnboardingWorkspaces());

    await act(async () => {
      await result.current.removeWorkspace("/Users/operator/project");
    });

    expect(mocks.deleteWorkspace).toHaveBeenCalledWith("ws_project");
    expect(result.current.resolveError).toBe("workspace has active sessions");
    expect(useOnboardingDraftStore.getState().workspaces).toEqual([
      { path: "/Users/operator/project", name: "project", workspaceId: "ws_project" },
    ]);
  });
});
