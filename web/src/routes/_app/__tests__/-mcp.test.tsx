import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SettingsMCPServerCollection, SettingsMCPServerEntry } from "@/systems/settings";
import type { WorkspacePayload } from "@/systems/workspace";
import type { useMcpPage } from "@/hooks/routes/use-mcp-page";
import type { useMCPAuthorize } from "@/hooks/routes/use-mcp-authorize";

type PageState = ReturnType<typeof useMcpPage>;
type AuthorizeState = ReturnType<typeof useMCPAuthorize>;

const filesystemEntry: SettingsMCPServerEntry = {
  name: "filesystem",
  transport: "stdio",
  command: "npx -y @modelcontextprotocol/server-filesystem",
  args: ["~/Dev"],
  scope: "global",
  runtime_status: {
    configured: true,
    initialized: true,
    state: "ready",
    probe: "succeeded",
    tool_count: 4,
  },
  source_metadata: {
    available_targets: ["global-mcp-sidecar", "global-config"],
    effective_source: { kind: "global-mcp-sidecar", scope: "global" },
    shadowed_sources: [{ kind: "global-config", scope: "global" }],
  },
};

const linearEntry: SettingsMCPServerEntry = {
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
    diagnostic: "token absent",
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
  workspace_id: "ws-polybot",
  source_metadata: {
    available_targets: ["workspace-config"],
    effective_source: { kind: "workspace-config", scope: "workspace", workspace_id: "ws-polybot" },
  },
};

const polybotWorkspace: WorkspacePayload = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/home/user/polybot",
  add_dirs: [],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};

const restartState = {
  isVisible: false,
  isRestartRequired: false,
  isPolling: false,
  isSuccessful: false,
  isFailed: false,
  operationId: null,
  status: null,
  failureReason: undefined,
  activeSessionCount: 0,
  lastMutation: null,
  trigger: vi.fn(),
  isTriggerPending: false,
  triggerError: null,
  dismiss: vi.fn(),
};

function makeCollection(
  servers: SettingsMCPServerEntry[],
  scope: "global" | "workspace"
): SettingsMCPServerCollection {
  return {
    collection: "mcp-servers",
    scope,
    available_scopes: ["global", "workspace"],
    mcp_servers: servers,
  };
}

let pageState: PageState;

const routeHarness = vi.hoisted(() => ({
  search: {
    scope: "workspace" as const,
    server: undefined as string | undefined,
    workspace_id: "ws-polybot" as string | undefined,
  },
  navigate: vi.fn(),
  hookOptions: [] as Array<Record<string, unknown>>,
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useSearch: () => routeHarness.search,
    useNavigate: () => routeHarness.navigate,
  }),
}));

vi.mock("@/hooks/routes/use-mcp-page", () => ({
  useMcpPage: (options: Record<string, unknown>) => {
    routeHarness.hookOptions.push(options);
    return pageState;
  },
}));

