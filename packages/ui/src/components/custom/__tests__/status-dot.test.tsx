import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusDot } from "../status-dot";

describe("StatusDot", () => {
  it.each([
    ["warning", "solid"],
    ["danger", "solid"],
    ["warning", "ring"],
    ["accent", "solid"],
    ["faint", "ring"],
  ] as const)("Should render the %s/%s tone", (tone, variant) => {
    const { container } = render(<StatusDot tone={tone} variant={variant} />);
    const node = container.querySelector<HTMLElement>('[data-slot="status-dot"]');
    expect(node?.dataset.tone).toBe(tone);
    expect(node?.dataset.variant).toBe(variant);
    expect(node?.className).toContain(`text-(--${tone})`);
    if (variant === "solid") {
      expect(node?.className).toContain("bg-current");
      expect(node?.className).not.toContain("border ");
    } else {
      expect(node?.className).toContain("border");
      expect(node?.className).toContain("border-current");
      expect(node?.className).not.toContain("bg-current");
    }
  });

  it("Should size to 6 px by default and 5 px at sm", () => {
    const { container: def } = render(<StatusDot tone="accent" />);
    expect(def.querySelector('[data-slot="status-dot"]')?.className).toContain("size-1.5");
    const { container: sm } = render(<StatusDot tone="accent" size="sm" />);
    expect(sm.querySelector('[data-slot="status-dot"]')?.className).toContain("size-[5px]");
  });

  it("Should expose an accessible label only when provided", () => {
    const { container: bare } = render(<StatusDot tone="accent" />);
    expect(bare.querySelector('[data-slot="status-dot"]')?.getAttribute("aria-hidden")).toBe(
      "true"
    );

    const { container: labeled } = render(<StatusDot tone="accent" label="Mentions" />);
    const node = labeled.querySelector('[data-slot="status-dot"]');
    expect(node?.getAttribute("aria-hidden")).toBeNull();
    expect(node?.getAttribute("aria-label")).toBe("Mentions");
    expect(node?.getAttribute("role")).toBe("img");
  });
});
