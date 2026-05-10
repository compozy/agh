import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Eyebrow } from "../eyebrow";

describe("Eyebrow", () => {
  it("Should default to sentence-case Inter at the eyebrow size", () => {
    render(<Eyebrow>Queue health</Eyebrow>);
    const eyebrow = screen.getByText("Queue health");

    expect(eyebrow).toHaveAttribute("data-slot", "eyebrow");
    expect(eyebrow).toHaveAttribute("data-tone", "neutral");
    expect(eyebrow).toHaveAttribute("data-case", "sentence");
    expect(eyebrow).toHaveAttribute("data-size", "eyebrow");
    expect(eyebrow.className).toContain("font-sans");
    expect(eyebrow.className).toContain("text-[12px]");
    expect(eyebrow.className).not.toContain("uppercase");
    expect(eyebrow.className).toContain("text-(--muted)");
  });

  it("Should render JetBrains Mono at the canonical eyebrow token for case='upper'", () => {
    render(<Eyebrow case="upper">Run state</Eyebrow>);
    const eyebrow = screen.getByText("Run state");

    expect(eyebrow).toHaveAttribute("data-case", "upper");
    expect(eyebrow.className).toContain("font-mono");
    expect(eyebrow.className).toContain("uppercase");
    expect(eyebrow.className).toContain("tracking-mono");
    expect(eyebrow.className).toContain("text-eyebrow");
    expect(eyebrow.className).not.toContain("tracking-[0.05em]");
    expect(eyebrow.className).not.toContain("text-[10.5px]");
  });

  it("Should switch to text-badge when size='badge' under upper-case", () => {
    render(
      <Eyebrow case="upper" size="badge">
        SLA
      </Eyebrow>
    );
    const eyebrow = screen.getByText("SLA");

    expect(eyebrow).toHaveAttribute("data-size", "badge");
    expect(eyebrow.className).toContain("text-badge");
    expect(eyebrow.className).not.toContain("text-eyebrow");
    expect(eyebrow.className).toContain("tracking-mono");
  });

  it("Should switch to text-micro when size='micro' under upper-case", () => {
    render(
      <Eyebrow case="upper" size="micro">
        Live
      </Eyebrow>
    );
    const eyebrow = screen.getByText("Live");

    expect(eyebrow).toHaveAttribute("data-size", "micro");
    expect(eyebrow.className).toContain("text-micro");
  });

  it("Should expose weight as a controlled typography variant", () => {
    render(<Eyebrow weight="semibold">Run state</Eyebrow>);
    const eyebrow = screen.getByText("Run state");

    expect(eyebrow).toHaveAttribute("data-weight", "semibold");
    expect(eyebrow.className).toContain("font-semibold");
    expect(eyebrow.className).not.toContain("font-medium");
  });

  it("Should map signal tones to semantic text tokens", () => {
    render(<Eyebrow tone="danger">Blocked</Eyebrow>);
    const eyebrow = screen.getByText("Blocked");

    expect(eyebrow).toHaveAttribute("data-tone", "danger");
    expect(eyebrow.className).toContain("text-(--danger)");
  });

  it("Should map subtle and strong tones to their text tokens", () => {
    const { rerender } = render(<Eyebrow tone="subtle">Quiet</Eyebrow>);
    expect(screen.getByText("Quiet").className).toContain("text-(--subtle)");

    rerender(<Eyebrow tone="strong">Loud</Eyebrow>);
    expect(screen.getByText("Loud").className).toContain("text-(--fg-strong)");
  });
});
