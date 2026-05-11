import { render, screen } from "@testing-library/react";
import { Sparkles } from "lucide-react";
import { describe, expect, it } from "vitest";

import { FormSection } from "../form-section";

describe("FormSection", () => {
  it("Should render --canvas-soft + --radius-lg with no border at comfortable density", () => {
    const { container } = render(
      <FormSection title="Scope">
        <span>body</span>
      </FormSection>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="form-section"]');
    expect(root?.dataset.size).toBe("comfortable");
    expect(root?.className).toContain("bg-(--canvas-soft)");
    expect(root?.className).toContain("rounded-(--radius-lg)");
    expect(root?.className).toContain("px-5");
    expect(root?.className).toContain("py-[18px]");
    expect(root?.className).not.toContain("border-(--line)");
  });

  it("Should switch to compact padding per ADR-015 §7", () => {
    const { container } = render(
      <FormSection title="Scope" size="compact">
        <span>body</span>
      </FormSection>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="form-section"]');
    expect(root?.dataset.size).toBe("compact");
    expect(root?.className).toContain("px-4");
    expect(root?.className).toContain("py-[14px]");
    expect(root?.className).not.toContain("py-[18px]");
  });

  it("Should render the 13/510 section-head title via token classes", () => {
    const { container } = render(<FormSection title="Scope">body</FormSection>);
    const title = container.querySelector<HTMLElement>('[data-slot="form-section-title"]');
    expect(title?.className).toContain("text-[length:var(--text-section-head)]");
    expect(title?.className).toContain("tracking-(--tracking-section-head)");
    expect(title?.style.fontWeight).toBe("510");
  });

  it("Should render an optional icon + right eyebrow", () => {
    render(
      <FormSection title="Scope" icon={Sparkles} rightLabel="optional">
        body
      </FormSection>
    );
    expect(screen.getByText("optional").dataset.slot).toBe("form-section-right-label");
    expect(screen.getByText("optional").className).toContain("eyebrow");
  });
});
