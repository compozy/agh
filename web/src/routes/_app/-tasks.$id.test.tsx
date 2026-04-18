import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc" };
let childMatches: Array<{ id: string }> = [];

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
  Outlet: () => <div data-testid="tasks-detail-outlet" />,
  useChildMatches: () => childMatches,
}));

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn().mockResolvedValue([]),
  getTask: vi.fn(),
  listTaskRuns: vi.fn().mockResolvedValue([]),
  getTaskTimeline: vi.fn().mockResolvedValue([]),
  getTaskTree: vi.fn().mockResolvedValue({ root: { depth: 0, task: { id: "task_abc" } } }),
  getTaskRun: vi.fn(),
  getTaskDashboard: vi.fn(),
  getTaskInbox: vi.fn(),
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

import { getTask } from "@/systems/tasks/adapters/tasks-api";

import { Route } from "./tasks.$id";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskDetailRoute = (Route as any).component as () => ReactNode;

function renderRoute() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TaskDetailRoute />
    </QueryClientProvider>
  );
}

const detailFixture = {
  task: {
    id: "task_abc",
    identifier: "TASK-42",
    title: "Summarize review feedback",
    status: "in_progress",
    scope: "workspace",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    description: "Pull CodeRabbit review",
  },
  summary: {
    id: "task_abc",
    title: "Summarize review feedback",
    status: "in_progress",
    scope: "workspace",
    created_by: { kind: "human", ref: "pedro@" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    origin: { kind: "cli", ref: "op" },
    child_count: 0,
    dependency_count: 0,
  },
  children: [],
  dependency_references: [],
  runs: [],
};

describe("TaskDetailRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc" };
    childMatches = [];
    vi.mocked(getTask).mockResolvedValue(detailFixture as never);
  });

  it("renders the task header once detail loads", async () => {
    renderRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-detail-content")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-detail-title")).toHaveTextContent("Summarize review feedback");
  });

  it("renders the detail tabs for the resolved task", async () => {
    renderRoute();
    await waitFor(() =>
      expect(screen.getByTestId("tasks-detail-tab-overview")).toBeInTheDocument()
    );
    expect(screen.getByTestId("tasks-detail-tab-runs")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-tab-timeline")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-tab-children")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-tab-dependencies")).toBeInTheDocument();
  });

  it("renders the outlet for nested run-detail routes when a child matches", () => {
    childMatches = [{ id: "/_app/tasks/$id/runs/$runId" }];
    renderRoute();
    expect(screen.getByTestId("tasks-detail-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-content")).not.toBeInTheDocument();
  });

  it("renders a not-found state when the task cannot be fetched", async () => {
    vi.mocked(getTask).mockRejectedValue(new Error("Task not found: task_abc"));
    renderRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-detail-not-found")).toBeInTheDocument());
  });
});
