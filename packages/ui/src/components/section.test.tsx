import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Section } from "./section";

describe("Section", () => {
  it("Should render the label + optional right slot + children", () => {
    const { container } = render(
      <Section label="Members" right={<span data-testid="section-right">filter</span>}>
        <p>Body content</p>
      </Section>
    );

    const label = container.querySelector('[data-slot="section-label"]');
    expect(label?.textContent).toBe("Members");
    expect(screen.getByTestId("section-right")).toBeInTheDocument();
    expect(screen.getByText("Body content")).toBeInTheDocument();
  });

  it("Should omit the header when neither label nor right are provided", () => {
    const { container } = render(
      <Section>
        <p>body only</p>
      </Section>
    );
    expect(container.querySelector('[data-slot="section-head"]')).toBeNull();
    expect(container.querySelector('[data-slot="section-body"]')?.textContent).toBe("body only");
  });
});
