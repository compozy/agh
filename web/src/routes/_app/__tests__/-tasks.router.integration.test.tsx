import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  useNavigate,
} from "@tanstack/react-router";
import { describe, expect, it } from "vitest";

import { UIProvider } from "@agh/ui";

import { TasksDetailHeader, TasksListSurface } from "@/systems/tasks";
import type { TaskDetailView, TaskListItem } from "@/systems/tasks";

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

  const taskCreateRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "new",
    component: () => <div data-testid="tasks-create-route">new task</div>,
  });

  const taskEditRoute = createRoute({
    getParentRoute: () => taskDetailRoute,
    path: "edit",
    component: () => {
      const params = taskDetailRoute.useParams();
      return (
        <div data-testid="tasks-edit-route">
          <span data-testid="tasks-edit-id">{params.id}</span>
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
      tasksRoute.addChildren([
        taskCreateRoute,
        taskDetailRoute.addChildren([taskEditRoute, taskRunDetailRoute]),
      ]),
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

  it("resolves /tasks/new inside the shared tasks shell", async () => {
    const router = buildTestRouter("/tasks/new");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-create-route")).toBeInTheDocument();
  });

  it("resolves /tasks/$id/edit inside the shared tasks shell", async () => {
    const router = buildTestRouter("/tasks/task_abc/edit");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-edit-route")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-edit-id")).toHaveTextContent("task_abc");
  });

  it("keeps the tasks shell mounted while navigating between base, create, detail, edit, and run-detail routes", async () => {
    const router = buildTestRouter("/tasks");
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(screen.getByTestId("tasks-shell")).toBeInTheDocument());
    const baseShell = screen.getByTestId("tasks-shell");

    await router.history.push("/tasks/new");
    await waitFor(() => expect(screen.getByTestId("tasks-create-route")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);

    await router.navigate({ to: "/tasks/$id", params: { id: "task_abc" } });
    await waitFor(() => expect(screen.getByTestId("tasks-detail")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);

    await router.history.push("/tasks/task_abc/edit");
    await waitFor(() => expect(screen.getByTestId("tasks-edit-route")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);

    await router.navigate({
      to: "/tasks/$id/runs/$runId",
      params: { id: "task_abc", runId: "run_001" },
    });
    await waitFor(() => expect(screen.getByTestId("tasks-run-detail")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-shell")).toBe(baseShell);
  });
});

// ----- List → detail selection integration -----

const FIXTURE_TASKS: TaskListItem[] = [
  {
    id: "task_001",
    identifier: "TASK-1",
    title: "Summarize review feedback",
    status: "in_progress",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
  } as TaskListItem,
  {
    id: "task_002",
    identifier: "TASK-2",
    title: "Generate API client",
    status: "ready",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
  } as TaskListItem,
];

function buildSelectionRouter(initialUrl: string) {
  const rootRoute = createRootRoute({
    component: () => (
      <UIProvider reducedMotion="always">
        <Outlet />
      </UIProvider>
    ),
  });

  const tasksRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "tasks",
    component: TasksListRouteComponent,
  });

  const detailRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "$id",
    component: TaskDetailRouteComponent,
  });

  const routeTree = rootRoute.addChildren([tasksRoute.addChildren([detailRoute])]);

  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [initialUrl] }),
  });

  return { router, detailRoute };

  function TasksListRouteComponent() {
    const navigate = useNavigate({ from: "/tasks" });

    return (
      <div data-testid="tasks-shell">
        <TasksListSurface
          onOwnerChange={() => {}}
          onPriorityChange={() => {}}
          onScopeChange={() => {}}
          onSelectTask={taskId => {
            void navigate({ params: { id: taskId }, to: "/tasks/$id" });
          }}
          onSortChange={() => {}}
          onStatusChange={() => {}}
          ownerFilter={null}
          ownerOptions={[]}
          priorityFilter={null}
          scopeFilter="all"
          sortBy="recent"
          statusFilter={null}
          tasks={FIXTURE_TASKS}
          totalCount={FIXTURE_TASKS.length}
        />
        <Outlet />
      </div>
    );
  }

  function TaskDetailRouteComponent() {
    const params = detailRoute.useParams() as unknown as { id: string };
    const match = FIXTURE_TASKS.find(task => task.id === params.id);
    if (!match) return null;
    const detail = {
      task: match,
      summary: match as unknown as TaskDetailView["summary"],
    } as TaskDetailView;
    return <TasksDetailHeader detail={detail} />;
  }
}

describe("tasks router selection (integration)", () => {
  it("navigates to /tasks/$id when a list row is clicked and renders the matching detail header", async () => {
    const { router } = buildSelectionRouter("/tasks");
    render(<RouterProvider router={router} />);

    await waitFor(() => expect(screen.getByTestId("tasks-list-surface")).toBeInTheDocument());
    expect(router.state.location.pathname).toBe("/tasks");

    // Click the second task row.
    fireEvent.click(screen.getByTestId("task-card-task_002"));

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/tasks/task_002");
    });

    const title = await screen.findByTestId("tasks-detail-title");
    expect(title).toHaveTextContent("Generate API client");
  });
});
