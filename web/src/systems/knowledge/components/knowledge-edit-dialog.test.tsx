import { UIProvider } from "@agh/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { KnowledgeEditDialog } from "./knowledge-edit-dialog";

function renderDialog(props: Partial<React.ComponentProps<typeof KnowledgeEditDialog>> = {}) {
  const merged: React.ComponentProps<typeof KnowledgeEditDialog> = {
    open: true,
    onOpenChange: vi.fn(),
    filename: "user.md",
    scope: "global",
    initialContent: "# Initial content",
    initialDescription: "initial description",
    isPending: false,
    onConfirm: vi.fn(),
    ...props,
  };
  return render(
    <UIProvider reducedMotion="always">
      <KnowledgeEditDialog {...merged} />
    </UIProvider>
  );
}

describe("KnowledgeEditDialog", () => {
  it("Should render the initial content and description", () => {
    renderDialog();
    expect(screen.getByTestId("knowledge-edit-content")).toHaveValue("# Initial content");
    expect(screen.getByTestId("knowledge-edit-description")).toHaveValue("initial description");
  });

  it("Should disable the confirm button until content changes", () => {
    renderDialog();
    expect(screen.getByTestId("confirm-edit-memory-btn")).toBeDisabled();
  });

  it("Should call onConfirm with the edited content and trimmed description", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    renderDialog({ onConfirm });

    await user.type(screen.getByTestId("knowledge-edit-content"), " more body");
    await user.click(screen.getByTestId("confirm-edit-memory-btn"));

    expect(onConfirm).toHaveBeenCalledWith({
      content: "# Initial content more body",
      description: "initial description",
    });
  });

  it("Should send undefined description when the field is cleared", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    renderDialog({ onConfirm });

    await user.clear(screen.getByTestId("knowledge-edit-description"));
    await user.type(screen.getByTestId("knowledge-edit-content"), " more");
    await user.click(screen.getByTestId("confirm-edit-memory-btn"));

    expect(onConfirm).toHaveBeenCalledWith({
      content: "# Initial content more",
      description: undefined,
    });
  });

  it("Should disable the confirm button while a save is pending", async () => {
    const user = userEvent.setup();
    renderDialog({ isPending: true });
    await user.type(screen.getByTestId("knowledge-edit-content"), " edit");
    expect(screen.getByTestId("confirm-edit-memory-btn")).toBeDisabled();
  });

  it("Should surface the dialog error message inside the dialog", () => {
    renderDialog({ error: "Edit rejected" });
    expect(screen.getByTestId("knowledge-edit-dialog-error")).toHaveTextContent("Edit rejected");
  });

  it("Should call onOpenChange(false) when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });
    await user.click(screen.getByTestId("cancel-edit-memory-btn"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