function makeAuthorize(overrides: Partial<AuthorizeState> = {}): AuthorizeState {
  return {
    phase: "idle",
    server: null,
    begin: null,
    error: null,
    prior: null,
    mode: "automatic",
    isBeginning: false,
    isExchanging: false,
    isAwaiting: false,
    isOpen: false,
    beginAuthorize: vi.fn(),
    retryBegin: vi.fn(),
    enterManual: vi.fn(),
    submitManual: vi.fn(),
    acknowledgeStatus: vi.fn(),
    cancel: vi.fn(),
    ...overrides,
  } as AuthorizeState;
}

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: makeCollection([filesystemEntry, linearEntry], "workspace"),
    servers: [filesystemEntry, linearEntry],
    counts: { total: 2, shadowed: 1 },
    restart: restartState,
    activeScope: "workspace",
    selectScope: vi.fn(),
    selectServer: vi.fn(),
    clearSelection: vi.fn(),
    selectedServer: "",
    selectedEntry: null,
    activeWorkspace: polybotWorkspace,
    activeWorkspaceId: polybotWorkspace.id,
    availableScopes: ["global", "workspace"],
    workspaceScopeAvailable: true,
    needsActiveWorkspace: false,
    queryEnabled: true,
    refetch: vi.fn(),
    editor: { mode: "closed" },
    editorErrors: {},
    editorIsValid: false,
    editorAvailableTargets: ["auto", "config", "sidecar"],
    editorVaultInventory: { status: "ready", refs: [] },
    editorSaveError: null,
    editorWarnings: undefined,
    editorIsSaving: false,
    openCreate: vi.fn(),
    openEdit: vi.fn(),
    closeEditor: vi.fn(),
    updateDraft: vi.fn(),
    setEditorTarget: vi.fn(),
    saveEditor: vi.fn(),
    requestRemoveFromEditor: vi.fn(),
    authorize: makeAuthorize(),
    authorizeEntry: null,
    openAuthorize: vi.fn(),
    deleteTarget: { mode: "closed" },
    deleteAvailableTargets: ["auto", "config", "sidecar"],
    deleteError: null,
    deleteIsPending: false,
    openDelete: vi.fn(),
    closeDelete: vi.fn(),
    setDeleteTargetKind: vi.fn(),
    confirmDelete: vi.fn(),
    lastAction: null,
    dismissLastAction: vi.fn(),
    ...overrides,
  } as PageState;
}

