import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";
import type { AgentPayload } from "@/systems/agent";
import { FIXTURE_AGENT_DEFINITION_DIGEST } from "@/systems/agent/mocks";
import type { RuntimeModelOption, RuntimeProviderOption } from "@/systems/runtime";
import type { WorkspacePayload } from "@/systems/workspace";

import { SessionCreateDialog, type SessionCreateDialogProps } from "../session-create-dialog";

const agents: AgentPayload[] = [
  {
    name: "claude-agent",
    provider: "claude",
    prompt: "help",
    origin: "workspace",
    workspace_id: "ws_alpha",
    definition_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
  },
  {
    name: "codex-agent",
    provider: "codex",
    prompt: "code",
    origin: "workspace",
    workspace_id: "ws_alpha",
    definition_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
  },
];

const workspace: WorkspacePayload = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: [],
  name: "alpha",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:00Z",
};

const runtimeProviders: RuntimeProviderOption[] = [
  { id: "claude", name: "Claude Code", harness: "acp", runtime_provider: "claude" },
  { id: "codex", name: "Codex", runtime_provider: "codex" },
  { id: "cursor", name: "Cursor", harness: "acp", runtime_provider: "cursor" },
  { id: "openrouter", name: "OpenRouter", harness: "pi_acp", runtime_provider: "openrouter" },
];

const runtimeModels: RuntimeModelOption[] = [
  {
    id: "gpt-5.4",
    provider: "codex",
    name: "GPT-5.4",
    efforts: ["low", "medium", "high"],
    availability: "live",
    curated: true,
  },
  {
    id: "gpt-5.4-mini",
    provider: "codex",
    name: "GPT-5.4 Mini",
    efforts: [],
    availability: "stale",
    curated: true,
  },
];

function getDialogBackdrop(): HTMLElement {
  const backdrop = document.querySelector('[data-slot="dialog-overlay"]');
  if (!(backdrop instanceof HTMLElement)) {
    throw new Error("Expected dialog backdrop to be rendered.");
  }
  return backdrop;
}

function makeProps(overrides: Partial<SessionCreateDialogProps> = {}): SessionCreateDialogProps {
  return {
    open: true,
    onOpenChange: vi.fn(),
    agents,
    workspace,
    selectedAgentName: "claude-agent",
    runtimeValue: { provider: "claude", model: "", reasoning_effort: "" },
    runtimeProviders,
    runtimeModels: [],
    catalogStale: false,
    catalogLoading: false,
    catalogLoaded: true,
    catalogError: null,
    catalogRefreshing: false,
    catalogRefreshError: null,
    providersLoading: false,
    providersError: null,
    hasProviderOptions: true,
    networkParticipation: {
      mode: "local",
      channelId: "",
      channelStrategy: "",
    },
    onAgentChange: vi.fn(),
    onRuntimeChange: vi.fn(),
    onNetworkParticipationChange: vi.fn(),
    onCatalogRefresh: vi.fn(),
    onOpenProviderSettings: vi.fn(),
    onSubmit: vi.fn(),
    isSubmitting: false,
    submitError: null,
    ...overrides,
  };
}

function renderDialog(overrides: Partial<SessionCreateDialogProps> = {}) {
  return render(
    <UIProvider reducedMotion="always">
      <SessionCreateDialog {...makeProps(overrides)} />
    </UIProvider>
  );
}

async function openRuntimePopup(user: ReturnType<typeof userEvent.setup>) {
  const segment = document.querySelector<HTMLElement>(
    '[data-testid="session-create-runtime-select"] button[data-focus="model"]'
  );
  if (!segment) throw new Error("Runtime selector model segment not found");
  await user.click(segment);
  return screen.findByTestId("runtime-selector-popup");
}

