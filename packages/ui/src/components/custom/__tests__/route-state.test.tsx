import { render, screen } from "@testing-library/react";
import { AlertTriangleIcon } from "lucide-react";
import { describe, expect, it } from "vitest";

import { RouteState } from "../route-state";

describe("RouteState", () => {
  it("Should render mode='loading' with role=status and a visible label", () => {
    render(<RouteState mode="loading" title="Loading sessions" />);
    const node = screen.getByRole("status");
    expect(node).toHaveAttribute("aria-live", "polite");
    expect(screen.getByText("Loading sessions")).toBeInTheDocument();
  });

  it("Should render mode='empty' with a title, message, icon, and action", () => {
    render(
      <RouteState
        mode="empty"
        title="No tasks"
        message="Create one to get started."
        icon={AlertTriangleIcon}
        action={<button type="button">Create</button>}
      />
    );
    expect(screen.getByText("No tasks")).toBeInTheDocument();
    expect(screen.getByText("Create one to get started.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create/i })).toBeInTheDocument();
  });

  it("Should render mode='error' with a recovery action and a cause node", () => {
    render(
      <RouteState
        mode="error"
        title="Something went wrong"
        message="The daemon dropped the connection."
        cause={<span data-testid="error-cause">request id 1234</span>}
        action={<button type="button">Retry</button>}
      />
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByTestId("error-cause")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });
});
