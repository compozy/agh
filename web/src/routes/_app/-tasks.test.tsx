import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let childMatches: Array<{ id: string }> = [];

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
  Outlet: () => <div data-testid="tasks-outlet" />,
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  useChildMatches: () => childMatches,
}));

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn().mockResolvedValue([]),
  getTask: vi.fn().mockResolvedValue({}),
  listTaskRuns: vi.fn().mockResolvedValue([]),
  getTaskTimeline: vi.fn().mockResolvedValue([]),
  getTaskTree: vi.fn().mockResolvedValue({}),
  getTaskRun: vi.fn().mockResolvedValue({}),
  getTaskDashboard: vi.fn().mockResolvedValue({}),
  getTaskInbox: vi.fn().mockResolvedValue({}),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  publishTask: vi.fn(),
  cancelTask: vi.fn(),
  approveTask: vi.fn(),
  rejectTask: vi.fn(),
  createChildTask: vi.fn(),
  addTaskDependency: vi.fn(),
  removeTaskDependency: vi.fn(),
  enqueueTaskRun: vi.fn(),
  attachTaskRunSession: vi.fn(),
  cancelTaskRun: vi.fn(),
  claimTaskRun: vi.fn(),
  startTaskRun: vi.fn(),
  completeTaskRun: vi.fn(),
  failTaskRun: vi.fn(),
  markTaskRead: vi.fn(),
  archiveTask: vi.fn(),
  dismissTask: vi.fn(),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_alpha", name: "Alpha" },
    activeWorkspaceId: "ws_alpha",
  }),
}));

import { Route } from "./tasks";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TasksRoute = (Route as any).component as () => ReactNode;

function renderTasksRoute() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TasksRoute />
    </QueryClientProvider>
  );
}

describe("TasksRoute", () => {
  beforeEach(() => {
    childMatches = [];
  });

  it("renders the shared tasks shell with the Tasks title", () => {
    renderTasksRoute();
    expect(screen.getByRole("heading", { level: 1, name: "Tasks" })).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-icon")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-body")).toBeInTheDocument();
  });

  it("renders mode pills, the create button, and the empty state when no tasks exist", async () => {
    renderTasksRoute();
    expect(screen.getByTestId("tasks-mode-pills")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-list")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-kanban")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("tasks-empty-state")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-empty-template-one_shot")).toBeInTheDocument();
  });

  it("renders the outlet inside the shell when a child route is active", () => {
    childMatches = [{ id: "/_app/tasks/$id" }];
    renderTasksRoute();
    expect(screen.getByTestId("tasks-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-empty-state")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-mode-pills")).not.toBeInTheDocument();
  });
});
