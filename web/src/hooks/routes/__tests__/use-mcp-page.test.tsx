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

vi.mock("@/systems/settings/adapters/settings-mcp-auth-api", () => ({
  beginSettingsMCPAuth: vi.fn(),
  exchangeSettingsMCPAuth: vi.fn(),
  logoutSettingsMCPAuth: vi.fn(),
}));

vi.mock("@/systems/vault/adapters/vault-api", () => ({
  listVaultSecrets: vi.fn(async () => []),
  getVaultSecret: vi.fn(),
}));

vi.mock("@/systems/workspace/adapters/workspace-api", () => ({
  fetchWorkspaces: vi.fn(),
  fetchWorkspace: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

let mockActiveWorkspaceId: string | null = "ws-polybot";
const setActiveWorkspaceId = vi.fn();
let mockActiveWorkspace: {
  id: string;
  name: string;
  root_dir: string;
  add_dirs: string[];
  created_at: string;
  updated_at: string;
} | null = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/home/user/polybot",
  add_dirs: [],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};
let mockWorkspaces = mockActiveWorkspace ? [mockActiveWorkspace] : [];

vi.mock("@/systems/workspace", async () => {
  const actual = await vi.importActual<typeof import("@/systems/workspace")>("@/systems/workspace");
  return {
    ...actual,
    useActiveWorkspace: () => ({
      activeWorkspace: mockActiveWorkspace,
      activeWorkspaceId: mockActiveWorkspaceId,
      workspaces: mockWorkspaces,
      hasWorkspaces: mockWorkspaces.length > 0,
      hasHydrated: true,
      selectedWorkspaceId: mockActiveWorkspaceId,
      setActiveWorkspaceId,
      clearActiveWorkspaceSelection: vi.fn(),
    }),
  };
});

import {
  deleteSettingsMCPServer,
  listSettingsMCPServers,
  putSettingsMCPServer,
  SettingsApiError,
} from "@/systems/settings/adapters/settings-api";
import { beginSettingsMCPAuth } from "@/systems/settings/adapters/settings-mcp-auth-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsMCPServerCollection } from "@/systems/settings";
import { type MCPActiveScope, useMcpPage, type UseMcpPageOptions } from "../use-mcp-page";

const polybotWorkspace = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/home/user/polybot",
  add_dirs: [] as string[],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};

const otherWorkspace = {
  ...polybotWorkspace,
  id: "ws-other",
  name: "other",
  root_dir: "/home/user/other",
};

type Entry = SettingsMCPServerCollection["mcp_servers"][number];

const filesystemEntry: Entry = {
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

const githubEntry: Entry = {
  name: "github",
  transport: "stdio",
  command: "npx -y @modelcontextprotocol/server-github",
  env_keys: ["GITHUB_TOKEN"],
  secret_env_keys: ["GITHUB_TOKEN"],
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

const paperEntry: Entry = {
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

const linearEntry: Entry = {
  name: "linear",
  transport: "http",
  url: "https://mcp.linear.app/mcp",
  auth: {
    type: "oauth2_pkce",
    client_id: "agh-linear-public",
    issuer_url: "https://auth.linear.app",
    client_secret_configured: false,
  },
  auth_status: {
    server_name: "linear",
    scope: "workspace",
    status: "needs_login",
    token_present: false,
    refreshable: true,
  },
  runtime_status: {
    configured: true,
    initialized: false,
    state: "auth_required",
    probe: "skipped",
    tool_count: 0,
  },
  catalog_entry: "linear",
  catalog_version: "1.4.0",
  scope: "workspace",
  workspace_id: polybotWorkspace.id,
  source_metadata: {
    available_targets: ["workspace-config"],
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
  mcp_servers: [paperEntry, linearEntry],
};

function renderMcpPage(options: Partial<UseMcpPageOptions> = {}) {
  const onScopeChange = vi.fn();
  const onWorkspaceChange = vi.fn();
  const onSelectServer = vi.fn();
  const resolved: UseMcpPageOptions = {
    scope: (options.scope ?? "workspace") as MCPActiveScope,
    workspaceId: options.workspaceId,
    selectedServer: options.selectedServer ?? "",
    onScopeChange: options.onScopeChange ?? onScopeChange,
    onWorkspaceChange: options.onWorkspaceChange ?? onWorkspaceChange,
    onSelectServer: options.onSelectServer ?? onSelectServer,
  };
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  const view = renderHook(() => useMcpPage(resolved), { wrapper });
  return { ...view, onScopeChange, onWorkspaceChange, onSelectServer };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockActiveWorkspaceId = polybotWorkspace.id;
  mockActiveWorkspace = polybotWorkspace;
  mockWorkspaces = [polybotWorkspace];
  setActiveWorkspaceId.mockImplementation(workspaceId => {
    mockActiveWorkspaceId = workspaceId;
    mockActiveWorkspace = mockWorkspaces.find(workspace => workspace.id === workspaceId) ?? null;
  });
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(listSettingsMCPServers).mockImplementation(async filter =>
    filter?.scope === "workspace" ? workspaceCollection : globalCollection
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useMcpPage scope + selection", () => {
  it("lists the active workspace scope and reflects the URL-provided scope", async () => {
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toEqual([paperEntry, linearEntry]));
    expect(result.current.activeScope).toBe("workspace");
    expect(listSettingsMCPServers).toHaveBeenCalledWith(
      { scope: "workspace", workspace_id: polybotWorkspace.id },
      expect.anything()
    );
  });

  it("lists the global scope when the URL selects global", async () => {
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toEqual([filesystemEntry, githubEntry]));
    expect(listSettingsMCPServers).toHaveBeenCalledWith({ scope: "global" }, expect.anything());
  });

  it("uses and activates a valid workspace from the deep link", async () => {
    mockWorkspaces = [polybotWorkspace, otherWorkspace];
    const { result } = renderMcpPage({ scope: "workspace", workspaceId: otherWorkspace.id });

    await waitFor(() => expect(result.current.activeWorkspaceId).toBe(otherWorkspace.id));
    expect(result.current.activeWorkspace).toEqual(otherWorkspace);
    expect(listSettingsMCPServers).toHaveBeenCalledWith(
      { scope: "workspace", workspace_id: otherWorkspace.id },
      expect.anything()
    );
    expect(setActiveWorkspaceId).toHaveBeenCalledWith(otherWorkspace.id);
  });

  it("adopts a deep link once and then follows the sidebar while syncing the URL", async () => {
    mockWorkspaces = [polybotWorkspace, otherWorkspace];
    const { result, rerender, onWorkspaceChange } = renderMcpPage({
      scope: "workspace",
      workspaceId: otherWorkspace.id,
    });

    await waitFor(() => expect(result.current.activeWorkspaceId).toBe(otherWorkspace.id));
    expect(setActiveWorkspaceId).toHaveBeenCalledWith(otherWorkspace.id);
    rerender();

    mockActiveWorkspaceId = polybotWorkspace.id;
    mockActiveWorkspace = polybotWorkspace;
    rerender();

    await waitFor(() => expect(result.current.activeWorkspaceId).toBe(polybotWorkspace.id));
    expect(listSettingsMCPServers).toHaveBeenCalledWith(
      { scope: "workspace", workspace_id: polybotWorkspace.id },
      expect.anything()
    );
    expect(onWorkspaceChange).toHaveBeenCalledWith(polybotWorkspace.id);
  });

  it("falls back to the active workspace when a deep link names an unknown workspace", async () => {
    const { result } = renderMcpPage({ scope: "workspace", workspaceId: "ws-missing" });

    await waitFor(() => expect(result.current.activeWorkspaceId).toBe(polybotWorkspace.id));
    expect(listSettingsMCPServers).toHaveBeenCalledWith(
      { scope: "workspace", workspace_id: polybotWorkspace.id },
      expect.anything()
    );
    expect(setActiveWorkspaceId).not.toHaveBeenCalled();
  });

  it("reports scope changes to the route instead of holding local state", async () => {
    const { result, onScopeChange } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.selectScope("global"));
    expect(onScopeChange).toHaveBeenCalledWith("global");
  });

  it("resolves the preselected server and reports selection changes to the route", async () => {
    const { result, onSelectServer } = renderMcpPage({
      scope: "workspace",
      selectedServer: "linear",
    });
    await waitFor(() => expect(result.current.selectedEntry?.name).toBe("linear"));
    act(() => result.current.selectServer("paper"));
    expect(onSelectServer).toHaveBeenCalledWith("paper");
    act(() => result.current.clearSelection());
    expect(onSelectServer).toHaveBeenCalledWith("");
  });

  it("guards the workspace scope when no workspace is active", async () => {
    mockActiveWorkspaceId = null;
    mockActiveWorkspace = null;
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.needsActiveWorkspace).toBe(true));
    expect(result.current.error).toBeNull();
    expect(result.current.queryEnabled).toBe(false);
    expect(listSettingsMCPServers).not.toHaveBeenCalled();
  });
});

