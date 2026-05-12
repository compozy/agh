import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewRow } from "../review-row";

describe("ReviewRow", () => {
  it("Should render leading, title, description, and actions", () => {
    render(
      <ReviewRow
        leading={<span data-testid="avatar">A</span>}
        title="Pedro"
        description="Approved the migration"
        actions={<button type="button">Comment</button>}
      />
    );
    expect(screen.getByTestId("avatar")).toBeInTheDocument();
    expect(screen.getByText("Pedro")).toBeInTheDocument();
    expect(screen.getByText("Approved the migration")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /comment/i })).toBeInTheDocument();
  });
});
