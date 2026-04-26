import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KindChip, KIND_DOT_COLORS } from "./kind-chip";

describe("KindChip", () => {
  it("Should render the kind label uppercase with the wire-dot chrome", () => {
    const { container } = render(<KindChip kind="greet" />);
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.textContent).toBe("greet");
    expect(chip?.className).toContain("font-mono");
    expect(chip?.className).toContain("uppercase");
    expect(chip?.className).toContain("border-[color:var(--color-divider)]");
    expect(chip?.className).toContain("bg-transparent");
    expect(chip?.className).toContain("text-[color:var(--color-text-tertiary)]");
    expect(chip?.getAttribute("data-kind")).toBe("greet");
  });

  it("Should render a colored 7px dot for known protocol kinds", () => {
    const { container } = render(<KindChip kind="receipt" />);
    const dot = container.querySelector<HTMLElement>('[data-slot="kind-chip-dot"]');
    expect(dot).not.toBeNull();
    expect(dot).toHaveStyle({ background: KIND_DOT_COLORS.receipt });
  });

  it("Should omit the dot for unknown kinds (platforms, event ids)", () => {
    const { container } = render(<KindChip kind="github" />);
    expect(container.querySelector('[data-slot="kind-chip-dot"]')).toBeNull();
  });

  it("Should display the explicit label when provided", () => {
    const { container } = render(<KindChip kind="greet" label="presence" />);
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip?.textContent).toBe("presence");
  });

  it("Should forward className alongside the defaults", () => {
    const { container } = render(<KindChip kind="whois" className="custom-class" />);
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip?.className).toContain("custom-class");
    expect(chip?.className).toContain("border-[color:var(--color-divider)]");
  });

  it("Should preserve internal data markers when conflicting attributes are passed", () => {
    const { container } = render(
      <KindChip kind="whois" data-slot="override-slot" data-kind="override-kind" />
    );
    const chip = container.querySelector<HTMLElement>('[data-slot="kind-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.getAttribute("data-kind")).toBe("whois");
  });
});