describe("SessionCreateDialog", () => {
  it("Should render the runtime selector wired to the selected provider", () => {
    renderDialog();

    expect(screen.getByTestId("session-create-dialog").className).toContain(
      "sm:max-w-(--width-modal-sm)"
    );
    expect(screen.getByTestId("session-create-dialog").className).not.toContain("sm:max-w-120");

    const trigger = screen.getByTestId("session-create-runtime-select");
    expect(trigger).toHaveTextContent("Claude Code");
    expect(trigger).not.toHaveAttribute("aria-disabled", "true");
  });

  it("Should preselect the incoming agent name in the agent picker trigger", () => {
    renderDialog({ selectedAgentName: "codex-agent" });

    expect(screen.getByTestId("session-create-agent-select")).toHaveTextContent("codex-agent");
  });

  it("Should call onAgentChange when the operator picks a different agent", async () => {
    const user = userEvent.setup();
    const onAgentChange = vi.fn();
    renderDialog({ onAgentChange });

    await user.click(screen.getByTestId("session-create-agent-select"));
    await user.click(screen.getByTestId("agent-command-item-codex-agent"));
    expect(onAgentChange).toHaveBeenCalledWith("codex-agent");
  });

  it("Should surface a stale catalog notice inside the selector without blocking submit", async () => {
    const user = userEvent.setup();
    renderDialog({ runtimeModels, catalogStale: true });

    await openRuntimePopup(user);
    expect(screen.getByTestId("session-create-catalog-stale")).toHaveTextContent(
      "Some models are stale"
    );
    expect(screen.getByTestId("session-create-dialog-submit")).toBeEnabled();
  });

  it("Should surface catalog source errors inside the selector while keeping it usable", async () => {
    const user = userEvent.setup();
    renderDialog({ runtimeModels: [], catalogError: "catalog upstream failed" });

    await openRuntimePopup(user);
    expect(screen.getByTestId("session-create-catalog-error")).toHaveTextContent(
      "catalog upstream failed"
    );
    expect(screen.getByTestId("session-create-runtime-select")).not.toHaveAttribute(
      "aria-disabled",
      "true"
    );
  });

  it("Should expose a refresh control that triggers onCatalogRefresh", async () => {
    const onCatalogRefresh = vi.fn();
    const user = userEvent.setup();
    renderDialog({ runtimeModels, onCatalogRefresh });

    await openRuntimePopup(user);
    await user.click(screen.getByTestId("runtime-selector-refresh"));
    expect(onCatalogRefresh).toHaveBeenCalledTimes(1);
  });

  it("Should call onSubmit only once when the form is submitted with a valid draft", () => {
    const onSubmit = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onSubmit })} />);

    fireEvent.click(screen.getByTestId("session-create-dialog-submit"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("Should disable submit when no providers are available and surface an empty-state note", () => {
    renderDialog({
      runtimeProviders: [],
      hasProviderOptions: false,
      runtimeValue: { provider: "", model: "", reasoning_effort: "" },
    });

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
    expect(screen.getByTestId("session-create-providers-empty")).toBeInTheDocument();
    expect(screen.getByTestId("session-create-providers-empty").className).toContain(
      "text-form-hint"
    );
  });

  it("Should surface providersError when the workspace provider list fails to load", () => {
    renderDialog({
      providersError: "Unable to load provider options for this workspace.",
    });

    expect(screen.getByTestId("session-create-providers-error")).toHaveTextContent(
      "Unable to load provider options for this workspace."
    );
  });

  it("Should surface submitError when creation fails", () => {
    render(<SessionCreateDialog {...makeProps({ submitError: "Server rejected the session" })} />);

    expect(screen.getByTestId("session-create-submit-error")).toHaveTextContent(
      "Server rejected the session"
    );
  });

  it("Should disable submit when the current selections are no longer available", () => {
    renderDialog({
      selectedAgentName: "missing-agent",
      runtimeValue: { provider: "missing-provider", model: "", reasoning_effort: "" },
    });

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
  });

  it("Should block a non-catalog model and explain how to recover inline", () => {
    renderDialog({
      runtimeModels,
      runtimeValue: {
        provider: "cursor",
        model: "cursor-grok-4.5-high",
        reasoning_effort: "",
      },
    });

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
    expect(screen.getByTestId("session-create-model-error")).toHaveTextContent(
      "not in the selected provider catalog"
    );
    expect(screen.getByTestId("session-create-model-error")).toHaveAttribute("role", "alert");
  });

  it("Should disable the runtime selector while providers are loading", () => {
    renderDialog({
      runtimeProviders: [],
      hasProviderOptions: false,
      providersLoading: true,
      runtimeValue: { provider: "", model: "", reasoning_effort: "" },
    });

    expect(screen.getByTestId("session-create-runtime-select")).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
  });

  it("Should disable both pickers until a workspace is selected", () => {
    renderDialog({
      workspace: undefined,
      selectedAgentName: "claude-agent",
      runtimeValue: { provider: "claude", model: "", reasoning_effort: "" },
    });

    expect(screen.getByTestId("session-create-agent-select")).toBeDisabled();
    expect(screen.getByTestId("session-create-agent-select")).toHaveTextContent(
      "Select a workspace first"
    );
    expect(screen.queryByTestId("session-create-agent-default")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-create-runtime-select")).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.queryByTestId("session-create-providers-empty")).not.toBeInTheDocument();
  });

  it("Should not render blank agent provider metadata for inherited providers", () => {
    renderDialog({
      agents: [
        {
          name: "general",
          provider: "",
          prompt: "help",
          origin: "workspace",
          workspace_id: "ws_alpha",
          definition_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
        },
      ],
      selectedAgentName: "general",
      runtimeValue: { provider: "codex", model: "", reasoning_effort: "" },
    });

    expect(screen.queryByTestId("session-create-agent-default")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-create-runtime-select")).toHaveTextContent("Codex");
  });

  it("Should announce truthful pending feedback and block dismissal while submit is in flight", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ isSubmitting: true, onOpenChange })} />);

    expect(screen.getByRole("status")).toHaveTextContent("Waiting for provider startup");
    expect(screen.getByTestId("session-create-dialog-submit")).toHaveTextContent(
      "Starting session"
    );
    fireEvent.click(getDialogBackdrop());
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("Should allow backdrop dismissal when submit is idle", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onOpenChange })} />);

    fireEvent.click(getDialogBackdrop());
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should close via cancel button", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onOpenChange })} />);

    fireEvent.click(screen.getByTestId("session-create-dialog-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
