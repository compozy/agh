import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ListingPage } from "../listing-page";

describe("ListingPage", () => {
  it("Should compose the listing shell, heading metadata, and body content", () => {
    render(
      <ListingPage data-testid="listing" banner={<p>Cached results</p>}>
        <ListingPage.Head
          count={3}
          countTestId="listing-count"
          meta={
            <>
              <span>Workspace loops</span>
              <ListingPage.MetaDot data-testid="meta-dot" />
              <span>launch-hq</span>
            </>
          }
          title="Loops"
        />
        <p>Listing body</p>
      </ListingPage>
    );

    expect(screen.getByTestId("listing")).toHaveAttribute("data-slot", "listing-page");
    expect(screen.getByRole("heading", { level: 1, name: "Loops" })).toBeInTheDocument();
    expect(screen.getByTestId("listing-count")).toHaveTextContent("3");
    expect(screen.getByTestId("meta-dot")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByText("Cached results")).toBeInTheDocument();
    expect(screen.getByText("Listing body")).toBeInTheDocument();
  });
});
