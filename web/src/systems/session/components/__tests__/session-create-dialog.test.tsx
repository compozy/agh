import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";
import type { AgentPayload } from "@/systems/agent";
import type { ModelOption } from "@/systems/model-catalog";
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";

import { SessionCreateDialog, type SessionCreateDialogProps } from "../session-create-dialog";

const agents: AgentPayload[] = [
  { name: "claude-agent", provider: "claude", prompt: "help" },
  { name: "codex-agent", provider: "codex", prompt: "code" },
];

const workspace: WorkspacePayload = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: [],
  name: "alpha",
  created_at: "2026-04-20T10:00:00Z",
  updated_at: "2026-04-20T10:00:00Z",
};

const providerOptions: SessionProviderOption[] = [
  {
    name: "claude",
    display_name: "Claude Code",
    harness: "acp",
    runtime_provider: "claude",
  },
  {
    name: "codex",
    display_name: "Codex",
  },
  {
    name: "openrouter",
    display_name: "OpenRouter",
    harness: "pi_acp",
    runtime_provider: "openrouter",
  },
];

const codexModelOptions: ModelOption[] = [
  {
    id: "gpt-5.4",
    displayName: "GPT-5.4",
    availabilityState: "available_live",
    available: true,
    stale: false,
    refreshedAt: "2026-05-07T10:00:00Z",
    source: "catalog",
  },
  {
    id: "gpt-5.4-mini",
    displayName: "GPT-5.4 Mini",
    availabilityState: "available_stale",
    available: true,
    stale: true,
    refreshedAt: "2026-05-06T10:00:00Z",
    source: "catalog",
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
  const selectedProvider = overrides.selectedProvider ?? "claude";
  const fallbackProviderOption =
    overrides.providerOptions?.find(option => option.name === selectedProvider) ??
    providerOptions.find(option => option.name === selectedProvider);
  return {
    open: true,
    onOpenChange: vi.fn(),
    agents,
    workspace,
    selectedAgentName: "claude-agent",
    selectedProvider,
    selectedProviderOption: fallbackProviderOption,
    selectedModel: "",
    selectedReasoning: "",
    modelOptions: [],
    reasoningOptions: [],
    reasoningSupported: false,
    catalogStale: false,
    catalogLoading: false,
    catalogError: null,
    catalogRefreshing: false,
    catalogRefreshError: null,
    defaultReasoning: null,
    providerOptions,
    providersLoading: false,
    providersError: null,
    onAgentChange: vi.fn(),
    onProviderChange: vi.fn(),
    onModelChange: vi.fn(),
    onReasoningChange: vi.fn(),
    onCatalogRefresh: vi.fn(),
    onSubmit: vi.fn(),
    isSubmitting: false,
    submitError: null,
    ...overrides,
  };
}

describe("SessionCreateDialog", () => {
  it("Should render the provider picker with the selected provider in the trigger", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog {...makeProps()} />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-dialog").className).toContain("sm:max-w-xl");
    expect(screen.getByTestId("session-create-dialog").className).not.toContain("sm:max-w-120");

    const trigger = screen.getByTestId("session-create-provider-select");
    expect(trigger).toBeEnabled();
    expect(trigger).toHaveTextContent("Claude Code");
    expect(screen.getByTestId("session-create-provider-runtime")).toHaveTextContent("acp");
  });

  it("Should preselect the incoming agent name in the agent picker trigger", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog {...makeProps({ selectedAgentName: "codex-agent" })} />
      </UIProvider>
    );

    const trigger = screen.getByTestId("session-create-agent-select");
    expect(trigger).toHaveTextContent("codex-agent");
  });

  it("Should call onAgentChange when the operator picks a different agent", async () => {
    const user = userEvent.setup();
    const onAgentChange = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog {...makeProps({ onAgentChange })} />
      </UIProvider>
    );

    await user.click(screen.getByTestId("session-create-agent-select"));
    await user.click(screen.getByTestId("agent-command-item-codex-agent"));
    expect(onAgentChange).toHaveBeenCalledWith("codex-agent");
  });

  it("Should call onProviderChange when the operator picks a different provider", async () => {
    const user = userEvent.setup();
    const onProviderChange = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog {...makeProps({ onProviderChange })} />
      </UIProvider>
    );

    await user.click(screen.getByTestId("session-create-provider-select"));
    await user.click(screen.getByTestId("provider-command-item-codex"));
    expect(onProviderChange).toHaveBeenCalledWith("codex");
  });

  it("Should let the operator select a catalog model and reasoning effort", async () => {
    const user = userEvent.setup();
    const onModelChange = vi.fn();
    const onReasoningChange = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "codex-agent",
            selectedProvider: "codex",
            modelOptions: codexModelOptions,
            onModelChange,
            onReasoningChange,
            reasoningSupported: true,
            reasoningOptions: [
              { value: "low", label: "Low", source: "catalog" },
              { value: "medium", label: "Medium", source: "catalog" },
              { value: "high", label: "High", source: "catalog" },
            ],
          })}
        />
      </UIProvider>
    );

    await user.click(screen.getByTestId("session-create-model-select"));
    await user.click(screen.getByTestId("model-command-item-gpt-5.4-mini"));
    expect(onModelChange).toHaveBeenCalledWith("gpt-5.4-mini");

    await user.click(screen.getByTestId("session-create-reasoning-select"));
    await user.click(screen.getByTestId("reasoning-command-item-high"));
    expect(onReasoningChange).toHaveBeenCalledWith("high");
  });

  it("Should render distinct availability badges for each catalog state", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "codex-agent",
            selectedProvider: "codex",
            modelOptions: [
              {
                id: "gpt-live",
                displayName: "GPT live",
                availabilityState: "available_live",
                available: true,
                stale: false,
                source: "catalog",
              },
              {
                id: "gpt-stale",
                displayName: "GPT stale",
                availabilityState: "available_stale",
                available: true,
                stale: true,
                source: "catalog",
              },
              {
                id: "gpt-down",
                displayName: "GPT down",
                availabilityState: "unavailable_live",
                available: false,
                stale: false,
                source: "catalog",
              },
              {
                id: "gpt-down-stale",
                displayName: "GPT down stale",
                availabilityState: "unavailable_stale",
                available: false,
                stale: true,
                source: "catalog",
              },
              {
                id: "gpt-unknown",
                displayName: "GPT unknown",
                availabilityState: "unknown",
                available: null,
                stale: false,
                source: "catalog",
              },
            ],
          })}
        />
      </UIProvider>
    );

    fireEvent.click(screen.getByTestId("session-create-model-select"));
    expect(screen.getByTestId("model-command-item-gpt-live-availability")).toHaveTextContent(
      "live"
    );
    expect(screen.getByTestId("model-command-item-gpt-stale-availability")).toHaveTextContent(
      "stale"
    );
    expect(screen.getByTestId("model-command-item-gpt-down-availability")).toHaveTextContent(
      "unavailable"
    );
    expect(screen.getByTestId("model-command-item-gpt-down-stale-availability")).toHaveTextContent(
      "stale · unavailable"
    );
    expect(screen.getByTestId("model-command-item-gpt-unknown-availability")).toHaveTextContent(
      "unknown"
    );
  });

  it("Should surface stale catalog state without blocking submit", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "codex-agent",
            selectedProvider: "codex",
            modelOptions: codexModelOptions,
            catalogStale: true,
          })}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-catalog-stale")).toHaveTextContent(
      "Some models are stale"
    );
    expect(screen.getByTestId("session-create-dialog-submit")).toBeEnabled();
  });

  it("Should surface catalog source errors without hiding manual entry", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "codex-agent",
            selectedProvider: "codex",
            modelOptions: [],
            catalogError: "catalog upstream failed",
          })}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-catalog-error")).toHaveTextContent(
      "catalog upstream failed"
    );
    expect(screen.getByTestId("session-create-model-select")).toBeEnabled();
  });

  it("Should expose a refresh control that triggers onCatalogRefresh", async () => {
    const onCatalogRefresh = vi.fn();
    const user = userEvent.setup();
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "codex-agent",
            selectedProvider: "codex",
            modelOptions: codexModelOptions,
            onCatalogRefresh,
          })}
        />
      </UIProvider>
    );

    await user.click(screen.getByTestId("session-create-catalog-refresh"));
    expect(onCatalogRefresh).toHaveBeenCalledTimes(1);
  });

  it("Should call onSubmit only once when the form is submitted with a valid draft", () => {
    const onSubmit = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onSubmit })} />);

    fireEvent.click(screen.getByTestId("session-create-dialog-submit"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("Should disable submit when no providers are available and surface an empty-state note", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            providerOptions: [],
            selectedProvider: "",
            selectedProviderOption: undefined,
          })}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
    expect(screen.getByTestId("session-create-providers-empty")).toBeInTheDocument();
    expect(screen.getByTestId("session-create-providers-empty").className).toContain("text-xs");
  });

  it("Should surface submitError when creation fails", () => {
    render(
      <SessionCreateDialog
        {...makeProps({ submitError: "Server rejected the session", isSubmitting: false })}
      />
    );

    expect(screen.getByTestId("session-create-submit-error")).toHaveTextContent(
      "Server rejected the session"
    );
  });

  it("Should disable submit when the current selections are no longer available", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            selectedAgentName: "missing-agent",
            selectedProvider: "missing-provider",
            selectedProviderOption: undefined,
          })}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
  });

  it("Should show provider-loading state and disable the picker while loading", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            providerOptions: [],
            providersLoading: true,
            selectedProvider: "",
            selectedProviderOption: undefined,
          })}
        />
      </UIProvider>
    );

    const picker = screen.getByTestId("session-create-provider-select");
    expect(picker).toBeDisabled();
    expect(picker).toHaveTextContent("Loading providers…");
  });

  it("Should disable both pickers until a workspace is selected", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            workspace: undefined,
            selectedAgentName: "claude-agent",
            selectedProvider: "claude",
            selectedProviderOption: undefined,
          })}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("session-create-agent-select")).toBeDisabled();
    expect(screen.getByTestId("session-create-agent-select")).toHaveTextContent(
      "Select a workspace first"
    );
    expect(screen.queryByTestId("session-create-agent-default")).not.toBeInTheDocument();

    const providerPicker = screen.getByTestId("session-create-provider-select");
    expect(providerPicker).toBeDisabled();
    expect(providerPicker).toHaveTextContent("Select a workspace first");
    expect(screen.queryByTestId("session-create-provider-runtime")).not.toBeInTheDocument();
    expect(screen.queryByTestId("session-create-providers-empty")).not.toBeInTheDocument();
  });

  it("Should not render blank agent provider metadata for inherited providers", () => {
    render(
      <UIProvider reducedMotion="always">
        <SessionCreateDialog
          {...makeProps({
            agents: [{ name: "general", provider: "", prompt: "help" }],
            selectedAgentName: "general",
            selectedProvider: "codex",
          })}
        />
      </UIProvider>
    );

    expect(screen.queryByTestId("session-create-agent-default")).not.toBeInTheDocument();
    expect(screen.getByTestId("session-create-provider-select")).toHaveTextContent("Codex");
  });

  it("Should block backdrop dismissal while submit is in flight", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ isSubmitting: true, onOpenChange })} />);

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
