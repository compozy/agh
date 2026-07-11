import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "@/systems/agent";
import type { AllModelsListResponse, AllModelsRefreshResponse } from "@/systems/model-catalog";
import type { WorkspaceDetailPayload, WorkspacePayload } from "@/systems/workspace";

import type { SessionPayload } from "../../types";
import { useSessionCreateDialog } from "../use-session-create-dialog";

type ProviderModelPayload = AllModelsListResponse["models"][number];

const visibleCatalogFlags = {
  curated: true,
  deprecated: false,
  featured: false,
  hidden: false,
} satisfies Pick<ProviderModelPayload, "curated" | "deprecated" | "featured" | "hidden">;

const {
  mockNavigate,
  mockMutateAsync,
  mockToastError,
  mockUseCreateSessionPending,
  mockWorkspaceQuery,
  mockListAllModels,
  mockRefreshAllModels,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
  mockMutateAsync: vi.fn<(input: unknown) => Promise<SessionPayload>>(),
  mockToastError: vi.fn(),
  mockUseCreateSessionPending: { current: false as boolean },
  mockWorkspaceQuery: vi.fn(),
  mockListAllModels: vi.fn<(input: unknown) => Promise<AllModelsListResponse>>(),
  mockRefreshAllModels: vi.fn<(input: unknown) => Promise<AllModelsRefreshResponse>>(),
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
    listAllModels: (...args: unknown[]) => mockListAllModels(args[0]),
    refreshAllModels: (...args: unknown[]) => mockRefreshAllModels(args[0]),
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
  available_commands: [],
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:01Z",
};

let workspaceQueryResult: {
  data: WorkspaceDetailPayload | undefined;
  isLoading: boolean;
  error: Error | null;
};

