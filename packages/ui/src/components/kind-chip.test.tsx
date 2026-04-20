import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KindChip } from "./kind-chip";

describe("KindChip", () => {
  it("Should render the original kind text with lowercase mono accent styling", () => {
    const { container } = render(<KindChip kind="Greet" />);
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.textContent).toBe("Greet");
    expect(chip?.className).toContain("font-mono");
    expect(chip?.className).toContain("lowercase");
    expect(chip?.className).toContain("rounded-[var(--radius-chip)]");
    expect(chip?.className).toContain("bg-[color:var(--color-accent-tint)]");
    expect(chip?.className).toContain("text-[color:var(--color-accent)]");
    expect(chip?.getAttribute("data-kind")).toBe("Greet");
  });

  it("Should forward the provided className alongside the defaults", () => {
    const { container } = render(<KindChip kind="whois" className="custom-class" />);
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip?.className).toContain("custom-class");
    expect(chip?.className).toContain("bg-[color:var(--color-accent-tint)]");
  });

  it("Should preserve internal data markers when conflicting data attributes are passed", () => {
    const { container } = render(
      <KindChip kind="whois" data-slot="override-slot" data-kind="override-kind" />
    );
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.getAttribute("data-kind")).toBe("whois");
  });
});
