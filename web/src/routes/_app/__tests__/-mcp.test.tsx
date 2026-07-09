import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SettingsMCPServerEntry, SettingsMCPServerTarget } from "@/systems/settings";
import type { WorkspacePayload } from "@/systems/workspace";

type RestartBanner = {
  isVisible: boolean;
  isRestartRequired: boolean;
  isPolling: boolean;
  isSuccessful: boolean;
  isFailed: boolean;
  operationId: string | null;
  status: string | null;
  failureReason?: string;
  activeSessionCount: number;
  trigger: ReturnType<typeof vi.fn>;
  isTriggerPending: boolean;
  triggerError: unknown;
  dismiss: ReturnType<typeof vi.fn>;
};

const filesystemEntry: SettingsMCPServerEntry = {
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

const githubEntry: SettingsMCPServerEntry = {
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

const polybotWorkspace: WorkspacePayload = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/home/user/polybot",
  add_dirs: [],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
};

type DeleteTarget =
  | { mode: "closed" }
  | { mode: "open"; entry: SettingsMCPServerEntry; target: SettingsMCPServerTarget };

type PageState = {
  isLoading: boolean;
  error: Error | null;
  envelope: { mcp_servers: SettingsMCPServerEntry[] } | null;
  servers: SettingsMCPServerEntry[];
  counts: { total: number; shadowed: number };
  restart: RestartBanner;
  activeScope: "workspace" | "global";
  activeWorkspace: WorkspacePayload | null;
  activeWorkspaceId: string | null;
  selectScope: ReturnType<typeof vi.fn>;
  editor:
    | { mode: "closed" }
    | {
        mode: "create";
        draft: {
          name: string;
          command: string;
          args: string[];
          env: { key: string; value: string }[];
        };
        target: SettingsMCPServerTarget;
      }
    | {
        mode: "edit";
        name: string;
        draft: {
          name: string;
          command: string;
          args: string[];
          env: { key: string; value: string }[];
        };
        entry: SettingsMCPServerEntry;
        target: SettingsMCPServerTarget;
      };
  editorIsValid: boolean;
  editorAvailableTargets: SettingsMCPServerTarget[];
  editorError: string | null;
  editorWarnings: string[] | undefined;
  editorIsSaving: boolean;
  openCreate: ReturnType<typeof vi.fn>;
  openEdit: ReturnType<typeof vi.fn>;
  closeEditor: ReturnType<typeof vi.fn>;
  updateDraft: ReturnType<typeof vi.fn>;
  setEditorTarget: ReturnType<typeof vi.fn>;
  saveEditor: ReturnType<typeof vi.fn>;
  deleteTarget: DeleteTarget;
  deleteAvailableTargets: SettingsMCPServerTarget[];
  deleteError: string | null;
  deleteIsPending: boolean;
  openDelete: ReturnType<typeof vi.fn>;
  closeDelete: ReturnType<typeof vi.fn>;
  setDeleteTargetKind: ReturnType<typeof vi.fn>;
  confirmDelete: ReturnType<typeof vi.fn>;
  lastAction: null | {
    kind: "saved" | "deleted";
    name: string;
    result: { restart_required: boolean; write_target?: string };
    remainingShadowed?: number;
  };
  dismissLastAction: ReturnType<typeof vi.fn>;
};

const restartBanner: RestartBanner = {
  isVisible: false,
  isRestartRequired: false,
  isPolling: false,
  isSuccessful: false,
  isFailed: false,
  operationId: null,
  status: null,
  failureReason: undefined,
  activeSessionCount: 0,
  trigger: vi.fn(),
  isTriggerPending: false,
  triggerError: null,
  dismiss: vi.fn(),
};

let pageState: PageState;

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-mcp-page", () => ({
  useMcpPage: () => pageState,
}));

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: { mcp_servers: [filesystemEntry, githubEntry] },
    servers: [filesystemEntry, githubEntry],
    counts: { total: 2, shadowed: 1 },
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    activeScope: "workspace",
    activeWorkspace: polybotWorkspace,
    activeWorkspaceId: polybotWorkspace.id,
    selectScope: vi.fn(),
    editor: { mode: "closed" },
    editorIsValid: false,
    editorAvailableTargets: ["auto", "config", "sidecar"],
    editorError: null,
    editorWarnings: undefined,
    editorIsSaving: false,
    openCreate: vi.fn(),
    openEdit: vi.fn(),
    closeEditor: vi.fn(),
    updateDraft: vi.fn(),
    setEditorTarget: vi.fn(),
    saveEditor: vi.fn(),
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
  };
}

