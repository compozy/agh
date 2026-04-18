import { render } from "@testing-library/react";
import { MotionConfig } from "motion/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { StatusDot, type StatusDotTone } from "./status-dot";

function WithMotion({
  reducedMotion,
  children,
}: {
  reducedMotion: "always" | "never";
  children: ReactNode;
}) {
  return <MotionConfig reducedMotion={reducedMotion}>{children}</MotionConfig>;
}

const TONE_TO_CSS_COLOR: Record<StatusDotTone, string> = {
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
  accent: "var(--color-accent)",
  neutral: "var(--color-text-tertiary)",
};

describe("StatusDot", () => {
  it("Should render a neutral dot by default", () => {
    const { container } = render(<StatusDot />);
    const dot = container.querySelector('[data-slot="status-dot"]');
    expect(dot).not.toBeNull();
    expect(dot?.getAttribute("data-tone")).toBe("neutral");
    expect(dot?.getAttribute("aria-hidden")).toBe("true");
    expect((dot as HTMLElement).style.backgroundColor).toBe(TONE_TO_CSS_COLOR.neutral);
  });

  it.each<StatusDotTone>(["success", "warning", "danger", "info", "accent", "neutral"])(
    "Should map tone %s to the semantic color token",
    tone => {
      const { container } = render(
        <WithMotion reducedMotion="never">
          <StatusDot tone={tone} />
        </WithMotion>
      );
      const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
      expect(dot?.getAttribute("data-tone")).toBe(tone);
      expect(dot?.style.backgroundColor).toBe(TONE_TO_CSS_COLOR[tone]);
    }
  );

  it("Should apply the pulse animation class when pulse is true and reduced motion is off", () => {
    const { container } = render(
      <WithMotion reducedMotion="never">
        <StatusDot tone="success" pulse />
      </WithMotion>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(dot?.className).toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBe("true");
  });

  it("Should not animate when pulse is false", () => {
    const { container } = render(
      <WithMotion reducedMotion="never">
        <StatusDot tone="success" />
      </WithMotion>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(dot?.className).not.toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBeNull();
  });

  it("Should suppress pulse animation when prefers-reduced-motion is reduce", () => {
    const { container } = render(
      <WithMotion reducedMotion="always">
        <StatusDot tone="success" pulse />
      </WithMotion>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(dot?.className).not.toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBeNull();
  });

  it("Should render the compact size variant", () => {
    const { container } = render(<StatusDot size="sm" />);
    const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(dot?.getAttribute("data-size")).toBe("sm");
    expect(dot?.className).toContain("size-1.5");
  });
});
