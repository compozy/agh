import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EntityDialogBody } from "../entity-dialog-body";

describe("EntityDialogBody", () => {
  it("Should own the scroll for the default variant and ignore side content", () => {
    render(
      <EntityDialogBody side={<p>Session defaults</p>}>
        <p>Location</p>
      </EntityDialogBody>
    );

    const body = screen.getByText("Location").closest('[data-slot="entity-dialog-body"]');
    expect(body).toHaveAttribute("data-variant", "default");
    expect(body?.className).toContain("overflow-y-auto");
    expect(screen.queryByText("Session defaults")).not.toBeInTheDocument();
  });

  it("Should give the split variant two independent scroll owners", () => {
    const { container } = render(
      <EntityDialogBody side={<p>Session defaults</p>} variant="split">
        <p>Location</p>
      </EntityDialogBody>
    );

    const body = container.querySelector('[data-slot="entity-dialog-body"]');
    expect(body).toHaveAttribute("data-variant", "split");

    const main = container.querySelector<HTMLElement>('[data-slot="entity-dialog-body-main"]');
    const side = container.querySelector<HTMLElement>('[data-slot="entity-dialog-body-side"]');
    expect(main?.className).toContain("overflow-y-auto");
    expect(side?.className).toContain("overflow-y-auto");
    expect(main).toContainElement(screen.getByText("Location"));
    expect(side).toContainElement(screen.getByText("Session defaults"));
  });
});
