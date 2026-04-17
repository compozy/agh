import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc" };
let childMatches: Array<{ id: string }> = [];

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
  Outlet: () => <div data-testid="tasks-detail-outlet" />,
  useChildMatches: () => childMatches,
}));

import { Route } from "./tasks.$id";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskDetailRoute = (Route as any).component as () => ReactNode;

describe("TaskDetailRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc" };
    childMatches = [];
  });

  it("renders the detail placeholder with the resolved task id", () => {
    render(<TaskDetailRoute />);
    const placeholder = screen.getByTestId("tasks-detail-placeholder");
    expect(placeholder).toHaveTextContent("task_abc");
  });

  it("does not render the run-detail outlet when no child route is active", () => {
    render(<TaskDetailRoute />);
    expect(screen.queryByTestId("tasks-detail-outlet")).not.toBeInTheDocument();
  });

  it("renders the outlet for nested run-detail routes when a child matches", () => {
    childMatches = [{ id: "/_app/tasks/$id/runs/$runId" }];
    render(<TaskDetailRoute />);
    expect(screen.getByTestId("tasks-detail-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-placeholder")).not.toBeInTheDocument();
  });

  it("re-renders the placeholder when navigating to a different task id", () => {
    const { rerender } = render(<TaskDetailRoute />);
    expect(screen.getByTestId("tasks-detail-placeholder")).toHaveTextContent("task_abc");
    routeParams = { id: "task_xyz" };
    rerender(<TaskDetailRoute />);
    expect(screen.getByTestId("tasks-detail-placeholder")).toHaveTextContent("task_xyz");
  });
});
