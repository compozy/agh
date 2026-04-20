import { UIProvider } from "@agh/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { KnowledgeDeleteDialog } from "./knowledge-delete-dialog";

describe("KnowledgeDeleteDialog", () => {
  it("does not render the dialog body when open is false", () => {
    render(
      <UIProvider reducedMotion="always">
        <KnowledgeDeleteDialog
          filename="global/user.md"
          isPending={false}
          onConfirm={vi.fn()}
          onOpenChange={vi.fn()}
          open={false}
          scope="global"
        />
      </UIProvider>
    );
    expect(screen.queryByTestId("knowledge-delete-dialog")).not.toBeInTheDocument();
  });

  it("renders the filename and scope in the description when open", () => {
    render(
      <UIProvider reducedMotion="always">
        <KnowledgeDeleteDialog
          filename="workspace/project-context.md"
          isPending={false}
          onConfirm={vi.fn()}
          onOpenChange={vi.fn()}
          open
          scope="workspace"
        />
      </UIProvider>
    );
    expect(screen.getByTestId("knowledge-delete-dialog")).toBeInTheDocument();
    expect(screen.getByText(/workspace\/project-context\.md/)).toBeInTheDocument();
    expect(screen.getByText(/workspace scope/)).toBeInTheDocument();
  });

  it("calls onConfirm when confirm is clicked", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <KnowledgeDeleteDialog
          filename="global/user.md"
          isPending={false}
          onConfirm={onConfirm}
          onOpenChange={vi.fn()}
          open
          scope="global"
        />
      </UIProvider>
    );
    await user.click(screen.getByTestId("confirm-delete-memory-btn"));
    expect(onConfirm).toHaveBeenCalled();
  });

  it("calls onOpenChange(false) when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <KnowledgeDeleteDialog
          filename="global/user.md"
          isPending={false}
          onConfirm={vi.fn()}
          onOpenChange={onOpenChange}
          open
          scope="global"
        />
      </UIProvider>
    );
    await user.click(screen.getByTestId("cancel-delete-memory-btn"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("disables the confirm button while a delete is pending", () => {
    render(
      <UIProvider reducedMotion="always">
        <KnowledgeDeleteDialog
          filename="global/user.md"
          isPending
          onConfirm={vi.fn()}
          onOpenChange={vi.fn()}
          open
          scope="global"
        />
      </UIProvider>
    );
    expect(screen.getByTestId("confirm-delete-memory-btn")).toBeDisabled();
  });
});
