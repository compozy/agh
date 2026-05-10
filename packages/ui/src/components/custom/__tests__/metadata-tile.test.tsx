import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MetadataTile } from "../metadata-tile";

describe("MetadataTile", () => {
  it("Should render an upper-case mono label and a tabular-nums value", () => {
    const { container } = render(<MetadataTile label="Last run" value="3m ago" />);
    const label = container.querySelector<HTMLElement>('[data-slot="metadata-tile-label"]');
    expect(label?.dataset.case).toBe("upper");
    const value = container.querySelector<HTMLElement>('[data-slot="metadata-tile-value"]');
    expect(value?.className).toContain("tabular-nums");
    expect(screen.getByText("3m ago")).toBeInTheDocument();
  });

  it("Should support sentence-case labels", () => {
    const { container } = render(
      <MetadataTile label="Last run" value="3m ago" labelCase="sentence" />
    );
    const label = container.querySelector<HTMLElement>('[data-slot="metadata-tile-label"]');
    expect(label?.dataset.case).toBe("sentence");
    expect(label?.className).not.toContain("uppercase");
  });
});
