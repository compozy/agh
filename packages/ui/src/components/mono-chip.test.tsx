import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MonoChip } from "./mono-chip";

describe("MonoChip", () => {
  it("Should render a neutral mono chip with elevated surface", () => {
    const { container } = render(<MonoChip>code</MonoChip>);
    const chip = container.querySelector<HTMLElement>('[data-slot="mono-chip"]');
    expect(chip).not.toBeNull();
    expect(chip?.textContent).toBe("code");
    expect(chip?.className).toContain("font-mono");
    expect(chip?.className).toContain("bg-[color:var(--color-surface-elevated)]");
    expect(chip?.className).toContain("text-[color:var(--color-text-secondary)]");
  });

  it("Should forward className", () => {
    const { container } = render(<MonoChip className="custom-class">tag</MonoChip>);
    const chip = container.querySelector<HTMLElement>('[data-slot="mono-chip"]');
    expect(chip?.className).toContain("custom-class");
  });
});
