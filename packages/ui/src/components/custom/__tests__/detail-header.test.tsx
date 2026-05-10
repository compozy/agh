import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DetailHeader } from "../detail-header";

describe("DetailHeader", () => {
  it("Should render crumbs, title, pills, meta, and actions", () => {
    render(
      <DetailHeader
        crumbs={<span>Tasks / detail</span>}
        title="Task #42"
        pills={<span data-testid="pill">Active</span>}
        meta={<span>id-42 · 3m ago</span>}
        actions={<button type="button">Run</button>}
      />
    );
    expect(screen.getByText("Tasks / detail")).toBeInTheDocument();
    expect(screen.getByText("Task #42")).toBeInTheDocument();
    expect(screen.getByTestId("pill")).toBeInTheDocument();
    expect(screen.getByText("id-42 · 3m ago")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run/i })).toBeInTheDocument();
  });
});
