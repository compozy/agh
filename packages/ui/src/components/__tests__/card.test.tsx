import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Card } from "../card";

describe("Card", () => {
  it("Should render flat on --canvas-soft with no default --line ring or border (ADR-004 §5)", () => {
    const { container } = render(<Card>body</Card>);
    const root = container.querySelector<HTMLElement>('[data-slot="card"]');
    expect(root).not.toBeNull();
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).not.toContain("ring-(--line)");
    expect(root?.className).not.toContain("border-(--line)");
  });

  it("Should still render the active-rail accent bar when activeRail is set", () => {
    const { container } = render(<Card activeRail>body</Card>);
    const root = container.querySelector<HTMLElement>('[data-slot="card"]');
    expect(root?.getAttribute("data-active-rail")).toBe("true");
    expect(root?.className).toContain("before:bg-(--accent)");
  });
});
