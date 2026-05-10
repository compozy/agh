import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FieldRow } from "../field-row";

describe("FieldRow", () => {
  it("Should render the label, control, and description in a stacked layout by default", () => {
    render(
      <FieldRow
        label="Email"
        description="We will never share it."
        htmlFor="email"
        control={<input id="email" />}
      />
    );
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByText("We will never share it.")).toBeInTheDocument();
  });

  it("Should switch the layout to two columns when requested", () => {
    const { container } = render(
      <FieldRow label="Name" layout="two-column" htmlFor="name" control={<input id="name" />} />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="field-row"]');
    expect(root?.dataset.layout).toBe("two-column");
  });
});
