import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Toolbar } from "../toolbar";

describe("Toolbar", () => {
  it("Should render children inside a role=toolbar element and wrap on narrow viewports", () => {
    const { container } = render(
      <Toolbar data-testid="tb">
        <button type="button">a</button>
        <button type="button">b</button>
      </Toolbar>
    );
    const toolbar = screen.getByTestId("tb");
    expect(toolbar).toHaveAttribute("role", "toolbar");
    expect(toolbar.className).toContain("flex-wrap");
    expect(container.querySelectorAll("button")).toHaveLength(2);
  });

  it("Should expose data-sticky and sticky class when sticky=true", () => {
    render(
      <Toolbar sticky data-testid="tb">
        <span>x</span>
      </Toolbar>
    );
    const toolbar = screen.getByTestId("tb");
    expect(toolbar).toHaveAttribute("data-sticky", "true");
    expect(toolbar.className).toContain("sticky");
  });
});
