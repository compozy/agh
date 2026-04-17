import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc", runId: "run_001" };

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
}));

import { Route } from "./tasks.$id.runs.$runId";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskRunDetailRoute = (Route as any).component as () => ReactNode;

describe("TaskRunDetailRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc", runId: "run_001" };
  });

  it("renders the run-detail placeholder with both task and run ids", () => {
    render(<TaskRunDetailRoute />);
    const placeholder = screen.getByTestId("tasks-run-detail-placeholder");
    expect(placeholder).toHaveTextContent("task_abc");
    expect(placeholder).toHaveTextContent("run_001");
  });

  it("updates the placeholder when the deep-link run id changes", () => {
    const { rerender } = render(<TaskRunDetailRoute />);
    expect(screen.getByTestId("tasks-run-detail-placeholder")).toHaveTextContent("run_001");
    routeParams = { id: "task_abc", runId: "run_002" };
    rerender(<TaskRunDetailRoute />);
    expect(screen.getByTestId("tasks-run-detail-placeholder")).toHaveTextContent("run_002");
  });
});
