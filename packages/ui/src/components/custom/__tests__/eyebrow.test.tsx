import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Eyebrow } from "../eyebrow";

describe("Eyebrow", () => {
  it("Should default to sentence-case Inter 12px", () => {
    render(<Eyebrow>Queue health</Eyebrow>);
    const eyebrow = screen.getByText("Queue health");

    expect(eyebrow).toHaveAttribute("data-slot", "eyebrow");
    expect(eyebrow).toHaveAttribute("data-tone", "neutral");
    expect(eyebrow).toHaveAttribute("data-case", "sentence");
    expect(eyebrow.className).toContain("font-sans");
    expect(eyebrow.className).toContain("text-[12px]");
    expect(eyebrow.className).not.toContain("uppercase");
    expect(eyebrow.className).toContain("text-(--muted)");
  });

  it("Should render JetBrains Mono 10.5px / 0.05em UPPERCASE for case='upper'", () => {
    render(<Eyebrow case="upper">Run state</Eyebrow>);
    const eyebrow = screen.getByText("Run state");

    expect(eyebrow).toHaveAttribute("data-case", "upper");
    expect(eyebrow.className).toContain("font-mono");
    expect(eyebrow.className).toContain("text-[10.5px]");
    expect(eyebrow.className).toContain("uppercase");
    expect(eyebrow.className).toContain("tracking-[0.05em]");
  });

  it("Should expose weight as a controlled typography variant", () => {
    render(<Eyebrow weight="medium">Run state</Eyebrow>);
    const eyebrow = screen.getByText("Run state");

    expect(eyebrow).toHaveAttribute("data-weight", "medium");
    expect(eyebrow.className).toContain("font-medium");
    expect(eyebrow.className).not.toContain("font-semibold");
  });

  it("Should map signal tones to semantic text tokens", () => {
    render(<Eyebrow tone="danger">Blocked</Eyebrow>);
    const eyebrow = screen.getByText("Blocked");

    expect(eyebrow).toHaveAttribute("data-tone", "danger");
    expect(eyebrow.className).toContain("text-(--danger)");
  });
});
