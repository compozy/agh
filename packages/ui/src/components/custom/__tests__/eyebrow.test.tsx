import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Eyebrow } from "../eyebrow";

function classes(node: HTMLElement): string[] {
  return node.className.split(/\s+/).filter(Boolean);
}

describe("Eyebrow", () => {
  it("Should render the single eyebrow utility (Inter UC 11/600/-0.005em) with no variant props", () => {
    render(<Eyebrow>Run state</Eyebrow>);
    const eyebrow = screen.getByText("Run state");

    expect(eyebrow).toHaveAttribute("data-slot", "eyebrow");
    expect(classes(eyebrow)).toEqual(["eyebrow"]);
    expect(classes(eyebrow)).not.toContain("eyebrow-badge");
    expect(classes(eyebrow)).not.toContain("eyebrow-micro");
    expect(classes(eyebrow)).not.toContain("font-mono");
    expect(classes(eyebrow)).not.toContain("font-medium");
    expect(classes(eyebrow)).not.toContain("text-[12px]");
  });

  it("Should merge a consumer className while keeping the eyebrow utility first", () => {
    render(<Eyebrow className="text-(--muted)">Queue health</Eyebrow>);
    const eyebrow = screen.getByText("Queue health");

    expect(classes(eyebrow)).toContain("eyebrow");
    expect(classes(eyebrow)).toContain("text-(--muted)");
  });

  it("Should forward span props (data-testid, aria, id) onto the rendered node", () => {
    render(
      <Eyebrow aria-label="meta" data-testid="eyebrow-target" id="eyebrow-id">
        Meta
      </Eyebrow>
    );
    const eyebrow = screen.getByTestId("eyebrow-target");

    expect(eyebrow.tagName).toBe("SPAN");
    expect(eyebrow).toHaveAttribute("id", "eyebrow-id");
    expect(eyebrow).toHaveAttribute("aria-label", "meta");
  });
});
