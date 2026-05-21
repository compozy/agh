import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "@/systems/agent";
import type {
  ProviderModelsListResponse,
  ProviderModelsRefreshResponse,
} from "@/systems/model-catalog";
import type { WorkspaceDetailPayload, WorkspacePayload } from "@/systems/workspace";

import type { SessionPayload } from "../../types";
import { useSessionCreateDialog } from "../use-session-create-dialog";

const {
  mockNavigate,
  mockMutateAsync,
  mockToastError,
  mockUseCreateSessionPending,
  mockWorkspaceQuery,
  mockListProviderModels,
  mockRefreshProviderModels,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
  mockMutateAsync: vi.fn<(input: unknown) => Promise<SessionPayload>>(),
  mockToastError: vi.fn(),
  mockUseCreateSessionPending: { current: false as boolean },
  mockWorkspaceQuery: vi.fn(),
  mockListProviderModels: vi.fn<(input: unknown) => Promise<ProviderModelsListResponse>>(),
  mockRefreshProviderModels: vi.fn<(input: unknown) => Promise<ProviderModelsRefreshResponse>>(),
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

vi.mock("@/systems/model-catalog/adapters/model-catalog-api", async () => {
  const actual = await vi.importActual<
    typeof import("@/systems/model-catalog/adapters/model-catalog-api")
  >("@/systems/model-catalog/adapters/model-catalog-api");
  return {
    ...actual,
    listProviderModels: (...args: unknown[]) => mockListProviderModels(args[0]),
    refreshProviderModels: (...args: unknown[]) => mockRefreshProviderModels(args[0]),
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

const agentsWithDefaultModel: AgentPayload[] = [
  { name: "claude-agent", provider: "claude", prompt: "help" },
  { name: "codex-agent", provider: "codex", model: "gpt-5.5", prompt: "code" },
];

const createdSession: SessionPayload = {
  id: "sess-new",
  agent_name: "codex-agent",
  provider: "codex",
  workspace_id: "ws_alpha",
  workspace_path: "/workspace/alpha",
  state: "active",
  badge: "idle",
  attachable: true,
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:01Z",
};

let workspaceQueryResult: {
  data: WorkspaceDetailPayload | undefined;
  isLoading: boolean;
  error: Error | null;
};

const codexCatalog: ProviderModelsListResponse = {
  models: [
    {
      provider_id: "codex",
      model_id: "gpt-5.4",
      display_name: "GPT-5.4",
      availability_state: "available_live",
      available: true,
      stale: false,
      refreshed_at: "2026-05-07T10:00:00Z",
      sources: [
        {
          source_id: "config",
          source_kind: "config",
          priority: 120,
          refreshed_at: "2026-05-07T10:00:00Z",
          stale: false,
        },
      ],
      supports_reasoning: true,
      reasoning_efforts: ["low", "medium", "high"],
      default_reasoning_effort: "medium",
    },
    {
      provider_id: "codex",
      model_id: "gpt-5.4-mini",
      display_name: "GPT-5.4 Mini",
      availability_state: "available_stale",
      available: true,
      stale: true,
      refreshed_at: "2026-05-06T10:00:00Z",
      sources: [
        {
          source_id: "models_dev",
          source_kind: "models_dev",
          priority: 50,
          refreshed_at: "2026-05-06T10:00:00Z",
          stale: true,
        },
      ],
      supports_reasoning: false,
    },
  ],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

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
    mockListProviderModels.mockReset();
    mockListProviderModels.mockResolvedValue(codexCatalog);
    mockRefreshProviderModels.mockReset();
    mockRefreshProviderModels.mockResolvedValue({
      sources: [
        {
          source_id: "models_dev",
          source_kind: "models_dev",
          priority: 50,
          provider_id: "codex",
          refresh_state: "succeeded",
          row_count: 2,
          stale: false,
        },
      ],
    });
  });

  it("Should derive the default provider once workspace providers arrive after opening", async () => {
    workspaceQueryResult = {
      data: {
        workspace: activeWorkspace,
        providers: [],
      },
      isLoading: true,
      error: null,
    };

    const { wrapper } = createWrapper();
    const { result, rerender } = renderHook(
      () => useSessionCreateDialog({ agents, activeWorkspace }),
      { wrapper }
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

  it("Should clear an explicit provider override when the operator changes agents", () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

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

  it("Should expose deduped catalog models for the selected provider", async () => {
    mockListProviderModels.mockResolvedValueOnce({
      models: [
        codexCatalog.models[0],
        codexCatalog.models[1],
        codexCatalog.models[0],
      ] as ProviderModelsListResponse["models"],
    });
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.modelOptions).toHaveLength(2);
    });
    expect(result.current.modelOptions.map(option => option.id)).toEqual([
      "gpt-5.4",
      "gpt-5.4-mini",
    ]);
  });

  it("Should keep manual model entry available when the catalog is empty", async () => {
    mockListProviderModels.mockResolvedValueOnce({ models: [] });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.catalogLoading).toBe(false);
    });
    expect(result.current.modelOptions).toEqual([]);

    act(() => {
      result.current.onModelChange("custom-experimental");
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "custom-experimental",
    });
  });

  it("Should expose stale catalog rows without blocking session creation", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.modelOptions).toHaveLength(2);
    });

    expect(result.current.catalogStale).toBe(true);
    const staleOption = result.current.modelOptions.find(option => option.id === "gpt-5.4-mini");
    expect(staleOption?.availabilityState).toBe("available_stale");
    const liveOption = result.current.modelOptions.find(option => option.id === "gpt-5.4");
    expect(liveOption?.availabilityState).toBe("available_live");

    act(() => {
      result.current.onModelChange("gpt-5.4-mini");
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "gpt-5.4-mini",
    });
  });

  it("Should surface catalog source errors without blocking manual entry", async () => {
    mockListProviderModels.mockReset();
    mockListProviderModels.mockRejectedValue(new Error("catalog upstream failed"));

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(
      () => {
        expect(result.current.catalogError).toContain("catalog upstream failed");
      },
      { timeout: 5000 }
    );
    expect(result.current.modelOptions).toEqual([]);

    act(() => {
      result.current.onModelChange("manual-fallback");
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "manual-fallback",
    });
  });

  it("Should invalidate catalog queries on refresh", async () => {
    const { queryClient, wrapper } = createWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.modelOptions).toHaveLength(2);
    });

    act(() => {
      result.current.refreshCatalog();
    });

    await waitFor(() => {
      expect(mockRefreshProviderModels).toHaveBeenCalledWith({
        providerId: "codex",
        force: true,
      });
    });
    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalled();
    });
  });

  it("Should submit selected model and reasoning overrides only when populated", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.modelOptions.length).toBeGreaterThan(0);
    });

    act(() => {
      result.current.onModelChange("gpt-5.4-mini");
    });
    expect(result.current.reasoningSupported).toBe(false);

    act(() => {
      result.current.onModelChange("gpt-5.4");
    });
    expect(result.current.reasoningSupported).toBe(true);

    act(() => {
      result.current.onReasoningChange("high");
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "gpt-5.4",
      reasoning_effort: "high",
    });
  });

  it("Should submit reasoning for an agent default model without sending a model override", async () => {
    mockListProviderModels.mockResolvedValueOnce({
      models: [
        ...codexCatalog.models,
        {
          provider_id: "codex",
          model_id: "gpt-5.5",
          display_name: "GPT-5.5",
          availability_state: "available_live",
          available: true,
          stale: false,
          refreshed_at: "2026-05-07T10:00:00Z",
          sources: [
            {
              source_id: "models_dev",
              source_kind: "models_dev",
              priority: 50,
              refreshed_at: "2026-05-07T10:00:00Z",
              stale: false,
            },
          ],
          supports_reasoning: true,
          reasoning_efforts: ["minimal", "low", "medium", "high", "xhigh"],
          default_reasoning_effort: "medium",
        },
      ],
    });
    const { wrapper } = createWrapper();
    const { result } = renderHook(
      () => useSessionCreateDialog({ agents: agentsWithDefaultModel, activeWorkspace }),
      { wrapper }
    );

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.reasoningSupported).toBe(true);
    });
    expect(result.current.selectedModel).toBe("");
    expect(result.current.defaultReasoning).toBe("medium");

    act(() => {
      result.current.onReasoningChange("high");
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      reasoning_effort: "high",
    });
  });
});
