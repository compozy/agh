import { render } from "@testing-library/react";
import { MotionConfig } from "motion/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { ConnectionIndicator, type ConnectionStatus } from "./connection-indicator";

function WithMotion({
  reducedMotion,
  children,
}: {
  reducedMotion: "always" | "never";
  children: ReactNode;
}) {
  return <MotionConfig reducedMotion={reducedMotion}>{children}</MotionConfig>;
}

describe("ConnectionIndicator", () => {
  it.each<{
    status: ConnectionStatus;
    tone: string;
    label: string;
  }>([
    { status: "connected", tone: "success", label: "Connected" },
    { status: "disconnected", tone: "danger", label: "Disconnected" },
    { status: "reconnecting", tone: "warning", label: "Reconnecting" },
  ])(
    "Should compose a StatusDot with the correct tone + default label for $status",
    ({ status, tone, label }) => {
      const { container } = render(<ConnectionIndicator status={status} />);
      const root = container.querySelector<HTMLElement>('[data-slot="connection-indicator"]');
      expect(root?.getAttribute("data-status")).toBe(status);
      const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
      expect(dot?.getAttribute("data-tone")).toBe(tone);
      const labelNode = container.querySelector<HTMLElement>(
        '[data-slot="connection-indicator-label"]'
      );
      expect(labelNode?.textContent).toBe(label);
    }
  );

  it("Should pulse the dot while reconnecting", () => {
    const { container } = render(
      <WithMotion reducedMotion="never">
        <ConnectionIndicator status="reconnecting" />
      </WithMotion>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(dot?.className).toContain("animate-pulse");
  });

  it("Should not pulse the dot while connected or disconnected", () => {
    const { container: connected } = render(
      <WithMotion reducedMotion="never">
        <ConnectionIndicator status="connected" />
      </WithMotion>
    );
    expect(
      connected.querySelector<HTMLElement>('[data-slot="status-dot"]')?.className
    ).not.toContain("animate-pulse");
    const { container: disconnected } = render(
      <WithMotion reducedMotion="never">
        <ConnectionIndicator status="disconnected" />
      </WithMotion>
    );
    expect(
      disconnected.querySelector<HTMLElement>('[data-slot="status-dot"]')?.className
    ).not.toContain("animate-pulse");
  });

  it("Should suppress the pulse when prefers-reduced-motion is reduce", () => {
    const { container } = render(
      <WithMotion reducedMotion="always">
        <ConnectionIndicator status="reconnecting" />
      </WithMotion>
    );
    expect(
      container.querySelector<HTMLElement>('[data-slot="status-dot"]')?.className
    ).not.toContain("animate-pulse");
  });

  it("Should allow overriding the default label", () => {
    const { container } = render(<ConnectionIndicator status="connected" label="Live" />);
    const labelNode = container.querySelector<HTMLElement>(
      '[data-slot="connection-indicator-label"]'
    );
    expect(labelNode?.textContent).toBe("Live");
  });
});
