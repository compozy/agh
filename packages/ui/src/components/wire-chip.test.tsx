import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WireChip } from "./wire-chip";

describe("WireChip", () => {
  it("Should render as a button with neutral chrome by default", () => {
    const { container } = render(<WireChip>say</WireChip>);
    const chip = container.querySelector<HTMLElement>('[data-slot="wire-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.tagName).toBe("BUTTON");
    expect(chip?.className).toContain("bg-[color:var(--color-surface)]");
    expect(chip?.className).toContain("border-[color:var(--color-divider)]");
    expect(chip?.getAttribute("aria-pressed")).toBe("false");
  });

  it("Should reflect the active state via aria-pressed and elevated surface", () => {
    const { container } = render(<WireChip active>direct</WireChip>);
    const chip = container.querySelector<HTMLElement>('[data-slot="wire-chip"]');
    expect(chip?.getAttribute("aria-pressed")).toBe("true");
    expect(chip?.getAttribute("data-active")).toBe("true");
    expect(chip?.className).toContain("bg-[color:var(--color-surface-elevated)]");
  });

  it("Should render a colored leading dot when dotColor is provided", () => {
    const { container } = render(<WireChip dotColor="var(--color-accent)">direct</WireChip>);
    const dot = container.querySelector<HTMLElement>('[data-slot="wire-chip-dot"]');
    expect(dot).not.toBeNull();
    expect(dot?.style.background).toBe("var(--color-accent)");
  });

  it("Should fire onClick when activated", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(<WireChip onClick={handle}>say</WireChip>);

    await user.click(screen.getByRole("button", { name: /say/i }));

    expect(handle).toHaveBeenCalledTimes(1);
  });
});
