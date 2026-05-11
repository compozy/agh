import { render } from "@testing-library/react";
import { Bot } from "lucide-react";
import { describe, expect, it } from "vitest";

import { OwnerAvatar } from "../owner-avatar";

describe("OwnerAvatar", () => {
  it("Should resolve bg/fg to var(--avatar-agent-N-{bg,fg}) for agent owners", () => {
    const { container } = render(
      <OwnerAvatar ownerKind="agent" ownerId="planner-prime" name="Planner Prime" />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(root?.dataset.ownerKind).toBe("agent");
    expect(root?.style.backgroundColor).toMatch(/^var\(--avatar-agent-[0-3]-bg\)$/);
    expect(root?.style.color).toMatch(/^var\(--avatar-agent-[0-3]-fg\)$/);
  });

  it("Should resolve to var(--avatar-human-N-{bg,fg}) for human owners", () => {
    const { container } = render(<OwnerAvatar ownerKind="human" ownerId="pedro" name="Pedro" />);
    const root = container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(root?.style.backgroundColor).toMatch(/^var\(--avatar-human-[0-2]-bg\)$/);
    expect(root?.style.color).toMatch(/^var\(--avatar-human-[0-2]-fg\)$/);
  });

  it("Should resolve to var(--avatar-system-{bg,fg}) for system owners", () => {
    const { container } = render(<OwnerAvatar ownerKind="system" ownerId="daemon" name="Daemon" />);
    const root = container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(root?.style.backgroundColor).toBe("var(--avatar-system-bg)");
    expect(root?.style.color).toBe("var(--avatar-system-fg)");
  });

  it("Should emit aria-label with the role prefix", () => {
    const agent = render(
      <OwnerAvatar ownerKind="agent" ownerId="planner" name="Planner Prime" />
    ).container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(agent?.getAttribute("aria-label")).toBe("Agent Planner Prime");

    const human = render(
      <OwnerAvatar ownerKind="human" ownerId="pedro" name="Pedro Nauck" />
    ).container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(human?.getAttribute("aria-label")).toBe("Human Pedro Nauck");

    const system = render(
      <OwnerAvatar ownerKind="system" ownerId="daemon" name="Daemon" />
    ).container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(system?.getAttribute("aria-label")).toBe("System Daemon");
  });

  it.each([
    ["sm", 20, "9px"],
    ["default", 24, "10px"],
    ["lg", 32, "12px"],
  ] as const)("Should render at the right pixel size for %s", (size, px, textPx) => {
    const { container } = render(<OwnerAvatar ownerKind="agent" ownerId="x" size={size} />);
    const root = container.querySelector<HTMLElement>('[data-slot="owner-avatar"]');
    expect(root?.dataset.size).toBe(size);
    expect(root?.style.width).toBe(`${px}px`);
    expect(root?.style.height).toBe(`${px}px`);
    expect(root?.className).toContain(`text-[${textPx}]`);
  });

  it("Should derive a 2-character monogram from the display name", () => {
    const { container } = render(
      <OwnerAvatar ownerKind="agent" ownerId="planner" name="Planner Prime" />
    );
    const monogram = container.querySelector<HTMLElement>('[data-slot="owner-avatar-monogram"]');
    expect(monogram?.textContent).toBe("PP");
  });

  it("Should fall back to the ownerId when no name is provided", () => {
    const { container } = render(<OwnerAvatar ownerKind="agent" ownerId="planner-prime" />);
    const monogram = container.querySelector<HTMLElement>('[data-slot="owner-avatar-monogram"]');
    expect(monogram?.textContent).toBe("PP");
  });

  it("Should render the glyph slot in place of the monogram when supplied", () => {
    const { container } = render(
      <OwnerAvatar
        ownerKind="system"
        ownerId="daemon"
        name="Daemon"
        glyph={<Bot data-testid="glyph" />}
      />
    );
    expect(container.querySelector('[data-slot="owner-avatar-monogram"]')).toBeNull();
    expect(container.querySelector('[data-slot="owner-avatar-glyph"]')).not.toBeNull();
    expect(container.querySelector('[data-testid="glyph"]')).not.toBeNull();
  });
});