beforeEach(() => {
  pageState = makeState();
  routeHarness.navigate.mockReset();
  routeHarness.hookOptions.length = 0;
  routeHarness.search.scope = "workspace";
  routeHarness.search.server = undefined;
  routeHarness.search.workspace_id = "ws-polybot";
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../mcp";

const MCPPage = routeComponent(Route);

describe("MCPPage", () => {
  it("Should render the skeleton loading state", () => {
    pageState = makeState({ isLoading: true, envelope: null, servers: [] });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-loading")).toBeInTheDocument();
  });

  it("Should render the error state with a retry action", () => {
    pageState = makeState({ envelope: null, servers: [], error: new Error("nope") });
    render(<MCPPage />);
    const error = screen.getByTestId("settings-page-mcp-servers-error");
    expect(error).toHaveTextContent("could not be loaded");
    expect(screen.getByTestId("settings-page-mcp-servers-retry")).toBeInTheDocument();
  });

  it("Should render Workspace and Global scope pills in the topbar", () => {
    render(<MCPPage />);
    expect(screen.getByTestId("mcp-scope-workspace")).toHaveAttribute("data-active", "true");
    expect(screen.getByTestId("mcp-scope-global")).toHaveAttribute("data-active", "false");
  });

  it("Should pass the deep-linked workspace to the page hook", () => {
    render(<MCPPage />);

    expect(routeHarness.hookOptions.at(-1)).toMatchObject({
      scope: "workspace",
      workspaceId: "ws-polybot",
    });
  });

  it("Should render the composed status matrix without a hardcoded success dot", () => {
    render(<MCPPage />);
    // Every status cell traces to a payload field; no blanket-green row dot.
    expect(
      screen.queryByTestId("settings-page-mcp-servers-row-filesystem-status")
    ).not.toBeInTheDocument();
    const authCell = screen.getByTestId("settings-page-mcp-servers-row-linear-auth-pill");
    expect(authCell).toHaveTextContent("Needs login");
    expect(authCell).toHaveAttribute("data-tone", "warning");
    const runtimeCell = screen.getByTestId("settings-page-mcp-servers-row-linear-runtime-pill");
    expect(runtimeCell).toHaveTextContent("Auth required");
    const probeCode = screen.getByTestId("settings-page-mcp-servers-row-linear-probe-code");
    expect(probeCode).toHaveTextContent("no probe attempted");
    // Healthy stdio server renders its real tool count.
    expect(
      screen.getByTestId("settings-page-mcp-servers-row-filesystem-probe-code")
    ).toHaveTextContent("4 tools");
  });

  it("Should offer Authorize only for oauth remotes that need it", () => {
    render(<MCPPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-linear-authorize"));
    expect(pageState.openAuthorize).toHaveBeenCalledWith(linearEntry);
    // stdio server never gets a browser authorize action.
    expect(
      screen.queryByTestId("settings-page-mcp-servers-row-filesystem-authorize")
    ).not.toBeInTheDocument();
  });

  it("Should wire create and edit triggers (delete is relocated to the editor)", () => {
    render(<MCPPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-create"));
    expect(pageState.openCreate).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-filesystem-edit"));
    expect(pageState.openEdit).toHaveBeenCalledWith(filesystemEntry);
    expect(
      screen.queryByTestId("settings-page-mcp-servers-row-filesystem-delete")
    ).not.toBeInTheDocument();
  });

  it("Should report scope changes to the route on pill click", () => {
    render(<MCPPage />);
    fireEvent.click(screen.getByTestId("mcp-scope-global"));
    expect(pageState.selectScope).toHaveBeenCalledWith("global");
  });

  it("Should clear workspace identity when the page hook switches to global scope", () => {
    render(<MCPPage />);
    const options = routeHarness.hookOptions.at(-1);
    const onScopeChange = options?.onScopeChange as
      | ((scope: "workspace" | "global") => void)
      | undefined;

    onScopeChange?.("global");

    const navigation = routeHarness.navigate.mock.calls.at(-1)?.[0] as {
      search: (previous: Record<string, unknown>) => Record<string, unknown>;
    };
    expect(
      navigation.search({ scope: "workspace", server: "linear", workspace_id: "ws-polybot" })
    ).toMatchObject({ scope: "global", server: undefined, workspace_id: undefined });
  });

  it("Should synchronize a sidebar workspace switch into the route", () => {
    render(<MCPPage />);
    const options = routeHarness.hookOptions.at(-1);
    const onWorkspaceChange = options?.onWorkspaceChange as
      | ((workspaceId: string) => void)
      | undefined;

    onWorkspaceChange?.("ws-other");

    const navigation = routeHarness.navigate.mock.calls.at(-1)?.[0] as {
      search: (previous: Record<string, unknown>) => Record<string, unknown>;
    };
    expect(
      navigation.search({ scope: "workspace", server: "linear", workspace_id: "ws-polybot" })
    ).toMatchObject({ scope: "workspace", server: undefined, workspace_id: "ws-other" });
  });

  it("Should render provenance with a marketplace cross-link for a selected catalog server", () => {
    pageState = makeState({ selectedServer: "linear", selectedEntry: linearEntry });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-row-linear")).not.toHaveAttribute(
      "aria-selected"
    );
    expect(screen.getByTestId("settings-page-mcp-servers-row-linear-name")).toHaveAttribute(
      "aria-current",
      "true"
    );
    const strip = screen.getByTestId("settings-page-mcp-selection");
    expect(strip).toHaveTextContent("installed from catalog · linear@1.4.0");
    expect(screen.getByTestId("settings-page-mcp-selection-marketplace-link")).toHaveAttribute(
      "href",
      "/marketplace/mcp/linear"
    );
  });

  it("Should render the context header with independent-signal copy and scope", () => {
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-context")).toHaveTextContent(
      "independent signals"
    );
    expect(screen.getByTestId("settings-page-mcp-servers-scope-label")).toHaveTextContent(
      "polybot"
    );
  });

  it("Should render the Empty card with an add action when the scope is empty", () => {
    pageState = makeState({
      activeScope: "global",
      servers: [],
      envelope: makeCollection([], "global"),
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPPage />);
    const empty = screen.getByTestId("settings-page-mcp-servers-empty");
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No MCP servers configured");
  });

  it("Should guard the workspace scope when no workspace is active", () => {
    pageState = makeState({
      needsActiveWorkspace: true,
      activeWorkspaceId: null,
      activeWorkspace: undefined,
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-workspace-guard")).toBeInTheDocument();
    expect(screen.queryByTestId("settings-page-mcp-servers-create")).not.toBeInTheDocument();
  });
});
