import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../tooltip";
import { Button } from "../button";

function TooltipExample({ delay = 0 }: { delay?: number }) {
  return (
    <TooltipProvider delay={delay}>
      <Tooltip>
        <TooltipTrigger render={<Button>Target</Button>} />
        <TooltipContent>Keyboard shortcut: ⌘K</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

describe("Tooltip", () => {
  it("Should not render the tooltip content until the trigger is hovered/focused", () => {
    render(<TooltipExample />);
    expect(screen.queryByText(/Keyboard shortcut/)).not.toBeInTheDocument();
  });

  it("Should open on focus and render the content", async () => {
    const user = userEvent.setup();
    render(<TooltipExample delay={0} />);
    await user.tab();
    await waitFor(() => expect(screen.getByText(/Keyboard shortcut/)).toBeInTheDocument());
  });

  it("Should close when the trigger is blurred", async () => {
    const user = userEvent.setup();
    render(<TooltipExample delay={0} />);
    await user.tab();
    await waitFor(() => expect(screen.getByText(/Keyboard shortcut/)).toBeInTheDocument());
    await user.tab();
    await waitFor(() => expect(screen.queryByText(/Keyboard shortcut/)).not.toBeInTheDocument(), {
      timeout: 1500,
    });
  });

  it("Should throw when TooltipContent is used outside <Tooltip>", () => {
    const originalError = console.error;
    console.error = () => {};
    expect(() =>
      render(
        <TooltipProvider>
          <TooltipContent>orphan</TooltipContent>
        </TooltipProvider>
      )
    ).toThrow(/Tooltip\.\* components must be used inside <Tooltip>/);
    console.error = originalError;
  });

  it("Should paint tooltip content on var(--canvas-soft) with a 1px line-soft ring", async () => {
    render(
      <TooltipProvider>
        <Tooltip open>
          <TooltipTrigger render={<Button>Open</Button>} />
          <TooltipContent>tooltip body</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
    await waitFor(() => expect(screen.getByText("tooltip body")).toBeInTheDocument());
    const content = document.body.querySelector(
      "[data-slot='tooltip-content']"
    ) as HTMLElement | null;
    expect(content?.className).toContain("bg-(--canvas-soft)");
    expect(content?.className).toContain("shadow-[0_0_0_1px_var(--line-soft)]");
  });

  it("Should respect a controlled open prop", async () => {
    const { rerender } = render(
      <TooltipProvider>
        <Tooltip open={false}>
          <TooltipTrigger render={<Button>Trigger</Button>} />
          <TooltipContent>controlled body</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
    expect(screen.queryByText("controlled body")).not.toBeInTheDocument();
    rerender(
      <TooltipProvider>
        <Tooltip open={true}>
          <TooltipTrigger render={<Button>Trigger</Button>} />
          <TooltipContent>controlled body</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
    await waitFor(() => expect(screen.getByText("controlled body")).toBeInTheDocument());
  });
});
