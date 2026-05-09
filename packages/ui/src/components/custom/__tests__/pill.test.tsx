import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MotionConfig } from "motion/react";
import type { MouseEvent, ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Pill, type PillTone } from "../pill";

interface WithMotionProps {
  reducedMotion: "always" | "never";
  children: ReactNode;
}

interface ToneExpectation {
  tone: PillTone;
  bg: string;
  text: string;
}

function WithMotion({ reducedMotion, children }: WithMotionProps) {
  return <MotionConfig reducedMotion={reducedMotion}>{children}</MotionConfig>;
}

describe("Pill", () => {
  it("Should render a neutral span at sm size by default", () => {
    render(<Pill>label</Pill>);
    const pill = screen.getByText("label");
    expect(pill.tagName).toBe("SPAN");
    expect(pill).toHaveAttribute("data-slot", "pill");
    expect(pill).toHaveAttribute("data-tone", "neutral");
    expect(pill).toHaveAttribute("data-size", "sm");
    expect(pill.className).toContain("bg-(--color-neutral-tint)");
    expect(pill.className).toContain("text-(--color-text-secondary)");
  });

  it.each<ToneExpectation>([
    { tone: "accent", bg: "bg-(--color-accent-tint)", text: "text-(--color-accent)" },
    { tone: "success", bg: "bg-(--color-success-tint)", text: "text-(--color-success)" },
    { tone: "warning", bg: "bg-(--color-warning-tint)", text: "text-(--color-warning)" },
    { tone: "danger", bg: "bg-(--color-danger-tint)", text: "text-(--color-danger)" },
    { tone: "info", bg: "bg-(--color-info-tint)", text: "text-(--color-info)" },
  ])("Should apply the $tone tint formula", ({ tone, bg, text }) => {
    render(<Pill tone={tone}>x</Pill>);
    const pill = screen.getByText("x");
    expect(pill).toHaveAttribute("data-tone", tone);
    expect(pill.className).toContain(bg);
    expect(pill.className).toContain(text);
  });

  it("Should switch to solid background and ink text when solid is true", () => {
    render(
      <Pill tone="accent" solid>
        NEW
      </Pill>
    );
    const pill = screen.getByText("NEW");
    expect(pill).toHaveAttribute("data-solid", "true");
    expect(pill.className).toContain("bg-(--color-accent)");
    expect(pill.className).toContain("text-(--color-accent-ink)");
  });

  it("Should adopt mono typography and uppercase when mono is true", () => {
    render(<Pill mono>token</Pill>);
    const pill = screen.getByText("token");
    expect(pill).toHaveAttribute("data-mono", "true");
    expect(pill.className).toContain("font-mono");
    expect(pill.className).toContain("uppercase");
  });

  it("Should respect uppercase={false} explicit override", () => {
    render(
      <Pill mono uppercase={false}>
        v1.2.3
      </Pill>
    );
    const pill = screen.getByText("v1.2.3");
    expect(pill.className).toContain("normal-case");
    expect(pill.className).not.toMatch(/(^| )uppercase( |$)/);
  });

  it("Should default xs size to non-uppercase chip chrome", () => {
    render(<Pill size="xs">capability-id</Pill>);
    const pill = screen.getByText("capability-id");
    expect(pill).toHaveAttribute("data-size", "xs");
    expect(pill.className).toContain("rounded-(--radius-chip)");
    expect(pill.className).toContain("normal-case");
  });

  it("Should apply md filter-pill chrome when size='md'", () => {
    render(<Pill size="md">FILTER</Pill>);
    const pill = screen.getByText("FILTER");
    expect(pill).toHaveAttribute("data-size", "md");
    expect(pill.className).toContain("h-8");
    expect(pill.className).toContain("font-semibold");
    expect(pill.className).toContain("uppercase");
  });

  it("Should render as a button when render={<button />} is provided", async () => {
    const user = userEvent.setup();
    const handle = vi.fn();
    render(<Pill render={<button type="button" onClick={handle} />}>FILTER</Pill>);
    const pill = screen.getByRole("button", { name: /filter/i });
    expect(pill.tagName).toBe("BUTTON");
    await user.click(pill);
    expect(handle).toHaveBeenCalledTimes(1);
  });

  it("Should override tone styling with toggle-on chrome when active=true", () => {
    render(
      <Pill mono active render={<button type="button" />}>
        FILTER
      </Pill>
    );
    const pill = screen.getByRole("button", { name: /filter/i });
    expect(pill).toHaveAttribute("data-active", "true");
    expect(pill.className).toContain("bg-(--color-surface-elevated)");
    expect(pill.className).toContain("text-(--color-text-primary)");
  });

  it("Should apply inactive interactive chrome when active=false", () => {
    render(
      <Pill mono active={false} render={<button type="button" />}>
        FILTER
      </Pill>
    );
    const pill = screen.getByRole("button", { name: /filter/i });
    expect(pill).toHaveAttribute("data-active", "false");
    expect(pill.className).toContain("bg-(--color-surface)");
    expect(pill.className).toContain("text-(--color-text-secondary)");
  });

  it("Should forward className alongside the variant defaults", () => {
    render(<Pill className="custom">label</Pill>);
    const pill = screen.getByText("label");
    expect(pill.className).toContain("custom");
    expect(pill.className).toContain("bg-(--color-neutral-tint)");
  });

  it("Should expose Pill.Link as an accessible anchor chip", async () => {
    const user = userEvent.setup();
    const handle = vi.fn((event: MouseEvent<HTMLAnchorElement>) => event.preventDefault());
    render(
      <Pill.Link href="/tasks/task-1" onClick={handle}>
        Open task
      </Pill.Link>
    );

    const link = screen.getByRole("link", { name: /open task/i });
    expect(link).toHaveAttribute("href", "/tasks/task-1");
    expect(link).toHaveAttribute("data-slot", "pill");
    expect(link).toHaveAttribute("data-tone", "accent");
    expect(link).toHaveAttribute("data-mono", "true");
    await user.click(link);
    expect(handle).toHaveBeenCalledTimes(1);
  });
});

