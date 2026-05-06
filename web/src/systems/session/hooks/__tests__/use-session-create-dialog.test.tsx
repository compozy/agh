import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "@/systems/agent";
import type { WorkspaceDetailPayload, WorkspacePayload } from "@/systems/workspace";

import type { SessionPayload } from "../../types";
import { useSessionCreateDialog } from "../use-session-create-dialog";

const {
  mockNavigate,
  mockMutateAsync,
  mockToastError,
  mockUseCreateSessionPending,
  mockWorkspaceQuery,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
  mockMutateAsync: vi.fn<(input: unknown) => Promise<SessionPayload>>(),
  mockToastError: vi.fn(),
  mockUseCreateSessionPending: { current: false as boolean },
  mockWorkspaceQuery: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("sonner", () => ({
  toast: {
    error: mockToastError,
  },
}));

vi.mock("@/systems/workspace", async () => {
  const actual = await vi.importActual<typeof import("@/systems/workspace")>("@/systems/workspace");

  return {
    ...actual,
    useWorkspace: (workspaceId: string, options?: { enabled?: boolean }) =>
      mockWorkspaceQuery(workspaceId, options),
  };
});

vi.mock("../use-session-actions", () => ({
  useCreateSession: () => ({
    mutateAsync: mockMutateAsync,
    isPending: mockUseCreateSessionPending.current,
  }),
}));

const activeWorkspace: WorkspacePayload = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: [],
  name: "alpha",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:00Z",
};

const agents: AgentPayload[] = [
  { name: "claude-agent", provider: "claude", prompt: "help" },
  { name: "codex-agent", provider: "codex", prompt: "code" },
];

const createdSession: SessionPayload = {
  id: "sess-new",
  agent_name: "codex-agent",
  provider: "codex",
  workspace_id: "ws_alpha",
  workspace_path: "/workspace/alpha",
  state: "active",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:01Z",
};

let workspaceQueryResult: {
  data: WorkspaceDetailPayload | undefined;
  isLoading: boolean;
  error: Error | null;
};

describe("useSessionCreateDialog", () => {
  beforeEach(() => {
    mockNavigate.mockReset();
    mockNavigate.mockResolvedValue(undefined);
    mockMutateAsync.mockReset();
    mockMutateAsync.mockResolvedValue(createdSession);
    mockToastError.mockReset();
    mockWorkspaceQuery.mockReset();
    mockUseCreateSessionPending.current = false;

    workspaceQueryResult = {
      data: {
        workspace: activeWorkspace,
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      error: null,
    };

    mockWorkspaceQuery.mockImplementation(() => workspaceQueryResult);
  });

  it("derives the default provider once workspace providers arrive after opening", async () => {
    workspaceQueryResult = {
      data: {
        workspace: activeWorkspace,
        providers: [],
      },
      isLoading: true,
      error: null,
    };

    const { result, rerender } = renderHook(() =>
      useSessionCreateDialog({ agents, activeWorkspace })
    );

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    expect(result.current.selectedAgentName).toBe("codex-agent");
    expect(result.current.selectedProvider).toBe("");

    workspaceQueryResult = {
      data: {
        workspace: activeWorkspace,
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      error: null,
    };

    rerender();

    expect(result.current.selectedProvider).toBe("codex");

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
    });
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name/sessions/$id",
      params: { name: "codex-agent", id: "sess-new" },
    });
  });

  it("clears an explicit provider override when the operator changes agents", () => {
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }));

    act(() => {
      result.current.openForAgent("claude-agent");
    });

    expect(result.current.selectedProvider).toBe("claude");

    act(() => {
      result.current.onProviderChange("gemini");
    });

    expect(result.current.selectedProvider).toBe("gemini");

    act(() => {
      result.current.onAgentChange("codex-agent");
    });

    expect(result.current.selectedAgentName).toBe("codex-agent");
    expect(result.current.selectedProvider).toBe("codex");
  });
});
