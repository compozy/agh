import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockCreateAgent, mockNavigate, mockSettingsProviders, mockToastError, mockProviderModels } =
  vi.hoisted(() => ({
    mockCreateAgent: vi.fn(),
    mockNavigate: vi.fn(),
    mockSettingsProviders: {
      data: {
        providers: [
          {
            name: "claude",
            settings: {
              display_name: "Claude Code",
              harness: "acp",
              runtime_provider: "claude",
            },
          },
        ],
      } as
        | {
            providers: Array<{
              name: string;
              settings: {
                display_name?: string;
                harness?: string;
                runtime_provider?: string;
              };
            }>;
          }
        | undefined,
      isLoading: false,
      isFetching: false,
      error: null as Error | null,
    },
    mockToastError: vi.fn(),
    mockProviderModels: {
      data: { models: [{ model_id: "gpt-5.4" }, { model_id: "gpt-5.4-mini" }] },
      isLoading: false,
      isFetching: false,
      error: null as Error | null,
    },
  }));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("sonner", () => ({
  toast: {
    error: mockToastError,
  },
}));

vi.mock("@/systems/settings", () => ({
  useSettingsProviders: () => mockSettingsProviders,
}));

vi.mock("@/systems/model-catalog", () => ({
  useProviderModels: () => mockProviderModels,
}));

vi.mock("../use-agents", () => ({
  useCreateAgent: () => ({
    mutateAsync: mockCreateAgent,
    isPending: false,
  }),
}));

import { useAgentCreateDialog } from "../use-agent-create-dialog";

const activeWorkspace = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: [],
  name: "alpha",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:00Z",
};

const workspaceProviders = [
  {
    name: "codex",
    display_name: "Codex",
    harness: "acp",
    runtime_provider: "codex",
  },
];

function renderAgentCreateDialog() {
  return renderHook(() =>
    useAgentCreateDialog({
      activeWorkspace,
      workspaceProviders,
      workspaceProvidersError: null,
      workspaceProvidersLoading: false,
    })
  );
}

describe("useAgentCreateDialog", () => {
  beforeEach(() => {
    mockCreateAgent.mockReset();
    mockCreateAgent.mockResolvedValue({
      name: "release-captain",
      provider: "codex",
      prompt: "Own release readiness.",
    });
    mockNavigate.mockReset();
    mockNavigate.mockResolvedValue(undefined);
    mockToastError.mockReset();
    mockSettingsProviders.data = {
      providers: [
        {
          name: "claude",
          settings: {
            display_name: "Claude Code",
            harness: "acp",
            runtime_provider: "claude",
          },
        },
      ],
    };
    mockSettingsProviders.isLoading = false;
    mockSettingsProviders.isFetching = false;
    mockSettingsProviders.error = null;
    mockProviderModels.data = {
      models: [{ model_id: "gpt-5.4" }, { model_id: "gpt-5.4-mini" }],
    };
    mockProviderModels.isLoading = false;
    mockProviderModels.isFetching = false;
    mockProviderModels.error = null;
  });

  it("defaults creation to the active workspace", () => {
    const { result } = renderAgentCreateDialog();

    act(() => {
      result.current.openDialog();
    });

    expect(result.current.open).toBe(true);
    expect(result.current.draft.scope).toBe("workspace");
    expect(result.current.providerOptions.map(option => option.name)).toEqual(["codex"]);
  });

  it("uses global settings-backed providers after switching scope", () => {
    const { result } = renderAgentCreateDialog();

    act(() => {
      result.current.openDialog();
      result.current.onDraftChange({ ...result.current.draft, scope: "global", provider: "" });
    });

    expect(result.current.providerOptions.map(option => option.name)).toEqual(["claude"]);
    expect(result.current.providerOptions[0]?.display_name).toBe("Claude Code");
  });

  it("submits a workspace create request and navigates to the created agent", async () => {
    const { result } = renderAgentCreateDialog();

    act(() => {
      result.current.openDialog();
      result.current.onDraftChange({
        ...result.current.draft,
        name: "release-captain",
        categoryPath: "Engineering/Release",
        provider: "codex",
        model: "gpt-5.4",
        prompt: "Own release readiness.",
        tools: ["agh__skill_view"],
        toolsets: ["agh__catalog"],
        denyTools: ["agh__task_*"],
        disabledSkills: ["copywriting"],
        permissions: "approve-reads",
      });
    });

    await act(async () => {
      await result.current.onSubmit();
    });

    expect(mockCreateAgent).toHaveBeenCalledWith({
      scope: "workspace",
      workspace: "ws_alpha",
      agent: {
        name: "release-captain",
        provider: "codex",
        prompt: "Own release readiness.",
        model: "gpt-5.4",
        tools: ["agh__skill_view"],
        toolsets: ["agh__catalog"],
        deny_tools: ["agh__task_*"],
        permissions: "approve-reads",
        category_path: ["Engineering", "Release"],
        skills: { disabled: ["copywriting"] },
      },
    });
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/agents/$name",
      params: { name: "release-captain" },
    });
    expect(result.current.open).toBe(false);
  });

  it("keeps the dialog open and reports submit failures", async () => {
    mockCreateAgent.mockRejectedValue(new Error("agent definition already exists"));
    const { result } = renderAgentCreateDialog();

    act(() => {
      result.current.openDialog();
      result.current.onDraftChange({
        ...result.current.draft,
        name: "release-captain",
        provider: "codex",
        prompt: "Own release readiness.",
      });
    });

    await act(async () => {
      await result.current.onSubmit();
    });

    expect(result.current.open).toBe(true);
    expect(result.current.submitError).toBe("agent definition already exists");
    expect(mockToastError).toHaveBeenCalledWith("agent definition already exists");
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("blocks global submit when global providers fail to load", async () => {
    mockSettingsProviders.data = undefined;
    mockSettingsProviders.error = new Error("Unable to load global provider settings.");
    const { result } = renderAgentCreateDialog();

    act(() => {
      result.current.openDialog();
      result.current.onDraftChange({
        ...result.current.draft,
        scope: "global",
        name: "global-reviewer",
        provider: "claude",
        prompt: "Review global work.",
      });
    });

    await act(async () => {
      await result.current.onSubmit();
    });

    expect(mockCreateAgent).not.toHaveBeenCalled();
    expect(result.current.submitError).toBe("Unable to load global provider settings.");
  });
});
