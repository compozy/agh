import * as React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../dialog";
import { Button } from "../button";

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

function RerenderingDialogExample() {
  const [value, setValue] = React.useState("");

  return (
    <Dialog open onOpenChange={() => undefined}>
      <DialogContent showCloseButton={false}>
        <DialogTitle>Stable dialog</DialogTitle>
        <input
          aria-label="stable-name"
          value={value}
          onChange={event => setValue(event.target.value)}
        />
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

  it("Should keep the same dialog node mounted across controlled rerenders while open", async () => {
    const user = userEvent.setup();
    render(<RerenderingDialogExample />);

    const initialDialog = screen.getByRole("dialog");
    await user.type(screen.getByLabelText("stable-name"), "abc");

    expect(screen.getByRole("dialog")).toBe(initialDialog);
  });

  it("Should mount a dialog-overlay slot when the dialog is open", async () => {
    render(<DialogExample defaultOpen />);
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const overlay = document.body.querySelector(
      "[data-slot='dialog-overlay']"
    ) as HTMLElement | null;

    expect(overlay).not.toBeNull();
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

  it("Should expose the footer close action through the dialog-close slot", async () => {
    const user = userEvent.setup();
    render(
      <Dialog defaultOpen>
        <DialogContent showCloseButton={false}>
          <DialogTitle>Footer close</DialogTitle>
          <DialogFooter showCloseButton />
        </DialogContent>
      </Dialog>
    );
    expect(document.body.querySelector('[data-slot="dialog-close"]')).not.toBeNull();
    await user.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument(), {
      timeout: 1500,
    });
  });

  it("Should expose the ruled DialogHeader variant via data attribute", async () => {
    render(
      <Dialog defaultOpen>
        <DialogContent unframed>
          <DialogHeader variant="ruled" data-testid="ruled-header">
            <DialogTitle>Add workspace</DialogTitle>
            <DialogDescription>Description.</DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const header = screen.getByTestId("ruled-header");
    expect(header.dataset.variant).toBe("ruled");
  });

  it("Should default the DialogHeader variant when none is provided", async () => {
    render(
      <Dialog defaultOpen>
        <DialogContent>
          <DialogHeader data-testid="default-header">
            <DialogTitle>Default</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const header = screen.getByTestId("default-header");
    expect(header.dataset.variant).toBe("default");
  });

  it("Should expose the ruled DialogFooter variant via data attribute", async () => {
    render(
      <Dialog defaultOpen>
        <DialogContent unframed showCloseButton={false}>
          <DialogTitle>Footer ruled</DialogTitle>
          <DialogFooter variant="ruled" data-testid="ruled-footer">
            <DialogClose render={<Button>Done</Button>} />
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const footer = screen.getByTestId("ruled-footer");
    expect(footer.dataset.variant).toBe("ruled");
  });

  it("Should reflect data-frame=unframed when DialogContent unframed is set", async () => {
    render(
      <Dialog defaultOpen>
        <DialogContent unframed showCloseButton={false}>
          <DialogTitle>Unframed</DialogTitle>
        </DialogContent>
      </Dialog>
    );

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const dialog = screen.getByRole("dialog");
    expect(dialog.dataset.frame).toBe("unframed");
  });

  it("Should default to data-frame=framed when unframed is omitted", async () => {
    render(
      <Dialog defaultOpen>
        <DialogContent>
          <DialogTitle>Framed</DialogTitle>
        </DialogContent>
      </Dialog>
    );

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    const dialog = screen.getByRole("dialog");
    expect(dialog.dataset.frame).toBe("framed");
  });

  it("Should throw when DialogContent is rendered outside <Dialog>", () => {
    const originalError = console.error;
    try {
      console.error = () => {};
      expect(() =>
        render(
          <DialogContent>
            <DialogTitle>orphan</DialogTitle>
          </DialogContent>
        )
      ).toThrow(/Dialog\.\* components must be used inside <Dialog>/);
    } finally {
      console.error = originalError;
    }
  });
});
