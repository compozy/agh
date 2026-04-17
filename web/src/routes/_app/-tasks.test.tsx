import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let childMatches: Array<{ id: string }> = [];

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="tasks-outlet" />,
  useChildMatches: () => childMatches,
}));

import { Route } from "./tasks";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TasksRoute = (Route as any).component as () => ReactNode;

describe("TasksRoute", () => {
  beforeEach(() => {
    childMatches = [];
  });

  it("renders the shared tasks shell with the Tasks title", () => {
    render(<TasksRoute />);
    expect(screen.getByRole("heading", { level: 1, name: "Tasks" })).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-icon")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-body")).toBeInTheDocument();
  });

  it("renders the landing placeholder when no child route is active", () => {
    render(<TasksRoute />);
    expect(screen.getByTestId("tasks-shell-placeholder")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-outlet")).not.toBeInTheDocument();
  });

  it("renders the outlet instead of the placeholder when a child route is active", () => {
    childMatches = [{ id: "/_app/tasks/$id" }];
    render(<TasksRoute />);
    expect(screen.getByTestId("tasks-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-shell-placeholder")).not.toBeInTheDocument();
  });
});
