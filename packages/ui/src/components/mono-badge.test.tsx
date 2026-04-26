import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MonoBadge, type MonoBadgeTone } from "./mono-badge";

describe("MonoBadge", () => {
  it("Should render children uppercase in mono with the default outline tone", () => {
    const { container } = render(<MonoBadge>agent-42</MonoBadge>);
    const badge = container.querySelector<HTMLElement>('[data-slot="mono-badge"]');
    expect(badge).not.toBeNull();
    expect(badge?.textContent).toBe("agent-42");
    expect(badge?.className).toContain("font-mono");
    expect(badge?.className).toContain("uppercase");
    expect(badge?.className).toContain("rounded-[var(--radius-mono-badge)]");
    expect(badge?.className).toContain("border-[color:var(--color-divider)]");
    expect(badge?.getAttribute("data-tone")).toBe("default");
  });

  it("Should respect uppercase={false} and keep the provided casing", () => {
    const { container } = render(<MonoBadge uppercase={false}>Agent</MonoBadge>);
    const badge = container.querySelector<HTMLElement>('[data-slot="mono-badge"]');
    expect(badge?.className).not.toContain("uppercase");
  });

  it.each<{ tone: MonoBadgeTone; background: string; text: string }>([
    {
      tone: "accent",
      background: "bg-[color:var(--color-accent-tint)]",
      text: "text-[color:var(--color-accent)]",
    },
    {
      tone: "success",
      background: "bg-[color:var(--color-success-tint)]",
      text: "text-[color:var(--color-success)]",
    },
    {
      tone: "warning",
      background: "bg-[color:var(--color-warning-tint)]",
      text: "text-[color:var(--color-warning)]",
    },
    {
      tone: "danger",
      background: "bg-[color:var(--color-danger-tint)]",
      text: "text-[color:var(--color-danger)]",
    },
    {
      tone: "info",
      background: "bg-[color:var(--color-info-tint)]",
      text: "text-[color:var(--color-info)]",
    },
    {
      tone: "neutral",
      background: "bg-[color:var(--color-neutral-tint)]",
      text: "text-[color:var(--color-text-label)]",
    },
    {
      tone: "solid-accent",
      background: "bg-[color:var(--color-accent)]",
      text: "text-[color:var(--color-accent-ink)]",
    },
  ])("Should apply the $tone tint tokens", ({ tone, background, text }) => {
    const { container } = render(<MonoBadge tone={tone}>token</MonoBadge>);
    const badge = container.querySelector<HTMLElement>('[data-slot="mono-badge"]');
    expect(badge?.getAttribute("data-tone")).toBe(tone);
    expect(badge?.className).toContain(background);
    expect(badge?.className).toContain(text);
  });

  it("Should preserve the requested slot while keeping the component tone marker stable", () => {
    const { container } = render(
      <MonoBadge tone="accent" data-slot="override-slot" data-tone="override-tone">
        token
      </MonoBadge>
    );
    const badge = container.querySelector<HTMLElement>('[data-slot="override-slot"]');
    expect(badge).not.toBeNull();
    expect(badge?.getAttribute("data-tone")).toBe("accent");
  });
});