describe("Pill.Dot", () => {
  it("Should render a 8px md dot keyed to neutral by default", () => {
    const { container } = render(<Pill.Dot />);
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot).not.toBeNull();
    expect(dot?.getAttribute("data-tone")).toBe("neutral");
    expect(dot?.getAttribute("data-size")).toBe("md");
    expect(dot?.className).toContain("size-2");
    expect(dot?.style.backgroundColor).toBe("var(--color-text-tertiary)");
    expect(dot?.getAttribute("aria-hidden")).toBe("true");
  });

  it.each<PillTone>(["accent", "success", "warning", "danger", "info"])(
    "Should map tone %s to the semantic color token",
    tone => {
      const { container } = render(<Pill.Dot tone={tone} />);
      const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
      expect(dot?.style.backgroundColor).toBe(`var(--color-${tone})`);
    }
  );

  it("Should let an explicit color override the tone-derived background", () => {
    const { container } = render(<Pill.Dot tone="success" color="#5BA6FF" />);
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.style.backgroundColor).toBe("rgb(91, 166, 255)");
  });

  it("Should pulse only when reduced motion is off", () => {
    const { container, rerender } = render(
      <WithMotion reducedMotion="never">
        <Pill.Dot tone="success" pulse />
      </WithMotion>
    );
    let dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.className).toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBe("true");

    rerender(
      <WithMotion reducedMotion="always">
        <Pill.Dot tone="success" pulse />
      </WithMotion>
    );
    dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.className).not.toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBeNull();
  });

  it("Should derive its size from the parent Pill context", () => {
    const { container } = render(
      <Pill size="md">
        <Pill.Dot />
        FILTER
      </Pill>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.getAttribute("data-size")).toBe("md");
    expect(dot?.className).toContain("size-2");
  });

  it("Should fall back to a small dot when nested inside a sm Pill", () => {
    const { container } = render(
      <Pill size="sm">
        <Pill.Dot />
        Online
      </Pill>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.getAttribute("data-size")).toBe("sm");
    expect(dot?.className).toContain("size-1.5");
  });

  it("Should respect an explicit size prop over the parent context", () => {
    const { container } = render(
      <Pill size="md">
        <Pill.Dot size="sm" />
        Online
      </Pill>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.getAttribute("data-size")).toBe("sm");
    expect(dot?.className).toContain("size-1.5");
  });
});
