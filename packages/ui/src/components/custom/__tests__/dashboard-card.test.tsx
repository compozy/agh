import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DashboardCard } from "../dashboard-card";

describe("DashboardCard", () => {
  it("Should render the value at 28px and an upper-case mono Eyebrow label by default", () => {
    const { container } = render(<DashboardCard label="Active runs" value="14" />);
    const value = container.querySelector<HTMLElement>('[data-slot="dashboard-card-value"]');
    expect(value?.className).toContain("text-[28px]");

    const label = container.querySelector<HTMLElement>('[data-slot="dashboard-card-label"]');
    expect(label?.dataset.slot).toBe("dashboard-card-label");
    expect(label?.dataset.case).toBe("upper");
    expect(label?.dataset.tone).toBe("muted");
    expect(label?.className).toContain("uppercase");
    expect(label?.className).toContain("font-mono");
    expect(label?.className).toContain("tracking-mono");
    expect(label?.className).toContain("text-eyebrow");
    expect(label?.className).not.toContain("text-[10.5px]");
    expect(label?.className).not.toContain("tracking-[0.05em]");
  });

  it("Should render an optional detail line", () => {
    render(<DashboardCard label="Active runs" value="14" detail="3 since yesterday" />);
    expect(screen.getByText("3 since yesterday")).toBeInTheDocument();
  });
});
