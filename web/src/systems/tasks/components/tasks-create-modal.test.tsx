import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";

import { TasksCreateModal } from "./tasks-create-modal";
import { getTaskTemplate, type TaskTemplateId } from "../lib/task-templates";

const INITIAL_DRAFT: CreateTaskDraftInput = {
  title: "",
  description: "",
  scope: "workspace",
  priority: "medium",
  ownerKind: "",
  ownerRef: "",
  parentTaskId: "",
  maxAttempts: 1,
  approvalPolicy: "none",
  networkChannel: "",
  identifier: "",
};

function Harness({
  onSubmit,
  initialDraft = INITIAL_DRAFT,
  onTemplateChange,
  templateId = "one_shot",
}: {
  onSubmit?: (draft: CreateTaskDraftInput, asDraft: boolean) => void;
  initialDraft?: CreateTaskDraftInput;
  onTemplateChange?: (id: TaskTemplateId) => void;
  templateId?: TaskTemplateId;
}) {
  const [draft, setDraft] = useState<CreateTaskDraftInput>(initialDraft);
  const [activeTemplate, setActiveTemplate] = useState<TaskTemplateId>(templateId);
  const template = getTaskTemplate(activeTemplate);

  return (
    <TasksCreateModal
      canSubmit={draft.title.trim().length > 0}
      draft={draft}
      onDraftChange={setDraft}
      onOpenChange={() => {}}
      onSubmit={(next, asDraft) => onSubmit?.(next, asDraft)}
      onTemplateChange={id => {
        setActiveTemplate(id);
        onTemplateChange?.(id);
      }}
      open
      template={template}
      templateId={activeTemplate}
      workspaceName="Polybot"
    />
  );
}

describe("TasksCreateModal", () => {
  it("disables submit until a title is entered, then forwards the create payload", () => {
    const onSubmit = vi.fn();
    render(<Harness onSubmit={onSubmit} />);

    expect(screen.getByTestId("tasks-create-modal-template-label")).toHaveTextContent(
      "Starting from One-shot template"
    );

    const submit = screen.getByTestId("tasks-create-modal-submit");
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByTestId("tasks-create-modal-title"), {
      target: { value: "Generate API client" },
    });
    expect(submit).not.toBeDisabled();

    fireEvent.click(submit);
    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit.mock.calls[0]?.[0]?.title).toBe("Generate API client");
    expect(onSubmit.mock.calls[0]?.[1]).toBe(false);
  });

  it("supports save-draft submissions with asDraft=true", () => {
    const onSubmit = vi.fn();
    render(<Harness onSubmit={onSubmit} />);
    fireEvent.change(screen.getByTestId("tasks-create-modal-title"), {
      target: { value: "Draft me" },
    });
    fireEvent.click(screen.getByTestId("tasks-create-modal-save-draft"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit.mock.calls[0]?.[1]).toBe(true);
  });

  it("lets the user pick priority, attempts, and approval policy", () => {
    render(<Harness />);

    fireEvent.click(screen.getByTestId("tasks-create-modal-priority-urgent"));
    expect(screen.getByTestId("tasks-create-modal-priority-urgent")).toHaveAttribute(
      "aria-pressed",
      "true"
    );

    fireEvent.click(screen.getByTestId("tasks-create-modal-attempts-3"));
    expect(screen.getByTestId("tasks-create-modal-attempts-3")).toHaveAttribute(
      "aria-pressed",
      "true"
    );

    fireEvent.click(screen.getByTestId("tasks-create-modal-approval-manual"));
    expect(screen.getByTestId("tasks-create-modal-approval-manual")).toHaveAttribute(
      "aria-pressed",
      "true"
    );
  });

  it("switches templates and updates the descriptive notice for recurring drafts", () => {
    render(<Harness />);

    fireEvent.click(screen.getByTestId("tasks-create-modal-template-recurring"));
    expect(screen.getByTestId("tasks-create-modal-template-recurring")).toHaveAttribute(
      "aria-pressed",
      "true"
    );

    expect(screen.getByTestId("tasks-create-modal-notice").textContent ?? "").toMatch(
      /Saves as a draft/i
    );
    expect(screen.getByTestId("tasks-create-modal-submit").textContent ?? "").toMatch(
      /Create task/i
    );
  });
});
