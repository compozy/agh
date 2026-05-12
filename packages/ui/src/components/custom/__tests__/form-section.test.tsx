import { render, screen } from "@testing-library/react";
import { Sparkles } from "lucide-react";
import { describe, expect, it } from "vitest";

import { FormSection } from "../form-section";

describe("FormSection", () => {
  it("Should default to comfortable density", () => {
    const { container } = render(
      <FormSection title="Scope">
        <span>body</span>
      </FormSection>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="form-section"]');
    expect(root?.dataset.size).toBe("comfortable");
  });

  it("Should switch to compact density via prop", () => {
    const { container } = render(
      <FormSection title="Scope" size="compact">
        <span>body</span>
      </FormSection>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="form-section"]');
    expect(root?.dataset.size).toBe("compact");
  });

  it("Should render an optional icon + right eyebrow", () => {
    render(
      <FormSection title="Scope" icon={Sparkles} rightLabel="optional">
        body
      </FormSection>
    );
    expect(screen.getByText("optional").dataset.slot).toBe("form-section-right-label");
  });
});
