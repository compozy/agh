import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { AgentCreateDialog, type AgentCreateDialogProps } from "../agent-create-dialog";
import {
  createDefaultAgentCreateDraft,
  type AgentCreateDialogDraft,
} from "../../lib/agent-create-draft";

const providers = [
  {
    name: "codex",
    display_name: "Codex",
    harness: "acp",
    runtime_provider: "codex",
  },
  {
    name: "claude",
    display_name: "Claude Code",
    harness: "acp",
    runtime_provider: "claude",
  },
];

function validDraft(overrides: Partial<AgentCreateDialogDraft> = {}): AgentCreateDialogDraft {
  return {
    ...createDefaultAgentCreateDraft(true),
    name: "release-captain",
    provider: "codex",
    prompt: "Own release readiness.",
    ...overrides,
  };
}

function makeProps(overrides: Partial<AgentCreateDialogProps> = {}): AgentCreateDialogProps {
  return {
    open: true,
    onOpenChange: vi.fn(),
    draft: createDefaultAgentCreateDraft(true),
    onDraftChange: vi.fn(),
    onSubmit: vi.fn(),
    providerOptions: providers,
    providersLoading: false,
    providersError: null,
    modelOptions: ["gpt-5.4", "gpt-5.4-mini"],
    modelCatalogLoading: false,
    modelCatalogError: null,
    submitError: null,
    isSubmitting: false,
    hasActiveWorkspace: true,
    workspaceName: "alpha",
    ...overrides,
  };
}

function renderDialog(props: Partial<AgentCreateDialogProps> = {}) {
  return render(
    <UIProvider reducedMotion="always">
      <AgentCreateDialog {...makeProps(props)} />
    </UIProvider>
  );
}

function renderStatefulDialog(props: Partial<AgentCreateDialogProps> = {}) {
  const baseProps = makeProps(props);

  function Harness() {
    const [draft, setDraft] = useState(baseProps.draft);
    return <AgentCreateDialog {...baseProps} draft={draft} onDraftChange={setDraft} />;
  }

  return render(
    <UIProvider reducedMotion="always">
      <Harness />
    </UIProvider>
  );
}

async function reachRuntimeStep() {
  const user = userEvent.setup();
  await user.type(screen.getByTestId("agent-create-name"), "release-captain");
  await user.click(screen.getByTestId("agent-create-next"));
  return user;
}

async function reachAccessStep() {
  const user = await reachRuntimeStep();
  await user.click(screen.getByTestId("agent-create-provider"));
  await user.click(screen.getByTestId("agent-create-provider-item-codex"));
  await user.click(screen.getByTestId("agent-create-next"));
  await user.type(screen.getByTestId("agent-create-prompt"), "Own release readiness.");
  await user.click(screen.getByTestId("agent-create-next"));
  return user;
}

describe("AgentCreateDialog", () => {
  it("Should anchor the dialog to the 880 px modal width token", () => {
    renderDialog();

    const dialog = screen.getByTestId("agent-create-dialog");
    expect(dialog.className).toContain("w-(--width-modal-lg)");
    expect(dialog.className).toContain("sm:max-w-(--width-modal-lg)");
  });

  it("Should block wizard progress until basics are valid", async () => {
    const user = userEvent.setup();
    renderStatefulDialog();

    expect(screen.getByTestId("agent-create-progress")).toHaveTextContent("Step 1 of 4");
    expect(screen.getByTestId("agent-create-next")).toBeDisabled();

    await user.type(screen.getByTestId("agent-create-name"), "../bad");

    expect(screen.getByTestId("agent-create-name-error")).toHaveTextContent(
      "Agent names cannot be . or .. and cannot contain path separators."
    );
    expect(screen.getByTestId("agent-create-next")).toBeDisabled();
  });

  it("Should advance through all wizard steps and submit a valid draft", async () => {
    const onSubmit = vi.fn();
    renderStatefulDialog({ onSubmit });

    const user = await reachAccessStep();

    expect(screen.getByTestId("agent-create-progress")).toHaveTextContent("Step 4 of 4");
    expect(screen.getByTestId("submit-agent-create")).toBeEnabled();

    await user.click(screen.getByTestId("submit-agent-create"));

    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("Should switch scope and surface provider-loading errors inline", async () => {
    const user = userEvent.setup();
    renderStatefulDialog({
      providerOptions: [],
      providersLoading: false,
      providersError: "Unable to load global provider settings.",
    });

    await user.click(screen.getByTestId("agent-create-scope-global"));
    await user.type(screen.getByTestId("agent-create-name"), "global-reviewer");
    await user.click(screen.getByTestId("agent-create-next"));

    expect(screen.getByTestId("agent-create-provider-error")).toHaveTextContent(
      "Unable to load global provider settings."
    );
    expect(screen.getByTestId("agent-create-next")).toBeDisabled();
  });

  it("Should accept comma and newline token input while preserving de-duped order", async () => {
    renderStatefulDialog();

    const user = await reachAccessStep();
    const input = screen.getByTestId("agent-create-tools-input");

    await user.type(input, "agh__skill_view,");
    await user.type(input, "mcp__github__*{enter}");
    await user.type(input, "agh__skill_view{enter}");

    expect(screen.getByTestId("agent-create-tools-tokens")).toHaveTextContent("agh__skill_view");
    expect(screen.getByTestId("agent-create-tools-tokens")).toHaveTextContent("mcp__github__*");
    expect(screen.getAllByLabelText("Remove agh__skill_view")).toHaveLength(1);
  });

  it("Should surface duplicate submit errors", async () => {
    renderStatefulDialog({
      draft: validDraft(),
      submitError: "agent definition already exists",
    });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("agent-create-next"));
    await user.click(screen.getByTestId("agent-create-next"));
    await user.click(screen.getByTestId("agent-create-next"));

    expect(screen.getByTestId("agent-create-submit-error")).toHaveTextContent(
      "agent definition already exists"
    );
    expect(screen.getByTestId("submit-agent-create")).toBeEnabled();
  });

  it("Should disable controls while submitting", () => {
    renderStatefulDialog({
      draft: validDraft(),
      isSubmitting: true,
    });

    expect(screen.getByTestId("agent-create-next")).toBeDisabled();
    expect(screen.getByTestId("agent-create-cancel")).toBeDisabled();
  });
});
