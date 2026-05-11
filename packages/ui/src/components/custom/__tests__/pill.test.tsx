import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MotionConfig } from "motion/react";
import type { MouseEvent, ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Pill, type PillSize, type PillTone } from "../pill";

interface WithMotionProps {
  reducedMotion: "always" | "never";
  children: ReactNode;
}

interface ToneExpectation {
  tone: PillTone;
  bg: string;
  text: string;
}

interface DotExpectation {
  tone: PillTone;
  color: string;
}

interface SizeHeightExpectation {
  size: PillSize;
  height: string;
}

const TONES: PillTone[] = ["neutral", "accent", "success", "warning", "danger", "info"];
const SIZES: PillSize[] = ["xs", "sm", "md"];

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
    expect(pill.className).toContain("bg-(--neutral-tint)");
    expect(pill.className).toContain("text-(--muted)");
  });

  it.each<ToneExpectation>([
    { tone: "accent", bg: "bg-(--accent-tint)", text: "text-(--accent)" },
    { tone: "success", bg: "bg-(--success-tint)", text: "text-(--success)" },
    { tone: "warning", bg: "bg-(--warning-tint)", text: "text-(--warning)" },
    { tone: "danger", bg: "bg-(--danger-tint)", text: "text-(--danger)" },
    { tone: "info", bg: "bg-(--info-tint)", text: "text-(--info)" },
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
    expect(pill.className).toContain("bg-(--accent)");
    expect(pill.className).toContain("text-(--accent-ink)");
  });

  it("Should adopt mono typography without forcing uppercase", () => {
    render(<Pill mono>token</Pill>);
    const pill = screen.getByText("token");
    expect(pill).toHaveAttribute("data-mono", "true");
    expect(pill.className).toContain("font-mono");
    expect(pill.className).not.toMatch(/(^| )uppercase( |$)/);
  });

  it.each<SizeHeightExpectation>([
    { size: "xs", height: "h-[17px]" },
    { size: "sm", height: "h-[19px]" },
    { size: "md", height: "h-[22px]" },
  ])("Should render the $size pill at the normalized height", ({ size, height }) => {
    render(<Pill size={size}>label-{size}</Pill>);
    const pill = screen.getByText(content => content.startsWith("label-"));
    expect(pill).toHaveAttribute("data-size", size);
    expect(pill.className).toContain(height);
  });

  it.each<PillSize>(SIZES)("Should render every size with the flat 4px chip radius", size => {
    render(
      <Pill size={size} data-testid={`pill-${size}`}>
        x
      </Pill>
    );
    const pill = screen.getByTestId(`pill-${size}`);
    expect(pill.className).toContain("rounded-(--radius-xs)");
    expect(pill.className).not.toContain("rounded-(--radius-chip)");
    expect(pill.className).not.toContain("rounded-(--radius-mono-badge)");
    expect(pill.className).not.toContain("rounded-(--radius-pill)");
  });

  it.each<PillSize>(SIZES)(
    "Should emit font-semibold tracking-[0] on mono pills regardless of size (%s)",
    size => {
      render(
        <Pill mono size={size} data-testid={`pill-mono-${size}`}>
          v1.2.3
        </Pill>
      );
      const pill = screen.getByTestId(`pill-mono-${size}`);
      expect(pill.className).toContain("font-semibold");
      expect(pill.className).toContain("tracking-[0]");
      expect(pill.className).not.toMatch(/font-(medium|normal|light)\b/);
    }
  );

  it("Should NOT emit any numeric font-weight inline style on mono pills", () => {
    render(<Pill mono>token</Pill>);
    const pill = screen.getByText("token");
    expect(pill.style.fontWeight).toBe("");
  });

  it.each<PillTone>(TONES)(
    "Should NOT emit a `border-` color class on the inactive %s pill",
    tone => {
      render(
        <Pill tone={tone} data-testid={`pill-${tone}`}>
          chip
        </Pill>
      );
      const pill = screen.getByTestId(`pill-${tone}`);
      expect(pill.className).not.toMatch(/(^| )border-\(/);
      expect(pill.className).not.toMatch(/(^| )border-[a-z]/);
    }
  );

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
    expect(pill.className).toContain("bg-(--elevated)");
    expect(pill.className).toContain("text-(--fg-strong)");
  });

  it("Should keep the borderless neutral tint when active=false", () => {
    render(
      <Pill mono active={false} render={<button type="button" />}>
        FILTER
      </Pill>
    );
    const pill = screen.getByRole("button", { name: /filter/i });
    expect(pill).toHaveAttribute("data-active", "false");
    expect(pill.className).toContain("bg-(--neutral-tint)");
    expect(pill.className).toContain("text-(--muted)");
    expect(pill.className).not.toMatch(/(^| )border-\(/);
  });

  it("Should forward className alongside the variant defaults", () => {
    render(<Pill className="custom">label</Pill>);
    const pill = screen.getByText("label");
    expect(pill.className).toContain("custom");
    expect(pill.className).toContain("bg-(--neutral-tint)");
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

describe("Pill class snapshot matrix", () => {
  for (const tone of TONES) {
    for (const size of SIZES) {
      it(`Should lock the sans ${tone} × ${size} class block`, () => {
        const testId = `pill-${tone}-${size}-sans`;
        render(
          <Pill tone={tone} size={size} data-testid={testId}>
            x
          </Pill>
        );
        const pill = screen.getByTestId(testId);
        expect(pill.className).toMatchSnapshot();
      });

      it(`Should lock the mono ${tone} × ${size} class block`, () => {
        const testId = `pill-${tone}-${size}-mono`;
        render(
          <Pill tone={tone} size={size} mono data-testid={testId}>
            x
          </Pill>
        );
        const pill = screen.getByTestId(testId);
        expect(pill.className).toMatchSnapshot();
      });
    }
  }
});

describe("Pill.Dot", () => {
  it("Should render a 8px md dot keyed to neutral by default", () => {
    const { container } = render(<Pill.Dot />);
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot).not.toBeNull();
    expect(dot?.getAttribute("data-tone")).toBe("neutral");
    expect(dot?.getAttribute("data-size")).toBe("md");
    expect(dot?.className).toContain("size-2");
    expect(dot?.style.backgroundColor).toBe("var(--subtle)");
    expect(dot?.getAttribute("aria-hidden")).toBe("true");
  });

  it.each<DotExpectation>([
    { tone: "accent", color: "var(--accent)" },
    { tone: "success", color: "var(--success)" },
    { tone: "warning", color: "var(--warning)" },
    { tone: "danger", color: "var(--danger)" },
    { tone: "info", color: "var(--info)" },
  ])("Should map tone %s to the semantic color token", ({ tone, color }) => {
    const { container } = render(<Pill.Dot tone={tone} />);
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.style.backgroundColor).toBe(color);
  });

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

  it("Should inherit pulse from the parent Pill context", () => {
    const { container } = render(
      <WithMotion reducedMotion="never">
        <Pill pulse>
          <Pill.Dot />
          Online
        </Pill>
      </WithMotion>
    );
    const dot = container.querySelector<HTMLElement>('[data-slot="pill-dot"]');
    expect(dot?.className).toContain("animate-pulse");
    expect(dot?.getAttribute("data-pulse")).toBe("true");
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
