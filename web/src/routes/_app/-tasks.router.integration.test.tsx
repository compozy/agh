import { render, screen, waitFor } from "@testing-library/react";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from "@tanstack/react-router";
import { describe, expect, it } from "vitest";

function buildTestRouter(initialUrl: string) {
  const rootRoute = createRootRoute({
    component: () => <Outlet />,
  });

  const appRoute = createRoute({
    getParentRoute: () => rootRoute,
    id: "_app",
    component: () => (
      <div data-testid="app-shell">
        <Outlet />
      </div>
    ),
  });

  const tasksRoute = createRoute({
    getParentRoute: () => appRoute,
    path: "tasks",
    component: () => (
      <div data-testid="tasks-shell">
        <Outlet />
      </div>
    ),
  });

  const taskDetailRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "$id",
    component: () => {
      const params = taskDetailRoute.useParams();
      return (
        <div data-testid="tasks-detail">
          <span data-testid="tasks-detail-id">{params.id}</span>
          <Outlet />
        </div>
      );
    },
  });

  const taskRunDetailRoute = createRoute({
    getParentRoute: () => taskDetailRoute,
    path: "runs/$runId",
    component: () => {
      const params = taskRunDetailRoute.useParams();
      return (
        <div data-testid="tasks-run-detail">
          <span data-testid="tasks-run-detail-task-id">{params.id}</span>
          <span data-testid="tasks-run-detail-run-id">{params.runId}</span>
        </div>
      );
    },
  });

  const routeTree = rootRoute.addChildren([
    appRoute.addChildren([
      tasksRoute.addChildren([taskDetailRoute.addChildren([taskRunDetailRoute])]),
    ]),
  ]);

  return createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [initialUrl] }),
  });
}

describe("tasks router registration (integration)", () => {
  it("resolves the base /tasks route inside the tasks shell", async () => {
    const router = buildTestRouter("/tasks");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    expect(screen.queryByTestId("tasks-detail")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-run-detail")).not.toBeInTheDocument();
  });

  it("resolves /tasks/$id with the shell still mounted", async () => {
    const router = buildTestRouter("/tasks/task_abc");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-detail")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-id")).toHaveTextContent("task_abc");
    expect(screen.queryByTestId("tasks-run-detail")).not.toBeInTheDocument();
  });

  it("resolves /tasks/$id/runs/$runId as a child of the detail route", async () => {
    const router = buildTestRouter("/tasks/task_abc/runs/run_001");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-detail")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-run-detail")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-run-detail-task-id")).toHaveTextContent("task_abc");
    expect(screen.getByTestId("tasks-run-detail-run-id")).toHaveTextContent("run_001");
  });

  it("keeps the tasks shell mounted while navigating between base, detail, and run-detail routes", async () => {
    const router = buildTestRouter("/tasks");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    const baseShell = screen.getByTestId("tasks-shell");

    await router.navigate({ to: "/tasks/$id", params: { id: "task_abc" } });
    await waitFor(() => expect(screen.getByTestId("tasks-detail")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);

    await router.navigate({
      to: "/tasks/$id/runs/$runId",
      params: { id: "task_abc", runId: "run_001" },
    });
    await waitFor(() => expect(screen.getByTestId("tasks-run-detail")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);
  });
});