describe("useMcpPage editor serialization", () => {
  it("creates a blank stdio draft that is invalid until a command is present", async () => {
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openCreate());
    expect(result.current.editor.mode).toBe("create");
    expect(result.current.editorIsValid).toBe(false);
    act(() => result.current.updateDraft(draft => ({ ...draft, name: "slack", command: "npx" })));
    expect(result.current.editorIsValid).toBe(true);
  });

  it("serializes a stdio save with transport stdio and command", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openCreate());
    act(() => result.current.updateDraft(draft => ({ ...draft, name: "slack", command: "npx" })));
    act(() => result.current.saveEditor());
    await waitFor(() => expect(putSettingsMCPServer).toHaveBeenCalled());
    const [name, body] = vi.mocked(putSettingsMCPServer).mock.calls[0];
    expect(name).toBe("slack");
    expect(body.server.transport).toBe("stdio");
    expect(body.server.command).toBe("npx");
    expect(body.server.url).toBeUndefined();
  });

  it("edits a remote server and serializes url + oauth without stdio fields", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "workspace",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openEdit(linearEntry));
    expect(result.current.editor.mode).toBe("edit");
    if (result.current.editor.mode === "edit") {
      expect(result.current.editor.draft.transport).toBe("http");
      expect(result.current.editor.draft.oauth.enabled).toBe(true);
    }
    act(() => result.current.saveEditor());
    await waitFor(() => expect(putSettingsMCPServer).toHaveBeenCalled());
    const [, body] = vi.mocked(putSettingsMCPServer).mock.calls[0];
    expect(body.server.transport).toBe("http");
    expect(body.server.url).toBe("https://mcp.linear.app/mcp");
    expect(body.server.command).toBeUndefined();
    expect(body.server.auth?.type).toBe("oauth2_pkce");
  });

  it("locks an exact-source edit to its presence-bearing target", async () => {
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => result.current.openEdit(githubEntry));

    expect(result.current.editor).toMatchObject({ mode: "edit", target: "sidecar" });
    expect(result.current.editorAvailableTargets).toEqual(["sidecar"]);
    if (result.current.editor.mode === "edit") {
      expect(result.current.editor.draft.secretEnv[0].binding).toMatchObject({
        mode: "preserve",
        existing: true,
      });
    }
  });

  it("strips source presence when editing an inherited entry into a workspace override", async () => {
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));

    act(() => result.current.openEdit(githubEntry));

    expect(result.current.editor).toMatchObject({ mode: "edit", target: "auto" });
    expect(result.current.editorAvailableTargets).toEqual(["auto", "config", "sidecar"]);
    expect(result.current.editorIsValid).toBe(false);
    if (result.current.editor.mode === "edit") {
      expect(result.current.editor.draft.secretEnv[0]).toMatchObject({
        originalKey: undefined,
        binding: { mode: "typed", existing: false },
      });
    }
  });

  it("invalidates a remote draft that is missing a url", async () => {
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openCreate());
    act(() =>
      result.current.updateDraft(draft => ({
        ...draft,
        name: "linear",
        transport: "http",
        url: "",
      }))
    );
    expect(result.current.editorIsValid).toBe(false);
    expect(result.current.editorErrors.url).toBe("URL is required for http transport");
  });

  it("blocks a create-mode name collision", async () => {
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openCreate());
    act(() => result.current.updateDraft(draft => ({ ...draft, name: "github", command: "npx" })));
    expect(result.current.editorIsValid).toBe(false);
    expect(result.current.editorErrors.name).toContain("already exists");
  });

  it("persists the operator-selected sidecar target on save", async () => {
    vi.mocked(putSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openCreate());
    act(() =>
      result.current.updateDraft(draft => ({ ...draft, name: "new-server", command: "npx new" }))
    );
    act(() => result.current.setEditorTarget("sidecar"));
    act(() => result.current.saveEditor());
    await waitFor(() => expect(result.current.lastAction?.kind).toBe("saved"));
    expect(putSettingsMCPServer).toHaveBeenCalledWith(
      "new-server",
      expect.anything(),
      expect.objectContaining({ scope: "global", target: "sidecar" })
    );
  });

  it("surfaces adapter validation errors without closing the editor", async () => {
    vi.mocked(putSettingsMCPServer).mockRejectedValue(
      new SettingsApiError("invalid server command", 400)
    );
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openEdit(filesystemEntry));
    act(() => result.current.saveEditor());
    await waitFor(() => expect(result.current.editorSaveError).toBe("invalid server command"));
    expect(result.current.editor.mode).toBe("edit");
    expect(result.current.lastAction).toBeNull();
  });
});

