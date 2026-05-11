import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Metric, type MetricTone } from "../metric";

describe("Metric", () => {
  it("Should render label as the canonical Eyebrow + value with the mock-specified typographic scale", () => {
    const { container } = render(<Metric label="Sessions" value="12" />);
    const root = container.querySelector<HTMLElement>('[data-slot="metric"]');
    expect(root).not.toBeNull();

    const label = container.querySelector<HTMLElement>('[data-slot="metric-label"]');
    expect(label?.textContent).toBe("Sessions");
    expect(label?.className).toContain("eyebrow");
    expect(label?.className).toContain("text-(--subtle)");

    const value = container.querySelector<HTMLElement>('[data-slot="metric-value"]');
    expect(value?.textContent).toBe("12");
    expect(value?.className).toContain("text-[24px]");
    expect(value?.className).toContain("font-medium");
    expect(value?.className).toContain("tracking-[-0.02em]");
  });

  it.each<{ tone: MetricTone; color: string }>([
    { tone: "success", color: "var(--success)" },
    { tone: "danger", color: "var(--danger)" },
    { tone: "warning", color: "var(--warning)" },
    { tone: "accent", color: "var(--accent)" },
  ])("Should apply the $tone semantic color to the value", ({ tone, color }) => {
    const { container } = render(<Metric label="Signal" value="08" tone={tone} />);
    const value = container.querySelector<HTMLElement>('[data-slot="metric-value"]');
    expect(value?.style.color).toBe(color);
    const root = container.querySelector<HTMLElement>('[data-slot="metric"]');
    expect(root).not.toBeNull();
    expect(container.querySelector('[data-slot="metric"]')?.getAttribute("data-tone")).toBe(tone);
  });

  it("Should default the value color to text-primary", () => {
    const { container } = render(<Metric label="Sessions" value="12" />);
    const value = container.querySelector<HTMLElement>('[data-slot="metric-value"]');
    expect(value?.style.color).toBe("var(--fg)");
  });

  it("Should render the optional detail slot inline with the value", () => {
    const { container } = render(<Metric label="Credits" value="56.4%" detail="+4.2%" />);
    const detail = container.querySelector<HTMLElement>('[data-slot="metric-detail"]');
    expect(detail?.textContent).toBe("+4.2%");
    expect(detail?.className).toContain("font-mono");
    expect(detail?.className).toContain("text-eyebrow");
  });

  it("Should render the optional subtext line with Inter 13px secondary styling", () => {
    const { container } = render(
      <Metric label="Credits" value="56.4%" subtext="Across 8 sessions" />
    );
    const subtext = container.querySelector<HTMLElement>('[data-slot="metric-subtext"]');
    expect(subtext?.textContent).toBe("Across 8 sessions");
    expect(subtext?.className).toContain("text-[13px]");
    expect(subtext?.className).toContain("text-(--muted)");
  });

  it("Should omit the detail and subtext slots when those props are absent", () => {
    const { container } = render(<Metric label="Credits" value="0" />);
    expect(container.querySelector('[data-slot="metric-detail"]')).toBeNull();
    expect(container.querySelector('[data-slot="metric-subtext"]')).toBeNull();
  });

  it("Should render flat — no default border on the card root (ADR-004 §5)", () => {
    const { container } = render(<Metric label="Sessions" value="12" />);
    const root = container.querySelector<HTMLElement>('[data-slot="metric"]');
    expect(root?.className).not.toContain("border-(--line)");
    expect(root?.className).toContain("bg-(--canvas-soft)");
  });
});
