import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kbd, KbdGroup } from "../kbd";

describe("Kbd", () => {
  it("Should default to font-mono and not contain font-sans", () => {
    const { container } = render(<Kbd>K</Kbd>);
    const node = container.querySelector('[data-slot="kbd"]');
    expect(node?.className).toContain("font-mono");
    expect(node?.className).not.toContain("font-sans");
  });

  it("Should render KbdGroup with inline-flex", () => {
    const { container } = render(
      <KbdGroup>
        <Kbd>⌘</Kbd>
        <Kbd>K</Kbd>
      </KbdGroup>
    );
    const group = container.querySelector('[data-slot="kbd-group"]');
    expect(group?.className).toContain("inline-flex");
  });
});
