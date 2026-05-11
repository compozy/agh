import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("motion/react", async () => {
  const actual = await vi.importActual<typeof import("motion/react")>("motion/react");
  return {
    ...actual,
    AnimatePresence: ({ children }: { children: ReactNode }) => <>{children}</>,
  };
});

import { TaskEditorModal } from "../task-editor-modal";
import {
  EMPTY_TASK_EDITOR_DRAFT,
  createTaskEditorDraft,
  type TaskEditorDraft,
} from "../../lib/task-editor";
import { getTaskTemplate } from "../../lib/task-templates";
import type { TaskRecord } from "../../types";

function renderNewModal(overrides: Partial<React.ComponentProps<typeof TaskEditorModal>> = {}) {
  const onOpenChange = vi.fn();
  const onTemplateChange = vi.fn();
  const onSubmit = vi.fn().mockResolvedValue(undefined);
  const onDraftChange = vi.fn();
  const draft: TaskEditorDraft = createTaskEditorDraft("one_shot", "ws_alpha");
  const template = getTaskTemplate("one_shot");
  const result = render(
    <TaskEditorModal
      canSubmit
      draft={draft}
      mode="new"
      onDraftChange={onDraftChange}
      onOpenChange={onOpenChange}
      onSubmit={onSubmit}
      onTemplateChange={onTemplateChange}
      open
      task={null}
      template={template}
      templateId="one_shot"
      workspaceName="Alpha"
      {...overrides}
    />
  );
  return { ...result, onOpenChange, onTemplateChange, onSubmit, onDraftChange, draft, template };
}

const editTask = {
  id: "task_42",
  identifier: "TASK-42",
  title: "Summarize review feedback",
  status: "in_progress",
  scope: "workspace",
  origin: { kind: "cli", ref: "op" },
  workspace_id: "ws_alpha",
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:30:00Z",
  created_by: { kind: "human", ref: "pedro@" },
} as unknown as TaskRecord;

describe("TaskEditorModal", () => {
  it("Should render the template picker first in new mode", () => {
    renderNewModal();
    expect(screen.getByTestId("task-editor-modal")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-modal-template-picker")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-template-one_shot")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-template-recurring")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-modal-title")).toHaveTextContent("New task");
  });

  it("Should hide the template picker in edit mode and open directly on the form", () => {
    const draft = {
      ...EMPTY_TASK_EDITOR_DRAFT,
      title: "Summarize review feedback",
      maxAttempts: 1,
    };
    render(
      <TaskEditorModal
        canSubmit
        draft={draft}
        mode="edit"
        onDraftChange={vi.fn()}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        open
        task={editTask}
        workspaceName="Alpha"
      />
    );
    expect(screen.queryByTestId("task-editor-modal-template-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("task-editor-modal-title")).toHaveTextContent("Edit task");
    expect(screen.getByTestId("task-editor-title-input")).toHaveValue("Summarize review feedback");
    expect(screen.getByTestId("task-editor-modal-submit")).toHaveTextContent("Save changes");
  });

  it("Should render the modal at the 720 px width token", () => {
    renderNewModal();
    const modal = screen.getByTestId("task-editor-modal");
    expect(modal.className).toContain("w-[var(--width-modal-md)]");
  });

  it("Should apply the overlay scrim token with backdrop-filter blur via inline style", () => {
    renderNewModal();
    const overlay = document.body.querySelector(
      "[data-slot='dialog-overlay']"
    ) as HTMLElement | null;
    expect(overlay).not.toBeNull();
    expect(overlay?.className).toContain("bg-(--overlay-scrim)");
    expect(overlay?.style.backdropFilter).toBe("blur(var(--overlay-blur))");
  });

  it("Should render Enqueue task when enqueueOnSubmit is true", () => {
    renderNewModal();
    expect(screen.getByTestId("task-editor-modal-submit")).toHaveTextContent("Enqueue task");
  });

  it("Should render Save draft when enqueueOnSubmit is false (recurring template)", () => {
    const template = getTaskTemplate("recurring");
    const draft = createTaskEditorDraft("recurring", "ws_alpha");
    renderNewModal({ template, templateId: "recurring", draft });
    expect(screen.getByTestId("task-editor-modal-submit")).toHaveTextContent("Save draft");
  });

  it("Should render the canonical footer hint adopted from the proposal", () => {
    renderNewModal();
    expect(screen.getByTestId("task-editor-modal-hint")).toHaveTextContent(
      "The contract is durable — runs descend from this task and respect dependencies."
    );
  });

  it("Should keep Cancel on the left of the footer actions", () => {
    renderNewModal();
    const footer = screen.getByTestId("task-editor-modal-footer");
    const buttons = footer.querySelectorAll("button");
    expect(buttons.length).toBeGreaterThanOrEqual(2);
    expect(buttons[0]).toHaveAttribute("data-testid", "task-editor-modal-cancel");
    expect(buttons[buttons.length - 1]).toHaveAttribute("data-testid", "task-editor-modal-submit");
  });

  it("Should expose only 1 / 2 / 3 / 5 in the max attempts options", () => {
    renderNewModal();
    expect(screen.getByTestId("task-editor-attempts-1")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-attempts-2")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-attempts-3")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-attempts-5")).toBeInTheDocument();
    expect(screen.queryByTestId("task-editor-attempts-default")).not.toBeInTheDocument();
    const attemptsGroup = screen.getByTestId("task-editor-attempts-options");
    expect(attemptsGroup.textContent ?? "").not.toMatch(/Default/);
  });

  it("Should default maxAttempts to 1", () => {
    const draft = createTaskEditorDraft("blank", "ws_alpha");
    expect(draft.maxAttempts).toBe(1);
  });

  it("Should render the 6-kind owner enum matching the backend OwnerKind", () => {
    renderNewModal();
    const select = screen.getByTestId("task-editor-owner-kind");
    const options = Array.from(select.querySelectorAll("option")).map(option => option.value);
    expect(options).toEqual([
      "",
      "agent_session",
      "human",
      "automation",
      "extension",
      "network_peer",
      "pool",
    ]);
  });

  it("Should invoke onOpenChange when Cancel is clicked", () => {
    const { onOpenChange } = renderNewModal();
    fireEvent.click(screen.getByTestId("task-editor-modal-cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should invoke onTemplateChange when a template card is selected", () => {
    const { onTemplateChange } = renderNewModal();
    fireEvent.click(screen.getByTestId("task-editor-template-recurring"));
    expect(onTemplateChange).toHaveBeenCalledWith("recurring");
  });
});
