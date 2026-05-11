import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MetadataTile } from "../metadata-tile";

describe("MetadataTile", () => {
  it("Should render an Inter UC eyebrow label and a tabular-nums value", () => {
    const { container } = render(<MetadataTile label="Last run" value="3m ago" />);
    const label = container.querySelector<HTMLElement>('[data-slot="metadata-tile-label"]');
    expect(label?.className).toContain("eyebrow");
    expect(label?.className).toContain("text-(--muted)");
    const value = container.querySelector<HTMLElement>('[data-slot="metadata-tile-value"]');
    expect(value?.className).toContain("tabular-nums");
    expect(screen.getByText("3m ago")).toBeInTheDocument();
  });

  it("Should render flat — no default border on the tile root", () => {
    const { container } = render(<MetadataTile label="Last run" value="3m ago" />);
    const root = container.querySelector<HTMLElement>('[data-slot="metadata-tile"]');
    expect(root?.className).not.toContain("border-(--line)");
    expect(root?.className).toContain("bg-(--canvas-soft)");
  });
});
