import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "@/systems/agent";
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";

import { SessionCreateDialog, type SessionCreateDialogProps } from "./session-create-dialog";

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
  { name: "claude" },
  { name: "codex" },
  { name: "gemini" },
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
    selectedProvider: "claude",
    providerOptions,
    providersLoading: false,
    providersError: null,
    onAgentChange: vi.fn(),
    onProviderChange: vi.fn(),
    onSubmit: vi.fn(),
    isSubmitting: false,
    submitError: null,
    ...overrides,
  };
}

describe("SessionCreateDialog", () => {
  it("renders the provider picker with every workspace provider option", () => {
    render(<SessionCreateDialog {...makeProps()} />);

    expect(screen.getByTestId("session-create-dialog").className).toContain("sm:max-w-lg");
    expect(screen.getByTestId("session-create-dialog").className).not.toContain("sm:max-w-[30rem]");

    const picker = screen.getByTestId("session-create-provider-select") as HTMLSelectElement;
    expect(picker).toBeEnabled();
    expect(picker.value).toBe("claude");
    const values = Array.from(picker.options).map(option => option.value);
    expect(values).toEqual(["claude", "codex", "gemini"]);
  });

  it("preselects the incoming agent name in the agent picker", () => {
    render(<SessionCreateDialog {...makeProps({ selectedAgentName: "codex-agent" })} />);

    const agentSelect = screen.getByTestId("session-create-agent-select") as HTMLSelectElement;
    expect(agentSelect.value).toBe("codex-agent");
  });

  it("calls onAgentChange when the operator picks a different agent", () => {
    const onAgentChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onAgentChange })} />);

    fireEvent.change(screen.getByTestId("session-create-agent-select"), {
      target: { value: "codex-agent" },
    });
    expect(onAgentChange).toHaveBeenCalledWith("codex-agent");
  });

  it("calls onProviderChange when the operator picks a different provider", () => {
    const onProviderChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onProviderChange })} />);

    fireEvent.change(screen.getByTestId("session-create-provider-select"), {
      target: { value: "codex" },
    });
    expect(onProviderChange).toHaveBeenCalledWith("codex");
  });

  it("calls onSubmit only once when the form is submitted with a valid draft", () => {
    const onSubmit = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onSubmit })} />);

    fireEvent.click(screen.getByTestId("session-create-dialog-submit"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("disables submit when no providers are available and surfaces an empty-state note", () => {
    render(<SessionCreateDialog {...makeProps({ providerOptions: [], selectedProvider: "" })} />);

    expect(screen.getByTestId("session-create-dialog-submit")).toBeDisabled();
    expect(screen.getByTestId("session-create-providers-empty")).toBeInTheDocument();
    expect(screen.getByTestId("session-create-providers-empty").className).toContain("text-xs");
  });

  it("disables submit and surfaces submitError when creation fails", () => {
    render(
      <SessionCreateDialog
        {...makeProps({ submitError: "Server rejected the session", isSubmitting: false })}
      />
    );

    expect(screen.getByTestId("session-create-submit-error")).toHaveTextContent(
      "Server rejected the session"
    );
  });

  it("shows provider-loading state and disables the picker while loading", () => {
    render(
      <SessionCreateDialog
        {...makeProps({ providerOptions: [], providersLoading: true, selectedProvider: "" })}
      />
    );

    const picker = screen.getByTestId("session-create-provider-select") as HTMLSelectElement;
    expect(picker).toBeDisabled();
    expect(picker).toHaveTextContent("Loading providers…");
  });

  it("blocks backdrop dismissal while submit is in flight", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ isSubmitting: true, onOpenChange })} />);

    fireEvent.click(getDialogBackdrop());
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("allows backdrop dismissal when submit is idle", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onOpenChange })} />);

    fireEvent.click(getDialogBackdrop());
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("closes via cancel button", () => {
    const onOpenChange = vi.fn();
    render(<SessionCreateDialog {...makeProps({ onOpenChange })} />);

    fireEvent.click(screen.getByTestId("session-create-dialog-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
