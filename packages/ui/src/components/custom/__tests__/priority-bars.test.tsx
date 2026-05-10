import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PriorityBars } from "../priority-bars";

describe("PriorityBars", () => {
  it("Should render four bars and fill them by level", () => {
    const { container } = render(<PriorityBars level="medium" tone="warning" />);
    const bars = Array.from(container.querySelectorAll('[data-slot="priority-bars-bar"]'));
    expect(bars).toHaveLength(4);
    const filled = bars.filter(bar => bar.getAttribute("data-filled") === "true");
    expect(filled).toHaveLength(2);
  });
});
