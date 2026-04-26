import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WireCard, WireCardBody, WireCardFoot, WireCardHead } from "./wire-card";

describe("WireCard", () => {
  it("Should render a bordered surface with a 520px max width by default", () => {
    const { container } = render(<WireCard>body</WireCard>);
    const card = container.querySelector<HTMLElement>('[data-slot="wire-card"]');
    expect(card).not.toBeNull();
    expect(card?.className).toContain("max-w-[520px]");
    expect(card?.className).toContain("border-[color:var(--color-divider)]");
    expect(card?.className).toContain("bg-[color:var(--color-surface)]");
  });

  it("Should switch to inline strip layout when inline is set", () => {
    const { container } = render(<WireCard inline>body</WireCard>);
    const card = container.querySelector<HTMLElement>('[data-slot="wire-card"]');
    expect(card?.getAttribute("data-inline")).toBe("true");
    expect(card?.className).toContain("inline-flex");
    expect(card?.className).not.toContain("max-w-[520px]");
  });

  it("Should render head/body/foot subcomponents with the canonical chrome", () => {
    const { container } = render(
      <WireCard>
        <WireCardHead>head</WireCardHead>
        <WireCardBody>body</WireCardBody>
        <WireCardFoot>foot</WireCardFoot>
      </WireCard>
    );
    const head = container.querySelector<HTMLElement>('[data-slot="wire-card-head"]');
    const body = container.querySelector<HTMLElement>('[data-slot="wire-card-body"]');
    const foot = container.querySelector<HTMLElement>('[data-slot="wire-card-foot"]');
    expect(head).not.toBeNull();
    expect(body).not.toBeNull();
    expect(foot).not.toBeNull();
    expect(head?.className).toContain("bg-[color:var(--color-canvas-deep)]");
    expect(foot?.className).toContain("bg-[color:var(--color-canvas-deep)]");
  });
});