beforeEach(() => {
  pageState = makeState();
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../mcp";

const MCPPage = routeComponent(Route);

describe("MCPPage", () => {
  it("Should render loading state", () => {
    pageState = makeState({ isLoading: true, envelope: null, servers: [] });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-loading")).toBeInTheDocument();
  });

  it("Should render the error state with a message", () => {
    pageState = makeState({
      envelope: null,
      servers: [],
      error: new Error("nope"),
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-error")).toHaveTextContent("nope");
  });

  it("Should render Workspace and Global scope pills in the topbar", () => {
    render(<MCPPage />);
    expect(screen.getByTestId("mcp-scope-workspace")).toHaveAttribute("data-active", "true");
    expect(screen.getByTestId("mcp-scope-global")).toHaveAttribute("data-active", "false");
  });

  it("Should render the servers table with env/args counts and source metadata", () => {
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-total")).toHaveTextContent("2 servers");
    expect(screen.getByTestId("settings-page-mcp-servers-shadowed-total")).toHaveTextContent(
      "1 shadowed sources"
    );
    expect(screen.getByTestId("settings-page-mcp-servers-row-filesystem-env")).toHaveTextContent(
      "0"
    );
    expect(screen.getByTestId("settings-page-mcp-servers-row-filesystem-args")).toHaveTextContent(
      "1"
    );
    expect(
      screen.getByTestId("settings-page-mcp-servers-row-filesystem-source-effective")
    ).toHaveTextContent("MCP.JSON");
    expect(
      screen.getByTestId("settings-page-mcp-servers-row-filesystem-source-shadowed")
    ).toHaveTextContent("CONFIG");
    expect(screen.getByTestId("settings-page-mcp-servers-row-github-env")).toHaveTextContent("1");
  });

  it("Should wire create, edit, and delete triggers", () => {
    render(<MCPPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-create"));
    expect(pageState.openCreate).toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-filesystem-edit"));
    expect(pageState.openEdit).toHaveBeenCalledWith(filesystemEntry);

    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-filesystem-delete"));
    expect(pageState.openDelete).toHaveBeenCalledWith(filesystemEntry);
  });

  it("Should switch scope on pill click", () => {
    render(<MCPPage />);
    fireEvent.click(screen.getByTestId("mcp-scope-global"));
    expect(pageState.selectScope).toHaveBeenCalledWith("global");
  });

  it("Should render workspace-scoped header and status when workspace scope is active", () => {
    pageState = makeState({
      activeScope: "workspace",
      activeWorkspace: polybotWorkspace,
      activeWorkspaceId: polybotWorkspace.id,
      servers: [],
      envelope: { mcp_servers: [] },
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-scope-label")).toHaveTextContent(
      "polybot"
    );
    const empty = screen.getByTestId("settings-page-mcp-servers-empty");
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No MCP servers configured");
  });

  it("Should render the Empty card when the global catalog is empty", () => {
    pageState = makeState({
      activeScope: "global",
      servers: [],
      envelope: { mcp_servers: [] },
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPPage />);
    const empty = screen.getByTestId("settings-page-mcp-servers-empty");
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No MCP servers configured");
  });

  it("Should render each row with a success StatusDot and name", () => {
    render(<MCPPage />);
    const dot = screen.getByTestId("settings-page-mcp-servers-row-filesystem-status");
    expect(dot).toHaveAttribute("data-slot", "pill-dot");
    expect(dot).toHaveAttribute("data-tone", "configured");
  });

  it("Should render the create editor with target selector defaulted to auto", () => {
    pageState = makeState({
      editor: {
        mode: "create",
        draft: { name: "", command: "", args: [], env: [] },
        target: "auto",
      },
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-mcp-servers-editor-title")).toHaveTextContent(
      "Add MCP server"
    );
    expect(screen.getByTestId("settings-mcp-servers-editor-target-input")).toHaveValue("auto");
  });

  it("Should fire setEditorTarget when operator changes the target selector", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "filesystem",
        draft: {
          name: "filesystem",
          command: "npx fs",
          args: ["~/Dev"],
          env: [],
        },
        entry: filesystemEntry,
        target: "auto",
      },
      editorAvailableTargets: ["auto", "config", "sidecar"],
    });
    render(<MCPPage />);
    fireEvent.change(screen.getByTestId("settings-mcp-servers-editor-target-input"), {
      target: { value: "config" },
    });
    expect(pageState.setEditorTarget).toHaveBeenCalledWith("config");
  });

  it("Should render available_targets as badges in the editor for an existing entry", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "filesystem",
        draft: {
          name: "filesystem",
          command: "npx fs",
          args: [],
          env: [],
        },
        entry: filesystemEntry,
        target: "auto",
      },
    });
    render(<MCPPage />);
    const container = screen.getByTestId("settings-mcp-servers-editor-available-targets");
    expect(container).toHaveTextContent("GLOBAL MCP");
    expect(container).toHaveTextContent("GLOBAL CFG");
  });

  it("Should explain fallback behavior in the delete dialog when shadowed sources exist", () => {
    pageState = makeState({
      deleteTarget: { mode: "open", entry: filesystemEntry, target: "auto" },
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-mcp-servers-delete-shadowed")).toHaveTextContent(
      "After delete, this becomes effective"
    );
    expect(screen.getByTestId("settings-mcp-servers-delete-target-input")).toHaveValue("auto");
  });

  it("Should note no-shadowed state when the current definition is the only source", () => {
    pageState = makeState({
      deleteTarget: { mode: "open", entry: githubEntry, target: "auto" },
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-mcp-servers-delete-no-shadowed")).toBeInTheDocument();
  });

  it("Should render the saved-action banner through Alert with role=status", () => {
    pageState = makeState({
      lastAction: {
        kind: "saved",
        name: "filesystem",
        result: { restart_required: false, write_target: "global-config" },
      },
    });
    render(<MCPPage />);
    const banner = screen.getByTestId("settings-page-mcp-servers-action-result");
    expect(banner).toHaveAttribute("role", "status");
    expect(banner).toHaveAttribute("data-kind", "saved");
    expect(banner).toHaveTextContent('Saved "filesystem"');
    expect(banner).toHaveTextContent("persisted to GLOBAL CFG");
  });

  it("Should render missing-workspace empty state when workspace scope has no active workspace", () => {
    pageState = makeState({
      activeScope: "workspace",
      activeWorkspace: null,
      activeWorkspaceId: null,
      envelope: null,
      servers: [],
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-workspace-guard")).toBeInTheDocument();
  });
});
