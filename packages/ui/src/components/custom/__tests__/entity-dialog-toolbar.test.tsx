import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EntityDialogToolbar } from "../entity-dialog-toolbar";

describe("EntityDialogToolbar", () => {
  it("Should rule only its bottom edge so it does not seam against the header", () => {
    const { container } = render(
      <EntityDialogToolbar trailing={<button type="button">ws</button>} />
    );
    const root = container.querySelector<HTMLElement>('[data-slot="entity-dialog-toolbar"]');

    // The ruled dialog header already draws a bottom border; a `border-y` here
    // stacked against it and printed a 2px seam.
    expect(root).toHaveClass("border-b");
    expect(root).not.toHaveClass("border-y");
  });

  it("Should render a trailing control with its label", () => {
    render(
      <EntityDialogToolbar
        trailing={<button type="button">launch-hq</button>}
        trailingLabel="Scope"
      />
    );

    expect(screen.getByText("Scope")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "launch-hq" })).toBeInTheDocument();
  });

  it("Should start the trailing control at the gutter when there is no disclosure tier", () => {
    const { container } = render(
      <EntityDialogToolbar trailing={<button type="button">launch-hq</button>} />
    );

    // The automation editors carry scope with no Simple/Advanced pills, so the
    // scope is the bar's only content and must not float against the far edge.
    expect(container.querySelector(".flex-1")).toBeNull();
    expect(screen.getByRole("button", { name: "launch-hq" })).toBeInTheDocument();
  });

  it("Should push the trailing control right once the leading edge is occupied", () => {
    const { container } = render(
      <EntityDialogToolbar
        leading={<button type="button">Simple</button>}
        trailing={<button type="button">launch-hq</button>}
      />
    );

    expect(container.querySelector(".flex-1")).not.toBeNull();
  });
});
