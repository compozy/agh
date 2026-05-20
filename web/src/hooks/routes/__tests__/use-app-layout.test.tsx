import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const {
  mockNavigate,
  mockMutateAsync,
  mockSetActiveWorkspaceId,
  mockToastError,
  mockWorkspaceQuery,
  mockOpenAgentCreate,
  mockUseCreateSessionPending,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
  mockMutateAsync: vi.fn<(input: unknown) => Promise<{ id: string; agent_name: string }>>(),
  mockSetActiveWorkspaceId: vi.fn<(workspaceId: string | null) => void>(),
  mockToastError: vi.fn(),
  mockWorkspaceQuery: vi.fn(),
  mockOpenAgentCreate: vi.fn<() => void>(),
  mockUseCreateSessionPending: { current: false as boolean },
}));

let mockActiveWorkspaceId: string | null = "ws_alpha";
let mockAgents: Array<{ name: string; provider: string; prompt: string }> = [
  { name: "claude-agent", provider: "claude", prompt: "help" },
  { name: "codex-agent", provider: "codex", prompt: "code" },
];
let mockAgentsLoading = false;
let mockAgentsError = false;

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

vi.mock("@/systems/status", () => ({
  useDaemonHealth: () => ({
    health: { version: "0.1.0" },
    connectionStatus: "connected",
  }),
}));

vi.mock("@/systems/agent", () => ({
  useAgents: () => ({
    data: mockAgents,
    isLoading: mockAgentsLoading,
    isError: mockAgentsError,
  }),
  useAgentCreateDialog: () => ({
    open: false,
    draft: {
      scope: "workspace",
      name: "",
      categoryPath: "",
      provider: "",
      model: "",
      command: "",
      prompt: "",
      permissions: "",
      tools: [],
      toolsets: [],
      denyTools: [],
      disabledSkills: [],
    },
    providerOptions: [],
    providersLoading: false,
    providersError: null,
    modelOptions: [],
    modelCatalogLoading: false,
    modelCatalogError: null,
    submitError: null,
    isSubmitting: false,
    hasActiveWorkspace: mockActiveWorkspaceId !== null,
    workspaceName: mockActiveWorkspaceId === null ? null : "alpha",
    openDialog: mockOpenAgentCreate,
    onDraftChange: vi.fn(),
    onOpenChange: vi.fn(),
    onSubmit: vi.fn(),
  }),
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useCreateSession: () => ({
    mutateAsync: mockMutateAsync,
    isPending: mockUseCreateSessionPending.current,
  }),
}));

vi.mock("@/systems/model-catalog", async () => {
  const actual =
    await vi.importActual<typeof import("@/systems/model-catalog")>("@/systems/model-catalog");
  return {
    ...actual,
    useProviderModels: () => ({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    }),
    useProviderModelStatus: () => ({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    }),
    useRefreshProviderModels: () => ({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
      error: null,
    }),
  };
});

vi.mock("@/systems/session", async () => {
  const useSessionCreateDialogModule = await vi.importActual<
    typeof import("@/systems/session/hooks/use-session-create-dialog")
  >("@/systems/session/hooks/use-session-create-dialog");
  return {
    useCreateSession: () => ({
      mutateAsync: mockMutateAsync,
      isPending: mockUseCreateSessionPending.current,
    }),
    useSessions: () => ({ data: [] }),
    useSessionCreateDialog: useSessionCreateDialogModule.useSessionCreateDialog,
  };
});

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
    setActiveWorkspaceId: mockSetActiveWorkspaceId,
    isLoading: false,
    isError: false,
  }),
  useWorkspace: (workspaceId: string, options?: { enabled?: boolean }) =>
    mockWorkspaceQuery(workspaceId, options),
}));

import { useAppLayout } from "../use-app-layout";

