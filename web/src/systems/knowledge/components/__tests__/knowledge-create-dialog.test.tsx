import { UIProvider } from "@agh/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { KnowledgeCreateDialog } from "../knowledge-create-dialog";

function renderDialog(props: Partial<React.ComponentProps<typeof KnowledgeCreateDialog>> = {}) {
  const merged: React.ComponentProps<typeof KnowledgeCreateDialog> = {
    open: true,
    onOpenChange: vi.fn(),
    scope: "workspace",
    defaultType: "project",
    isPending: false,
    onConfirm: vi.fn(),
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeCreateDialog {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeCreateDialog", () => {
  it("Should render empty fields with the default type", () => {
    renderDialog();
    expect(screen.getByTestId("knowledge-create-type")).toHaveValue("project");
    expect(screen.getByTestId("knowledge-create-name")).toHaveValue("");
    expect(screen.getByTestId("knowledge-create-content")).toHaveValue("");
  });

  it("Should disable the confirm button until name and content are present", async () => {
    const user = userEvent.setup();
    renderDialog();

    expect(screen.getByTestId("confirm-create-memory-btn")).toBeDisabled();
    await user.type(screen.getByTestId("knowledge-create-name"), "Launch Memory");
    expect(screen.getByTestId("confirm-create-memory-btn")).toBeDisabled();
    await user.type(screen.getByTestId("knowledge-create-content"), "Use the launch playbook.");
    expect(screen.getByTestId("confirm-create-memory-btn")).toBeEnabled();
  });

  it("Should call onConfirm with trimmed structured input", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    renderDialog({ onConfirm });

    await user.selectOptions(screen.getByTestId("knowledge-create-type"), "reference");
    await user.type(screen.getByTestId("knowledge-create-name"), "  Launch Memory  ");
    await user.type(screen.getByTestId("knowledge-create-description"), "  contract  ");
    await user.type(screen.getByTestId("knowledge-create-content"), "Use the launch playbook.");
    await user.click(screen.getByTestId("confirm-create-memory-btn"));

    expect(onConfirm).toHaveBeenCalledWith({
      type: "reference",
      name: "Launch Memory",
      description: "contract",
      content: "Use the launch playbook.",
    });
  });

  it("Should preserve in-progress draft input when defaultType changes while open", async () => {
    const user = userEvent.setup();
    const view = renderDialog();

    await user.selectOptions(screen.getByTestId("knowledge-create-type"), "feedback");
    await user.type(screen.getByTestId("knowledge-create-name"), "Launch Memory");
    await user.type(screen.getByTestId("knowledge-create-description"), "workspace contract");
    await user.type(screen.getByTestId("knowledge-create-content"), "Use the launch playbook.");

    view.rerender(
      <UIProvider reducedMotion="always">
        <KnowledgeCreateDialog
          open
          onOpenChange={vi.fn()}
          scope="workspace"
          defaultType="reference"
          isPending={false}
          onConfirm={vi.fn()}
        />
      </UIProvider>
    );

    expect(screen.getByTestId("knowledge-create-type")).toHaveValue("feedback");
    expect(screen.getByTestId("knowledge-create-name")).toHaveValue("Launch Memory");
    expect(screen.getByTestId("knowledge-create-description")).toHaveValue("workspace contract");
    expect(screen.getByTestId("knowledge-create-content")).toHaveValue("Use the launch playbook.");
  });

  it("Should surface the dialog error", () => {
    renderDialog({ error: "Write rejected" });
    expect(screen.getByTestId("knowledge-create-dialog-error")).toHaveTextContent("Write rejected");
  });

  it("Should call onOpenChange(false) when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    await user.click(screen.getByTestId("cancel-create-memory-btn"));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