describe("useMcpPage authorize + delete", () => {
  it("begins authorization for an oauth remote and exposes the live URL", async () => {
    vi.mocked(beginSettingsMCPAuth).mockResolvedValue({
      authorization_url: "https://auth.linear.app/oauth/authorize?state=x",
      callback_url: "http://127.0.0.1:2123/api/mcp/oauth/callback",
      expires_at: "2026-07-15T00:05:00Z",
      manual_supported: true,
      state: "agh_mcp_x",
    });
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openAuthorize(linearEntry));
    await waitFor(() => expect(result.current.authorize.phase).toBe("waiting"));
    expect(beginSettingsMCPAuth).toHaveBeenCalledWith(
      "linear",
      {
        scope: "workspace",
        workspace_id: polybotWorkspace.id,
      },
      { mode: "automatic" }
    );
    expect(result.current.authorize.begin?.authorization_url).toContain("auth.linear.app");
  });

  it("deletes a server with the scoped target", async () => {
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "workspace",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openDelete(paperEntry));
    act(() => result.current.confirmDelete());
    await waitFor(() => expect(deleteSettingsMCPServer).toHaveBeenCalled());
    const [name, filter] = vi.mocked(deleteSettingsMCPServer).mock.calls[0];
    expect(name).toBe("paper");
    expect(filter).toMatchObject({ scope: "workspace", workspace_id: polybotWorkspace.id });
  });

  it("reports remainingShadowed after deleting a shadowed server", async () => {
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openDelete(filesystemEntry));
    act(() => result.current.confirmDelete());
    await waitFor(() => expect(result.current.lastAction?.kind).toBe("deleted"));
    expect(result.current.lastAction).toMatchObject({ name: "filesystem", remainingShadowed: 1 });
  });

  it("passes the operator-selected delete target to the adapter", async () => {
    vi.mocked(deleteSettingsMCPServer).mockResolvedValue({
      section: "general",
      scope: "global",
      applied: true,
    } as never);
    const { result } = renderMcpPage({ scope: "global" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openDelete(filesystemEntry));
    act(() => result.current.setDeleteTargetKind("config"));
    act(() => result.current.confirmDelete());
    await waitFor(() => expect(deleteSettingsMCPServer).toHaveBeenCalled());
    expect(deleteSettingsMCPServer).toHaveBeenCalledWith(
      "filesystem",
      expect.objectContaining({ scope: "global", target: "config" })
    );
  });
});

