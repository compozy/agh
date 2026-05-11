import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PriorityBars, type PriorityLevel } from "../priority-bars";

const LEVEL_FILL_CLASS: Record<PriorityLevel, string> = {
  low: "bg-(--faint)",
  medium: "bg-(--fg)",
  high: "bg-(--warning)",
  urgent: "bg-(--danger)",
};

describe("PriorityBars", () => {
  it("Should render exactly three bars per ADR-006 §4 (count never varies with level)", () => {
    for (const level of ["low", "medium", "high", "urgent"] as const) {
      const { container } = render(<PriorityBars level={level} />);
      const bars = container.querySelectorAll('[data-slot="priority-bars-bar"]');
      expect(bars).toHaveLength(3);
    }
  });

  it("Should paint every bar with the level-derived signal color (color from level, not fill count)", () => {
    for (const level of ["low", "medium", "high", "urgent"] as const) {
      const { container } = render(<PriorityBars level={level} />);
      const bars = Array.from(container.querySelectorAll('[data-slot="priority-bars-bar"]'));
      for (const bar of bars) {
        expect(bar.className).toContain(LEVEL_FILL_CLASS[level]);
      }
    }
  });

  it("Should render bars with ascending heights 4 / 8 / 12 px (Tailwind h-1 / h-2 / h-3)", () => {
    const { container } = render(<PriorityBars level="urgent" />);
    const bars = Array.from(container.querySelectorAll('[data-slot="priority-bars-bar"]'));
    expect(bars[0]?.className).toMatch(/\bh-1\b/);
    expect(bars[1]?.className).toMatch(/\bh-2\b/);
    expect(bars[2]?.className).toMatch(/\bh-3\b/);
  });

  it("Should expose role=img and aria-label='{level} priority' by default per ADR-006 §4", () => {
    const { container } = render(<PriorityBars level="high" />);
    const root = container.querySelector('[data-slot="priority-bars"]');
    expect(root).toHaveAttribute("role", "img");
    expect(root).toHaveAttribute("aria-label", "high priority");
    expect(root).toHaveAttribute("data-level", "high");
  });

  it("Should honor a caller-provided aria-label override", () => {
    const { container } = render(<PriorityBars ariaLabel="Critical" level="urgent" />);
    const root = container.querySelector('[data-slot="priority-bars"]');
    expect(root).toHaveAttribute("aria-label", "Critical");
  });
});
