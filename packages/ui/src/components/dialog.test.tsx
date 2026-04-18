import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from "./dialog";
import { Button } from "./button";

function DialogExample({ defaultOpen = false }: { defaultOpen?: boolean }) {
  return (
    <Dialog defaultOpen={defaultOpen}>
      <DialogTrigger render={<Button>Open</Button>} />
      <DialogContent>
        <DialogTitle>Rename task</DialogTitle>
        <DialogDescription>Change the display name of the selected task.</DialogDescription>
        <input aria-label="name" defaultValue="task" />
        <DialogClose render={<Button>Confirm</Button>} />
      </DialogContent>
    </Dialog>
  );
}

describe("Dialog", () => {
  it("Should render the trigger without opening the content", () => {
    render(<DialogExample />);
    expect(screen.getByRole("button", { name: "Open" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("Should open on trigger click and render title + description", async () => {
    const user = userEvent.setup();
    render(<DialogExample />);
    await user.click(screen.getByRole("button", { name: "Open" }));
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(screen.getByText("Rename task")).toBeInTheDocument();
    expect(screen.getByText("Change the display name of the selected task.")).toBeInTheDocument();
  });

  it("Should close on Escape key press", async () => {
    const user = userEvent.setup();
    render(<DialogExample defaultOpen />);
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument(), {
      timeout: 1500,
    });
  });

  it("Should render a default close button that dismisses the dialog", async () => {
    const user = userEvent.setup();
    render(<DialogExample defaultOpen />);
    const closeButton = screen.getByRole("button", { name: "Close" });
    await user.click(closeButton);
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument(), {
      timeout: 1500,
    });
  });

  it("Should hide the default close button when showCloseButton=false", () => {
    render(
      <Dialog defaultOpen>
        <DialogContent showCloseButton={false}>
          <DialogTitle>No close</DialogTitle>
        </DialogContent>
      </Dialog>
    );
    expect(screen.queryByRole("button", { name: "Close" })).not.toBeInTheDocument();
  });

  it("Should throw when DialogContent is rendered outside <Dialog>", () => {
    const originalError = console.error;
    console.error = () => {};
    expect(() =>
      render(
        <DialogContent>
          <DialogTitle>orphan</DialogTitle>
        </DialogContent>
      )
    ).toThrow(/Dialog\.\* components must be used inside <Dialog>/);
    console.error = originalError;
  });
});
