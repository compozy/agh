import { fireEvent, render, screen } from "@testing-library/react";
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

type Selection = { scope: "global" } | { scope: "workspace"; workspaceId: string };

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
  selection: Selection;
  selectedWorkspace: WorkspacePayload | null;
  workspaces: WorkspacePayload[];
  workspacesLoading: boolean;
  availableScopes: ("global" | "workspace")[];
  selectGlobal: ReturnType<typeof vi.fn>;
  selectWorkspace: ReturnType<typeof vi.fn>;
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

vi.mock("@/hooks/routes/use-settings-mcp-servers-page", () => ({
  useSettingsMCPServersPage: () => pageState,
}));

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: { mcp_servers: [filesystemEntry, githubEntry] },
    servers: [filesystemEntry, githubEntry],
    counts: { total: 2, shadowed: 1 },
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    selection: { scope: "global" },
    selectedWorkspace: null,
    workspaces: [polybotWorkspace],
    workspacesLoading: false,
    availableScopes: ["global", "workspace"],
    selectGlobal: vi.fn(),
    selectWorkspace: vi.fn(),
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

import { Route } from "./mcp-servers";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const MCPServersPage = (Route as any).component as () => ReactNode;

describe("MCPServersSettingsPage", () => {
  it("renders loading state", () => {
    pageState = makeState({ isLoading: true, envelope: null, servers: [] });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-loading")).toBeInTheDocument();
  });

  it("renders the error state with a message", () => {
    pageState = makeState({
      envelope: null,
      servers: [],
      error: new Error("nope"),
    });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-error")).toHaveTextContent("nope");
  });

  it("renders the scope row with global + workspace chips", () => {
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-scope-global")).toHaveAttribute(
      "data-active",
      "true"
    );
    expect(
      screen.getByTestId("settings-page-mcp-servers-scope-workspace-ws-polybot")
    ).toHaveAttribute("data-active", "false");
  });

  it("renders the servers table with env/args counts and source metadata", () => {
    render(<MCPServersPage />);
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

  it("wires create, edit, and delete triggers", () => {
    render(<MCPServersPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-create"));
    expect(pageState.openCreate).toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-filesystem-edit"));
    expect(pageState.openEdit).toHaveBeenCalledWith(filesystemEntry);

    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-row-filesystem-delete"));
    expect(pageState.openDelete).toHaveBeenCalledWith(filesystemEntry);
  });

  it("switches scope on click and shows workspace-specific label", () => {
    render(<MCPServersPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-scope-workspace-ws-polybot"));
    expect(pageState.selectWorkspace).toHaveBeenCalledWith("ws-polybot");
  });

  it("renders workspace-scoped header and status when workspace scope is active", () => {
    pageState = makeState({
      selection: { scope: "workspace", workspaceId: "ws-polybot" },
      selectedWorkspace: polybotWorkspace,
      servers: [],
      envelope: { mcp_servers: [] },
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-page-mcp-servers-scope-label")).toHaveTextContent(
      "polybot"
    );
    const empty = screen.getByTestId("settings-page-mcp-servers-empty");
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No MCP servers configured");
  });

  it("renders the @agh/ui Empty card when the global catalog is empty", () => {
    pageState = makeState({
      servers: [],
      envelope: { mcp_servers: [] },
      counts: { total: 0, shadowed: 0 },
    });
    render(<MCPServersPage />);
    const empty = screen.getByTestId("settings-page-mcp-servers-empty");
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No MCP servers configured");
  });

  it("renders each row with a success StatusDot and name", () => {
    render(<MCPServersPage />);
    const dot = screen.getByTestId("settings-page-mcp-servers-row-filesystem-status");
    expect(dot).toHaveAttribute("data-slot", "pill-dot");
    expect(dot).toHaveAttribute("data-tone", "configured");
  });

  it("renders the create editor with target selector defaulted to auto", () => {
    pageState = makeState({
      editor: {
        mode: "create",
        draft: { name: "", command: "", args: [], env: [] },
        target: "auto",
      },
    });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-mcp-servers-editor-title")).toHaveTextContent(
      "Add MCP server"
    );
    expect(screen.getByTestId("settings-mcp-servers-editor-target-input")).toHaveValue("auto");
  });

  it("fires setEditorTarget when operator changes the target selector", () => {
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
    render(<MCPServersPage />);
    fireEvent.change(screen.getByTestId("settings-mcp-servers-editor-target-input"), {
      target: { value: "config" },
    });
    expect(pageState.setEditorTarget).toHaveBeenCalledWith("config");
  });

  it("renders available_targets as badges in the editor for an existing entry", () => {
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
    render(<MCPServersPage />);
    const container = screen.getByTestId("settings-mcp-servers-editor-available-targets");
    expect(container).toHaveTextContent("GLOBAL MCP");
    expect(container).toHaveTextContent("GLOBAL CFG");
  });

  it("explains fallback behavior in the delete dialog when shadowed sources exist", () => {
    pageState = makeState({
      deleteTarget: { mode: "open", entry: filesystemEntry, target: "auto" },
    });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-mcp-servers-delete-shadowed")).toHaveTextContent(
      "After delete, this becomes effective"
    );
    expect(screen.getByTestId("settings-mcp-servers-delete-target-input")).toHaveValue("auto");
  });

  it("notes no-shadowed state when the current definition is the only source", () => {
    pageState = makeState({
      deleteTarget: { mode: "open", entry: githubEntry, target: "auto" },
    });
    render(<MCPServersPage />);
    expect(screen.getByTestId("settings-mcp-servers-delete-no-shadowed")).toBeInTheDocument();
  });

  it("renders the saved-action banner through @agh/ui Alert with role=status", () => {
    pageState = makeState({
      lastAction: {
        kind: "saved",
        name: "filesystem",
        result: { restart_required: false, write_target: "global-config" },
      },
    });
    render(<MCPServersPage />);
    const banner = screen.getByTestId("settings-page-mcp-servers-action-result");
    expect(banner).toHaveAttribute("data-slot", "alert");
    expect(banner).toHaveAttribute("role", "status");
  });

  it("shows the last action banner for saved and deleted actions", () => {
    pageState = makeState({
      lastAction: {
        kind: "deleted",
        name: "filesystem",
        result: { restart_required: true, write_target: "global-mcp-sidecar" },
        remainingShadowed: 1,
      },
    });
    render(<MCPServersPage />);
    const banner = screen.getByTestId("settings-page-mcp-servers-action-result");
    expect(banner).toHaveTextContent('Deleted "filesystem"');
    expect(banner).toHaveTextContent("1 shadowed source may become effective on reload");
  });

  it("updates the draft args list via the args editor controls", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "filesystem",
        draft: {
          name: "filesystem",
          command: "npx fs",
          args: ["~/Dev", "--flag"],
          env: [],
        },
        entry: filesystemEntry,
        target: "auto",
      },
    });
    render(<MCPServersPage />);
    fireEvent.change(screen.getByTestId("settings-mcp-servers-editor-args-input-0"), {
      target: { value: "~/Projects" },
    });
    expect(pageState.updateDraft).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("settings-mcp-servers-editor-args-remove-1"));
    expect(pageState.updateDraft).toHaveBeenCalledTimes(2);
    fireEvent.click(screen.getByTestId("settings-mcp-servers-editor-args-add"));
    expect(pageState.updateDraft).toHaveBeenCalledTimes(3);
  });

  it("updates env pairs via the env editor controls", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "github",
        draft: {
          name: "github",
          command: "npx gh",
          args: [],
          env: [{ key: "GITHUB_TOKEN", value: "secret" }],
        },
        entry: githubEntry,
        target: "auto",
      },
    });
    render(<MCPServersPage />);
    fireEvent.change(screen.getByTestId("settings-mcp-servers-editor-env-key-0"), {
      target: { value: "GITHUB_TOKEN_V2" },
    });
    fireEvent.change(screen.getByTestId("settings-mcp-servers-editor-env-value-0"), {
      target: { value: "rotated" },
    });
    fireEvent.click(screen.getByTestId("settings-mcp-servers-editor-env-remove-0"));
    fireEvent.click(screen.getByTestId("settings-mcp-servers-editor-env-add"));
    expect(pageState.updateDraft).toHaveBeenCalledTimes(4);
  });

  it("changes the delete target via the select control", () => {
    pageState = makeState({
      deleteTarget: { mode: "open", entry: filesystemEntry, target: "auto" },
      deleteAvailableTargets: ["auto", "config", "sidecar"],
    });
    render(<MCPServersPage />);
    fireEvent.change(screen.getByTestId("settings-mcp-servers-delete-target-input"), {
      target: { value: "config" },
    });
    expect(pageState.setDeleteTargetKind).toHaveBeenCalledWith("config");
  });

  it("clicks the global scope chip when switching back from workspace", () => {
    pageState = makeState({
      selection: { scope: "workspace", workspaceId: "ws-polybot" },
      selectedWorkspace: polybotWorkspace,
    });
    render(<MCPServersPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-scope-global"));
    expect(pageState.selectGlobal).toHaveBeenCalled();
  });

  it("shows 'no workspaces yet' when the workspace scope has no workspaces", () => {
    pageState = makeState({ workspaces: [], workspacesLoading: false });
    render(<MCPServersPage />);
    expect(
      screen.getByTestId("settings-page-mcp-servers-scope-workspace-empty")
    ).toBeInTheDocument();
  });

  it("renders the saved-action banner with write_target metadata", () => {
    pageState = makeState({
      lastAction: {
        kind: "saved",
        name: "filesystem",
        result: { restart_required: false, write_target: "global-config" },
      },
    });
    render(<MCPServersPage />);
    const banner = screen.getByTestId("settings-page-mcp-servers-action-result");
    expect(banner).toHaveTextContent('Saved "filesystem"');
    expect(banner).toHaveTextContent("persisted to GLOBAL CFG");
    expect(banner).toHaveTextContent("applied immediately");
  });

  it("dismisses the action banner", () => {
    pageState = makeState({
      lastAction: {
        kind: "saved",
        name: "filesystem",
        result: { restart_required: false },
      },
    });
    render(<MCPServersPage />);
    fireEvent.click(screen.getByTestId("settings-page-mcp-servers-action-result-dismiss"));
    expect(pageState.dismissLastAction).toHaveBeenCalled();
  });
});