const codexCatalog: AllModelsListResponse = {
  models: [
    {
      ...visibleCatalogFlags,
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
      ...visibleCatalogFlags,
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
    mockListAllModels.mockReset();
    mockListAllModels.mockResolvedValue(codexCatalog);
    mockRefreshAllModels.mockReset();
    mockRefreshAllModels.mockResolvedValue({
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
    expect(result.current.runtimeValue.provider).toBe("");

    workspaceQueryResult = {
      data: {
        workspace: activeWorkspace,
        providers: [{ name: "claude" }, { name: "codex" }, { name: "gemini" }],
      },
      isLoading: false,
      error: null,
    };

    rerender();

    expect(result.current.runtimeValue.provider).toBe("codex");

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

  it("Should map workspace providers onto the runtime rail options", () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    expect(result.current.hasProviderOptions).toBe(true);
    expect(result.current.runtimeProviders.map(option => option.id)).toEqual([
      "claude",
      "codex",
      "gemini",
    ]);
  });

  it("Should clear an explicit provider override when the operator changes agents", () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("claude-agent");
    });

    expect(result.current.runtimeValue.provider).toBe("claude");

    act(() => {
      result.current.onRuntimeChange({ provider: "gemini", model: "", reasoning_effort: "" });
    });

    expect(result.current.runtimeValue.provider).toBe("gemini");

    act(() => {
      result.current.onAgentChange("codex-agent");
    });

    expect(result.current.selectedAgentName).toBe("codex-agent");
    expect(result.current.runtimeValue.provider).toBe("codex");
  });

  it("Should expose deduped runtime models for the selected provider using the all view", async () => {
    mockListAllModels.mockResolvedValueOnce({
      models: [
        codexCatalog.models[0],
        codexCatalog.models[1],
        codexCatalog.models[0],
      ] as AllModelsListResponse["models"],
    });
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.runtimeModels).toHaveLength(2);
    });
    expect(result.current.runtimeModels.map(option => option.id)).toEqual([
      "gpt-5.4",
      "gpt-5.4-mini",
    ]);
    // Single aggregate `view=all` request across providers — never per-provider.
    expect(mockListAllModels).toHaveBeenCalledWith(
      expect.objectContaining({ includeStale: true, view: "all" })
    );
  });

  it("Should keep manual model entry available when the catalog is empty", async () => {
    mockListAllModels.mockResolvedValueOnce({ models: [] });

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
    expect(result.current.runtimeModels).toEqual([]);

    act(() => {
      result.current.onRuntimeChange({
        provider: "codex",
        model: "custom-experimental",
        reasoning_effort: "",
      });
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
      expect(result.current.runtimeModels).toHaveLength(2);
    });

    expect(result.current.catalogStale).toBe(true);
    const staleOption = result.current.runtimeModels.find(option => option.id === "gpt-5.4-mini");
    expect(staleOption?.availability).toBe("stale");
    const liveOption = result.current.runtimeModels.find(option => option.id === "gpt-5.4");
    expect(liveOption?.availability).toBe("live");

    act(() => {
      result.current.onRuntimeChange({
        provider: "codex",
        model: "gpt-5.4-mini",
        reasoning_effort: "",
      });
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
    mockListAllModels.mockReset();
    mockListAllModels.mockRejectedValue(new Error("catalog upstream failed"));

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
    expect(result.current.runtimeModels).toEqual([]);

    act(() => {
      result.current.onRuntimeChange({
        provider: "codex",
        model: "manual-fallback",
        reasoning_effort: "",
      });
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
      expect(result.current.runtimeModels).toHaveLength(2);
    });

    act(() => {
      result.current.refreshCatalog();
    });

    await waitFor(() => {
      // The refresh affordance truthfully refreshes the whole catalog.
      expect(mockRefreshAllModels).toHaveBeenCalledWith({});
    });
    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalled();
    });
  });

  it("Should thread the model and reasoning override when the model supports reasoning", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.runtimeModels.length).toBeGreaterThan(0);
    });

    act(() => {
      result.current.onRuntimeChange({
        provider: "codex",
        model: "gpt-5.4",
        reasoning_effort: "high",
      });
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenLastCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "gpt-5.4",
      reasoning_effort: "high",
    });
  });

  it("Should omit the reasoning override when the selected model lacks reasoning support", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSessionCreateDialog({ agents, activeWorkspace }), {
      wrapper,
    });

    act(() => {
      result.current.openForAgent("codex-agent");
    });

    await waitFor(() => {
      expect(result.current.runtimeModels.length).toBeGreaterThan(0);
    });

    act(() => {
      result.current.onRuntimeChange({
        provider: "codex",
        model: "gpt-5.4-mini",
        reasoning_effort: "high",
      });
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenLastCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      model: "gpt-5.4-mini",
    });
  });

  it("Should submit reasoning for an agent default model without sending a model override", async () => {
    mockListAllModels.mockResolvedValueOnce({
      models: [
        ...codexCatalog.models,
        {
          ...visibleCatalogFlags,
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
      expect(result.current.runtimeModels.some(option => option.id === "gpt-5.5")).toBe(true);
    });
    // The inherited agent-default model is RENDERED so its reasoning is reachable…
    expect(result.current.runtimeValue.model).toBe("gpt-5.5");

    act(() => {
      // …and the selector emits the effective model with the chosen effort.
      result.current.onRuntimeChange({
        provider: "codex",
        model: "gpt-5.5",
        reasoning_effort: "high",
      });
    });

    await act(async () => {
      await result.current.submit();
    });

    // …but the POST omits `model` (still inherited) while sending reasoning_effort.
    expect(mockMutateAsync).toHaveBeenCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "codex",
      reasoning_effort: "high",
    });
  });

  it("Should send the model when the same model id is chosen under a different provider", async () => {
    // The same canonical id exists under BOTH codex and claude. The agent default
    // is codex/gpt-5.5; picking gpt-5.5 under CLAUDE must ship the model with the
    // chosen provider (compound identity), never silently inherit the codex default.
    const sharedModel = {
      ...visibleCatalogFlags,
      model_id: "gpt-5.5",
      display_name: "GPT-5.5",
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
      supports_reasoning: false,
    };
    mockListAllModels.mockResolvedValueOnce({
      models: [
        ...codexCatalog.models,
        { ...sharedModel, provider_id: "codex" },
        { ...sharedModel, provider_id: "claude" },
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
      expect(
        result.current.runtimeModels.some(
          option => option.id === "gpt-5.5" && option.provider === "claude"
        )
      ).toBe(true);
    });

    act(() => {
      result.current.onRuntimeChange({
        provider: "claude",
        model: "gpt-5.5",
        reasoning_effort: "",
      });
    });

    await act(async () => {
      await result.current.submit();
    });

    expect(mockMutateAsync).toHaveBeenLastCalledWith({
      agent_name: "codex-agent",
      workspace: "ws_alpha",
      provider: "claude",
      model: "gpt-5.5",
    });
  });
});
