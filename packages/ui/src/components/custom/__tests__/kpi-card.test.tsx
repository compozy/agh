import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KpiCard } from "../kpi-card";

describe("KpiCard", () => {
  it("Should render the value at 28px and an Inter UC eyebrow label", () => {
    const { container } = render(<KpiCard label="Active runs" value="14" />);
    const value = container.querySelector<HTMLElement>('[data-slot="kpi-card-value"]');
    expect(value?.className).toContain("text-[28px]");

    const label = container.querySelector<HTMLElement>('[data-slot="kpi-card-label"]');
    expect(label?.dataset.slot).toBe("kpi-card-label");
    expect(label?.className).toContain("eyebrow");
    expect(label?.className).toContain("text-(--muted)");
    expect(label?.className).not.toContain("text-[10.5px]");
    expect(label?.className).not.toContain("tracking-[0.05em]");
  });

  it("Should render an optional detail line", () => {
    render(<KpiCard label="Active runs" value="14" detail="3 since yesterday" />);
    expect(screen.getByText("3 since yesterday")).toBeInTheDocument();
  });

  it("Should render flat on --canvas-soft with no border", () => {
    const { container } = render(<KpiCard label="Active runs" value="14" />);
    const root = container.querySelector<HTMLElement>('[data-slot="kpi-card"]');
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).not.toContain("border-(--line)");
    expect(root?.className).not.toContain("ring-(--line)");
  });
});
