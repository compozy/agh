import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsRestartStatus: vi.fn(),
  listSettingsEnvironments: vi.fn(),
  putSettingsEnvironment: vi.fn(),
  deleteSettingsEnvironment: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
}));

import {
  deleteSettingsEnvironment,
  listSettingsEnvironments,
  putSettingsEnvironment,
  SettingsApiError,
} from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsEnvironmentCollection } from "@/systems/settings";
import { useSettingsEnvironmentsPage } from "./use-settings-environments-page";

const localEnv: SettingsEnvironmentCollection["environments"][number] = {
  name: "local",
  workspace_usage_count: 3,
  profile: {
    backend: "local",
    sync_mode: "none",
    persistence: "transient",
    runtime_root: "~",
    env: { NODE_ENV: "development" },
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "global-config", scope: "global" },
  },
};

const daytonaEnv: SettingsEnvironmentCollection["environments"][number] = {
  name: "daytona-staging",
  workspace_usage_count: 1,
  profile: {
    backend: "daytona",
    sync_mode: "session-bidir",
    persistence: "reuse",
    runtime_root: "/workspace",
    network: { required: true, allow_outbound: true },
    daytona: { api_url: "https://daytona.dev", target: "staging" },
  },
  source_metadata: {
    available_targets: ["global-config", "workspace-config"],
    effective_source: {
      kind: "workspace-config",
      scope: "workspace",
      workspace_id: "ws_alpha",
    },
    shadowed_sources: [{ kind: "global-config", scope: "global" }],
  },
};

const collection: SettingsEnvironmentCollection = {
  collection: "environments",
  scope: "global",
  available_scopes: ["global"],
  environments: [localEnv, daytonaEnv],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(listSettingsEnvironments).mockResolvedValue(collection);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsEnvironmentsPage", () => {
  it("computes total counts and workspace usage", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));
    expect(result.current.counts).toEqual({ total: 2, totalWorkspaces: 4 });
  });

  it("seeds edit drafts from the selected entry without dropping it", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));

    act(() => {
      result.current.openEdit(daytonaEnv);
    });

    expect(result.current.editor).toMatchObject({
      mode: "edit",
      name: "daytona-staging",
      draft: expect.objectContaining({
        backend: "daytona",
        sync_mode: "session-bidir",
        persistence: "reuse",
        runtime_root: "/workspace",
      }),
    });
  });

  it("submits the replace request preserving unedited nested profile keys", async () => {
    vi.mocked(putSettingsEnvironment).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));

    act(() => {
      result.current.openEdit(daytonaEnv);
    });
    act(() => {
      result.current.updateDraft(draft => ({ ...draft, runtime_root: "/home/agh" }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => expect(result.current.lastAction?.kind).toBe("saved"));

    expect(putSettingsEnvironment).toHaveBeenCalledWith("daytona-staging", {
      profile: expect.objectContaining({
        backend: "daytona",
        sync_mode: "session-bidir",
        persistence: "reuse",
        runtime_root: "/home/agh",
        network: { required: true, allow_outbound: true },
        daytona: { api_url: "https://daytona.dev", target: "staging" },
      }),
    });
  });

  it("rejects saves when the backend is empty", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));

    act(() => {
      result.current.openEdit(daytonaEnv);
      result.current.updateDraft(draft => ({ ...draft, backend: "" }));
    });

    expect(result.current.editorIsValid).toBe(false);
  });

  it("records delete actions with the workspace usage count", async () => {
    vi.mocked(deleteSettingsEnvironment).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));

    act(() => {
      result.current.openDelete(localEnv);
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => expect(result.current.lastAction?.kind).toBe("deleted"));
    expect(deleteSettingsEnvironment).toHaveBeenCalledWith("local");
    expect(result.current.lastAction).toMatchObject({ name: "local", usageCount: 3 });
  });

  it("keeps the delete target open on conflict errors", async () => {
    vi.mocked(deleteSettingsEnvironment).mockRejectedValue(
      new SettingsApiError("environment still referenced", 409)
    );

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsEnvironmentsPage(), { wrapper });

    await waitFor(() => expect(result.current.environments).toHaveLength(2));

    act(() => {
      result.current.openDelete(localEnv);
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => expect(result.current.deleteError).toBe("environment still referenced"));
    expect(result.current.deleteTarget.mode).toBe("open");
  });
});
