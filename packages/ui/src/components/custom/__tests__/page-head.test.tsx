import { render, screen } from "@testing-library/react";
import { Repeat2 } from "lucide-react";
import { describe, expect, it } from "vitest";

import { PageHead } from "../page-head";

describe("PageHead", () => {
  it("Should render body summary chrome without creating a competing page H1", () => {
    render(
      <PageHead
        count={4}
        countTestId="head-count"
        data-testid="head"
        icon={Repeat2}
        meta={
          <>
            <span>Installed skills available to agents.</span>
            <PageHead.MetaDot data-testid="meta-dot" />
            <span>launch-hq</span>
          </>
        }
        title="Skills"
      />
    );

    const head = screen.getByTestId("head");
    expect(head).toHaveAttribute("data-slot", "page-head");
    expect(head).toHaveAttribute("data-variant", "index");
    expect(screen.queryByRole("heading", { level: 1 })).not.toBeInTheDocument();
    expect(screen.getByText("Skills")).toHaveAttribute("data-slot", "page-head-title");
    expect(screen.getByTestId("head-count")).toHaveTextContent("4");
    expect(screen.getByTestId("meta-dot")).toHaveAttribute("aria-hidden", "true");
    expect(head.querySelector("[data-slot='page-head-icon']")).toHaveAttribute(
      "aria-hidden",
      "true"
    );
  });

  it("Should render detail variant with pills row and no count chip", () => {
    render(
      <PageHead
        data-testid="head"
        pills={<span data-testid="status-pill">ready</span>}
        title="software-delivery"
        variant="detail"
      />
    );

    expect(screen.getByTestId("head")).toHaveAttribute("data-variant", "detail");
    expect(screen.getByTestId("status-pill")).toBeInTheDocument();
    expect(screen.getByTestId("head").querySelector("[data-slot='page-head-count']")).toBeNull();
  });

  it("Should host trailing hero controls in the actions zone", () => {
    render(
      <PageHead
        actions={<button data-testid="runtime-selector" type="button" />}
        data-testid="head"
        title="release-captain"
        variant="detail"
      />
    );

    const zone = screen.getByTestId("head").querySelector("[data-slot='page-head-actions']");
    expect(zone).not.toBeNull();
    expect(zone).toContainElement(screen.getByTestId("runtime-selector"));
  });

  it("Should render compact variant with mono pre-title above the body title", () => {
    render(
      <PageHead
        data-testid="head"
        pretitle="operator-style.md"
        title="Operator Style"
        variant="compact"
      />
    );

    expect(screen.getByTestId("head")).toHaveAttribute("data-variant", "compact");
    expect(screen.getByText("operator-style.md")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { level: 1 })).not.toBeInTheDocument();
    expect(screen.getByText("Operator Style")).toHaveAttribute("data-slot", "page-head-title");
  });

  it("Should prefer a custom leading mark over the icon well", () => {
    render(
      <PageHead
        data-testid="head"
        icon={Repeat2}
        leading={<span data-testid="custom-mark" />}
        title="claude-main"
        variant="detail"
      />
    );

    expect(screen.getByTestId("custom-mark")).toBeInTheDocument();
    expect(screen.getByTestId("head").querySelector("[data-slot='page-head-icon']")).toBeNull();
  });
});
