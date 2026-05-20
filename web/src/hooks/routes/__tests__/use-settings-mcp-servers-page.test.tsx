import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsRestartStatus: vi.fn(),
  listSettingsMCPServers: vi.fn(),
  putSettingsMCPServer: vi.fn(),
  deleteSettingsMCPServer: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
}));

vi.mock("@/systems/workspace/adapters/workspace-api", () => ({
  fetchWorkspaces: vi.fn(),
  fetchWorkspace: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

import {
  deleteSettingsMCPServer,
  listSettingsMCPServers,
  putSettingsMCPServer,
  SettingsApiError,
} from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsMCPServerCollection } from "@/systems/settings";
import { fetchWorkspaces } from "@/systems/workspace/adapters/workspace-api";
import type { WorkspacePayload } from "@/systems/workspace";
import { useSettingsMCPServersPage } from "../use-settings-mcp-servers-page";

const polybotWorkspace: WorkspacePayload = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/home/user/polybot",
  add_dirs: [],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};

const filesystemEntry: SettingsMCPServerCollection["mcp_servers"][number] = {
  name: "filesystem",
  transport: "stdio",
  command: "npx -y @modelcontextprotocol/server-filesystem",
  args: ["~/Dev"],
  scope: "global",
  source_metadata: {
    available_targets: ["global-mcp-sidecar", "global-config"],
    effective_source: { kind: "global-mcp-sidecar", scope: "global" },
    shadowed_sources: [{ kind: "global-config", scope: "global" }],
  },
};

const githubEntry: SettingsMCPServerCollection["mcp_servers"][number] = {
  name: "github",
  transport: "stdio",
  command: "npx -y @modelcontextprotocol/server-github",
  env: { GITHUB_TOKEN: "env:GITHUB_TOKEN" },
  scope: "global",
  source_metadata: {
    available_targets: ["global-mcp-sidecar"],
    effective_source: { kind: "global-mcp-sidecar", scope: "global" },
  },
};

const globalCollection: SettingsMCPServerCollection = {
  collection: "mcp-servers",
  scope: "global",
  available_scopes: ["global", "workspace"],
  mcp_servers: [filesystemEntry, githubEntry],
};

const workspaceEntry: SettingsMCPServerCollection["mcp_servers"][number] = {
  name: "paper",
  transport: "stdio",
  command: "npx -y @paper-design/mcp-paper",
  scope: "workspace",
  workspace_id: polybotWorkspace.id,
  source_metadata: {
    available_targets: ["workspace-mcp-sidecar", "workspace-config"],
    effective_source: {
      kind: "workspace-config",
      scope: "workspace",
      workspace_id: polybotWorkspace.id,
    },
  },
};

const workspaceCollection: SettingsMCPServerCollection = {
  collection: "mcp-servers",
  scope: "workspace",
  workspace_id: polybotWorkspace.id,
  available_scopes: ["global", "workspace"],
  mcp_servers: [workspaceEntry],
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
  vi.mocked(listSettingsMCPServers).mockImplementation(async filter => {
    if (filter?.scope === "workspace") {
      return workspaceCollection;
    }
    return globalCollection;
  });
  vi.mocked(fetchWorkspaces).mockResolvedValue([polybotWorkspace]);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsMCPServersPage", () => {
  it("loads the global collection with precedence metadata", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => {
      expect(result.current.servers).toHaveLength(2);
    });

    expect(result.current.selection).toEqual({ scope: "global" });
    expect(result.current.counts).toEqual({ total: 2, shadowed: 1 });
    expect(result.current.availableScopes).toEqual(["global", "workspace"]);
  });

  it("switches to workspace scope and reloads the scoped collection", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.selectWorkspace(polybotWorkspace.id);
    });

    await waitFor(() => {
      expect(result.current.servers).toEqual([workspaceEntry]);
    });

    expect(result.current.selection).toEqual({
      scope: "workspace",
      workspaceId: polybotWorkspace.id,
    });
    expect(listSettingsMCPServers).toHaveBeenLastCalledWith(
      { scope: "workspace", workspace_id: polybotWorkspace.id },
      expect.anything()
    );
  });

  it("opens a create editor with auto target and empty draft", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
    });

    expect(result.current.editor).toMatchObject({
      mode: "create",
      draft: { name: "", command: "", args: [], env: [] },
      target: "auto",
    });
  });

  it("blocks create save when the name collides with an existing server", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
      result.current.updateDraft(draft => ({
        ...draft,
        name: "FileSystem",
        command: "npx cmd",
      }));
    });

    expect(result.current.editorIsValid).toBe(false);
  });

  it("seeds the edit draft from the entry and exposes available targets", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openEdit(filesystemEntry);
    });

    expect(result.current.editor).toMatchObject({
      mode: "edit",
      name: "filesystem",
      draft: expect.objectContaining({
        command: "npx -y @modelcontextprotocol/server-filesystem",
        args: ["~/Dev"],
      }),
      target: "auto",
    });
    expect(result.current.editorAvailableTargets).toEqual(["auto", "config", "sidecar"]);
  });

  it("submits auto-target PUT with global scope and records last action", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
      write_target: "global-mcp-sidecar",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openEdit(filesystemEntry);
    });
    act(() => {
      result.current.updateDraft(draft => ({ ...draft, command: "npx filesystem-v2" }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsMCPServer).toHaveBeenCalledWith(
      "filesystem",
      {
        server: {
          name: "filesystem",
          transport: "stdio",
          command: "npx filesystem-v2",
          args: ["~/Dev"],
        },
      },
      { scope: "global", target: "auto" }
    );
    expect(result.current.editor.mode).toBe("closed");
  });

  it("persists target=sidecar when operator changes the target selector", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
      write_target: "global-mcp-sidecar",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
      result.current.updateDraft(draft => ({
        ...draft,
        name: "new-server",
        command: "npx new",
      }));
      result.current.setEditorTarget("sidecar");
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsMCPServer).toHaveBeenCalledWith(
      "new-server",
      {
        server: { name: "new-server", transport: "stdio", command: "npx new" },
      },
      { scope: "global", target: "sidecar" }
    );
  });

  it("submits workspace-scoped mutations when workspace scope is active", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "mcp-servers",
      scope: "workspace",
      workspace_id: polybotWorkspace.id,
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
      write_target: "workspace-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.selectWorkspace(polybotWorkspace.id);
    });
    await waitFor(() => expect(result.current.servers).toEqual([workspaceEntry]));

    act(() => {
      result.current.openEdit(workspaceEntry);
    });
    act(() => {
      result.current.updateDraft(draft => ({ ...draft, command: "npx paper-v2" }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsMCPServer).toHaveBeenCalledWith(
      "paper",
      {
        server: { name: "paper", transport: "stdio", command: "npx paper-v2" },
      },
      { scope: "workspace", workspace_id: polybotWorkspace.id, target: "auto" }
    );
  });

  it("surfaces validation errors from the adapter without closing the editor", async () => {
    vi.mocked(putSettingsMCPServer).mockRejectedValue(
      new SettingsApiError("invalid server command", 400)
    );

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openEdit(filesystemEntry);
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.editorError).toBe("invalid server command");
    });
    expect(result.current.editor.mode).toBe("edit");
    expect(result.current.lastAction).toBeNull();
  });

  it("reports remainingShadowed on delete so the UI can explain fallback", async () => {
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
      write_target: "global-mcp-sidecar",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openDelete(filesystemEntry);
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("deleted");
    });

    expect(deleteSettingsMCPServer).toHaveBeenCalledWith("filesystem", {
      scope: "global",
      target: "auto",
    });
    expect(result.current.lastAction).toMatchObject({
      name: "filesystem",
      remainingShadowed: 1,
    });
  });

  it("passes the selected delete target to the adapter", async () => {
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
      active_config_hash: "sha256:test-active",
      active_generation: 1,
      apply_record_id: "cfg_apply_test",
      lifecycle: "live",
      next_action: "none",
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openDelete(filesystemEntry);
      result.current.setDeleteTargetKind("config");
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("deleted");
    });

    expect(deleteSettingsMCPServer).toHaveBeenCalledWith("filesystem", {
      scope: "global",
      target: "config",
    });
  });

  it("resets editor/delete state when switching scopes mid-flow", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsMCPServersPage(), { wrapper });

    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => {
      result.current.openEdit(filesystemEntry);
    });
    expect(result.current.editor.mode).toBe("edit");

    act(() => {
      result.current.selectWorkspace(polybotWorkspace.id);
    });

    expect(result.current.editor.mode).toBe("closed");
    expect(result.current.deleteTarget.mode).toBe("closed");
  });
});
