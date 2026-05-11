import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KpiCard } from "../kpi-card";

describe("KpiCard", () => {
  it("Should render the label and value content", () => {
    const { container } = render(<KpiCard label="Active runs" value="14" />);
    const value = container.querySelector<HTMLElement>('[data-slot="kpi-card-value"]');
    expect(value?.textContent).toBe("14");

    const label = container.querySelector<HTMLElement>('[data-slot="kpi-card-label"]');
    expect(label?.dataset.slot).toBe("kpi-card-label");
    expect(label?.textContent).toBe("Active runs");
  });

  it("Should render an optional detail line", () => {
    render(<KpiCard label="Active runs" value="14" detail="3 since yesterday" />);
    expect(screen.getByText("3 since yesterday")).toBeInTheDocument();
  });
});