describe("useMcpPage transient reset", () => {
  it("resets the open editor and delete dialog when the operator switches scope", async () => {
    const { result } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => {
      result.current.openEdit(paperEntry);
      result.current.openDelete(linearEntry);
    });
    expect(result.current.editor.mode).toBe("edit");
    expect(result.current.deleteTarget.mode).toBe("open");
    act(() => result.current.selectScope("global"));
    expect(result.current.editor.mode).toBe("closed");
    expect(result.current.deleteTarget.mode).toBe("closed");
  });

  it("resets the open editor and delete dialog when the active workspace changes", async () => {
    const { result, rerender } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => {
      result.current.openEdit(paperEntry);
      result.current.openDelete(linearEntry);
    });
    expect(result.current.editor.mode).toBe("edit");
    expect(result.current.deleteTarget.mode).toBe("open");

    mockActiveWorkspaceId = "ws-other";
    mockActiveWorkspace = { ...polybotWorkspace, id: "ws-other", name: "other" };
    rerender();

    await waitFor(() => expect(result.current.editor.mode).toBe("closed"));
    expect(result.current.deleteTarget.mode).toBe("closed");
  });

  it("cancels an in-flight OAuth authorization when the active workspace changes", async () => {
    vi.mocked(beginSettingsMCPAuth).mockResolvedValue({
      authorization_url: "https://auth.linear.app/oauth/authorize?state=x",
      callback_url: "http://127.0.0.1:2123/api/mcp/oauth/callback",
      expires_at: "2026-07-15T00:05:00Z",
      manual_supported: true,
      state: "agh_mcp_x",
    });
    const { result, rerender } = renderMcpPage({ scope: "workspace" });
    await waitFor(() => expect(result.current.servers).toHaveLength(2));
    act(() => result.current.openAuthorize(linearEntry));
    await waitFor(() => expect(result.current.authorize.phase).toBe("waiting"));

    mockActiveWorkspaceId = "ws-other";
    mockActiveWorkspace = { ...polybotWorkspace, id: "ws-other", name: "other" };
    rerender();

    await waitFor(() => expect(result.current.authorize.phase).toBe("idle"));
    expect(result.current.authorize.isOpen).toBe(false);
  });
});