describe("useAppLayout", () => {
  beforeEach(() => {
    mockActiveWorkspaceId = "ws_alpha";
    mockAgents = [
      { name: "claude-agent", provider: "claude", prompt: "help" },
      { name: "codex-agent", provider: "codex", prompt: "code" },
    ];
    mockAgentsLoading = false;
    mockAgentsError = false;
    mockNavigate.mockReset();
    mockMutateAsync.mockReset();
    mockSetActiveWorkspaceId.mockReset();
    mockToastError.mockReset();
    mockWorkspaceQuery.mockReset();
    mockOpenAgentCreate.mockReset();
    mockUseCreateSessionPending.current = false;
    mockWorkspaceQuery.mockReturnValue({
      data: {
        workspace: {
          id: "ws_alpha",
          root_dir: "/workspace/alpha",
          add_dirs: [],
          name: "alpha",
          created_at: "2026-04-20T10:00:00Z",
          updated_at: "2026-04-20T10:00:00Z",
        },
        agents: undefined,
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      error: null,
    });
  });

  it("opens the create-session dialog instead of creating a session immediately", () => {
    const { result } = renderHook(() => useAppLayout());

    expect(result.current.sessionCreate.open).toBe(false);

    act(() => {
      result.current.handleNewSession("claude-agent");
    });

    expect(mockMutateAsync).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
    expect(result.current.sessionCreate.open).toBe(true);
    expect(result.current.sessionCreate.selectedAgentName).toBe("claude-agent");
    expect(result.current.sessionCreate.selectedProvider).toBe("claude");
    expect(result.current.sessionCreate.providerOptions.map(option => option.name)).toEqual([
      "claude",
      "codex",
      "gemini",
    ]);
  });

  it("preselects the chosen agent default provider when opening for a different agent", () => {
    const { result } = renderHook(() => useAppLayout());

    act(() => {
      result.current.handleNewSession("codex-agent");
    });

    expect(result.current.sessionCreate.selectedAgentName).toBe("codex-agent");
    expect(result.current.sessionCreate.selectedProvider).toBe("codex");
  });

  it("uses workspace-scoped agents when the active workspace detail provides them", () => {
    mockWorkspaceQuery.mockReturnValue({
      data: {
        workspace: {
          id: "ws_alpha",
          root_dir: "/workspace/alpha",
          add_dirs: [],
          name: "alpha",
          created_at: "2026-04-20T10:00:00Z",
          updated_at: "2026-04-20T10:00:00Z",
        },
        agents: [{ name: "workspace-review", provider: "gemini", prompt: "review" }],
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      isError: false,
      error: null,
    });

    const { result } = renderHook(() => useAppLayout());

    expect(result.current.agents?.map(agent => agent.name)).toEqual(["workspace-review"]);

    act(() => {
      result.current.handleNewSession("workspace-review");
    });

    expect(result.current.sessionCreate.selectedAgentName).toBe("workspace-review");
    expect(result.current.sessionCreate.selectedProvider).toBe("gemini");
  });

  it("ignores global agent loading when workspace-scoped agents are already present", () => {
    mockAgentsLoading = true;
    mockWorkspaceQuery.mockReturnValue({
      data: {
        workspace: {
          id: "ws_alpha",
          root_dir: "/workspace/alpha",
          add_dirs: [],
          name: "alpha",
          created_at: "2026-04-20T10:00:00Z",
          updated_at: "2026-04-20T10:00:00Z",
        },
        agents: [{ name: "workspace-review", provider: "gemini", prompt: "review" }],
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      isError: false,
      error: null,
    });

    const { result } = renderHook(() => useAppLayout());

    expect(result.current.agentsLoading).toBe(false);
    expect(result.current.agents?.map(agent => agent.name)).toEqual(["workspace-review"]);
  });

  it("ignores global agent errors when workspace-scoped agents are already present", () => {
    mockAgentsError = true;
    mockWorkspaceQuery.mockReturnValue({
      data: {
        workspace: {
          id: "ws_alpha",
          root_dir: "/workspace/alpha",
          add_dirs: [],
          name: "alpha",
          created_at: "2026-04-20T10:00:00Z",
          updated_at: "2026-04-20T10:00:00Z",
        },
        agents: [{ name: "workspace-review", provider: "gemini", prompt: "review" }],
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      isError: false,
      error: null,
    });

    const { result } = renderHook(() => useAppLayout());

    expect(result.current.agentsError).toBe(false);
    expect(result.current.agents?.map(agent => agent.name)).toEqual(["workspace-review"]);
  });

  it("submits the dialog with agent name, workspace, and selected provider", async () => {
    mockMutateAsync.mockResolvedValue({ id: "sess-new", agent_name: "claude-agent" });
    mockNavigate.mockResolvedValue(undefined);

    const { result } = renderHook(() => useAppLayout());

    act(() => {
      result.current.handleNewSession("claude-agent");
    });

    act(() => {
      result.current.sessionCreate.onProviderChange("gemini");
    });

    await act(async () => {
      await result.current.sessionCreate.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "claude-agent",
      workspace: "ws_alpha",
      provider: "gemini",
    });
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name/sessions/$id",
      params: { name: "claude-agent", id: "sess-new" },
    });
    expect(result.current.sessionCreate.open).toBe(false);
  });

  it("keeps the dialog open and surfaces submitError when creation fails", async () => {
    mockMutateAsync.mockRejectedValue(new Error("Failed to create"));

    const { result } = renderHook(() => useAppLayout());

    act(() => {
      result.current.handleNewSession("claude-agent");
    });

    await act(async () => {
      await result.current.sessionCreate.submit();
    });

    expect(result.current.sessionCreate.open).toBe(true);
    expect(result.current.sessionCreate.submitError).toBe("Failed to create");
    expect(mockToastError).toHaveBeenCalledWith("Failed to create");
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("refuses to open the dialog when there is no active workspace", () => {
    mockActiveWorkspaceId = null;

    const { result } = renderHook(() => useAppLayout());

    act(() => {
      result.current.handleNewSession("claude-agent");
    });

    expect(result.current.sessionCreate.open).toBe(false);
    expect(mockToastError).toHaveBeenCalledWith(
      "Select an active workspace before starting a session."
    );
  });

  it("exposes the agent creation dialog controller", () => {
    const { result } = renderHook(() => useAppLayout());

    act(() => {
      result.current.agentCreate.openDialog();
    });

    expect(mockOpenAgentCreate).toHaveBeenCalledOnce();
  });
});
