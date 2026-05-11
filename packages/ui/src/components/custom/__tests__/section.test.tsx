import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Section } from "../section";

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

  it("Should render note and divided section chrome when requested", () => {
    const { container } = render(
      <Section label="Runtime" note="Read-only daemon state" divided>
        <p>Body content</p>
      </Section>
    );

    expect(container.querySelector('[data-slot="section-note"]')).toHaveTextContent(
      "Read-only daemon state"
    );
    const root = container.querySelector('[data-slot="section"]');
    expect(root?.className).toContain("border-t");
    expect(root?.className).toContain("first:border-t-0");
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

  it.each([false, null])("Should omit the header when label is %p", label => {
    const { container } = render(
      <Section label={label}>
        <p>body only</p>
      </Section>
    );
    expect(container.querySelector('[data-slot="section-head"]')).toBeNull();
  });

  it("Should still render numeric zero labels", () => {
    const { container } = render(
      <Section label={0}>
        <p>body only</p>
      </Section>
    );
    expect(container.querySelector('[data-slot="section-label"]')?.textContent).toBe("0");
  });

  it("Should render the head without bottom border by default (ADR-003 §2)", () => {
    const { container } = render(<Section label="Members">body</Section>);
    const head = container.querySelector('[data-slot="section-head"]');
    expect(head?.className).not.toContain("border-b");
    expect(head?.getAttribute("data-bordered")).toBeNull();
  });

  it("Should opt into a `--line` hairline when `bordered` is true", () => {
    const { container } = render(
      <Section label="Members" bordered>
        body
      </Section>
    );
    const head = container.querySelector('[data-slot="section-head"]');
    expect(head?.className).toContain("border-b");
    expect(head?.className).toContain("border-(--line)");
    expect(head?.getAttribute("data-bordered")).toBe("true");
  });

  it("Should render the H2 at --text-section-head (13 px) — never the legacy 22 px tuple", () => {
    const { container } = render(<Section label="Members">body</Section>);
    const heading = container.querySelector<HTMLElement>('[data-slot="section-label"]');
    expect(heading?.className).toContain("text-(length:--text-section-head)");
    expect(heading?.className).toContain("tracking-(--tracking-section-head)");
    expect(heading?.className).not.toContain("text-[22px]");
    expect(heading?.className).not.toContain("tracking-[-0.026em]");
  });
});
