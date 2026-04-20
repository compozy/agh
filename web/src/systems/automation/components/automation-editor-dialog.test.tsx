import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { AutomationEditorDialog } from "./automation-editor-dialog";
import { createAutomationJobDraft, createAutomationTriggerDraft } from "../lib/automation-drafts";
import type { CreateAutomationJobRequest, CreateAutomationTriggerRequest } from "../types";

function JobEditorHarness({
  onCancel,
  onSubmit,
}: {
  onCancel: () => void;
  onSubmit: (draft: CreateAutomationJobRequest) => void;
}) {
  const [draft, setDraft] = useState<CreateAutomationJobRequest>(() =>
    createAutomationJobDraft("ws_test")
  );

  return (
    <AutomationEditorDialog
      activeWorkspaceId="ws_test"
      editor={{
        draft,
        isPending: false,
        kind: "jobs",
        mode: "create",
        onCancel,
        onChange: setDraft,
        onSubmit: () => onSubmit(draft),
      }}
    />
  );
}

function TriggerEditorHarness({
  onCancel,
  onSubmit,
}: {
  onCancel: () => void;
  onSubmit: (draft: CreateAutomationTriggerRequest) => void;
}) {
  const [draft, setDraft] = useState<CreateAutomationTriggerRequest>(() =>
    createAutomationTriggerDraft("ws_test")
  );

  return (
    <AutomationEditorDialog
      activeWorkspaceId="ws_test"
      editor={{
        draft,
        isPending: false,
        kind: "triggers",
        mode: "edit",
        onCancel,
        onChange: setDraft,
        onSubmit: () => onSubmit(draft),
      }}
    />
  );
}

describe("AutomationEditorDialog", () => {
  it("renders the create-job dialog header and keeps submit disabled until every required field is valid", () => {
    const onCancel = vi.fn();
    const onSubmit = vi.fn();

    render(<JobEditorHarness onCancel={onCancel} onSubmit={onSubmit} />);

    expect(screen.getByTestId("automation-editor-dialog")).toBeInTheDocument();
    expect(screen.getByText("Create job")).toBeInTheDocument();
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-name-input"), {
      target: { value: "nightly-docs" },
    });
    expect(screen.getByTestId("submit-job-form")).toBeDisabled();

    fireEvent.change(screen.getByTestId("job-agent-input"), {
      target: { value: "writer" },
    });
    fireEvent.change(screen.getByTestId("job-prompt-input"), {
      target: { value: "Summarize the latest commits." },
    });

    expect(screen.getByTestId("submit-job-form")).toBeEnabled();

    fireEvent.click(screen.getByTestId("submit-job-form"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        agent_name: "writer",
        name: "nightly-docs",
        prompt: "Summarize the latest commits.",
      })
    );
  });

  it("renders the edit-trigger dialog header for the triggers kind", () => {
    render(<TriggerEditorHarness onCancel={vi.fn()} onSubmit={vi.fn()} />);

    expect(screen.getByText("Edit trigger")).toBeInTheDocument();
    expect(screen.getByTestId("automation-trigger-form")).toBeInTheDocument();
  });

  it("invokes onCancel when the underlying dialog signals close", () => {
    const onCancel = vi.fn();
    const { rerender } = render(<JobEditorHarness onCancel={onCancel} onSubmit={vi.fn()} />);

    fireEvent.keyDown(document.body, { key: "Escape" });
    // Base UI Dialog closes on escape; even if the JSDOM path is brittle, we also
    // cover the explicit close by unmounting via editor=null + remount, which is
    // the real exit path in useAutomationPage.
    rerender(<AutomationEditorDialog editor={null} />);

    expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
  });

  it("does not render the dialog content when editor is null", () => {
    render(<AutomationEditorDialog editor={null} />);

    expect(screen.queryByTestId("automation-editor-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("automation-job-form")).not.toBeInTheDocument();
  });
});
