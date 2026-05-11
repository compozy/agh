import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Card } from "../card";

describe("Card", () => {
  it("Should render a card root with the stable data-slot", () => {
    const { container } = render(<Card>body</Card>);
    const root = container.querySelector<HTMLElement>('[data-slot="card"]');
    expect(root).not.toBeNull();
  });

  it("Should reflect data-active-rail when activeRail is set", () => {
    const { container } = render(<Card activeRail>body</Card>);
    const root = container.querySelector<HTMLElement>('[data-slot="card"]');
    expect(root?.getAttribute("data-active-rail")).toBe("true");
  });
});
