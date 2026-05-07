import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc" };
let childMatches: Array<{ id: string }> = [];
const navigateMock = vi.fn();

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
  useNavigate: () => navigateMock,
}));

const treeWithDescendantFixture = {
  root: {
    depth: 0,
    task: {
      id: "task_abc",
      identifier: "TASK-38",
      title: "Triage",
      status: "in_progress",
      scope: "workspace",
      owner: { kind: "agent_session", ref: "Researcher" },
    },
    active_run: {
      id: "run_root",
      attempt: 1,
      max_attempts: 3,
      queued_at: "2026-04-17T10:00:00Z",
      status: "running",
      task_id: "task_abc",
      session_id: "sess_root",
    },
    child_count: 1,
    last_activity_at: "2026-04-17T10:01:00Z",
  },
  descendants: [
    {
      depth: 1,
      parent_task_id: "task_abc",
      task: {
        id: "task_child",
        identifier: "TASK-39",
        status: "in_progress",
        scope: "workspace",
        title: "Reproduce",
        owner: { kind: "agent_session", ref: "Coder" },
      },
      active_run: {
        id: "run_child",
        attempt: 1,
        max_attempts: 2,
        queued_at: "2026-04-17T10:00:10Z",
        status: "running",
        task_id: "task_child",
        session_id: "sess_child",
      },
      child_count: 0,
      last_activity_at: "2026-04-17T10:01:00Z",
    },
  ],
};

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

import { getTask, getTaskTree } from "@/systems/tasks/adapters/tasks-api";

import { Route } from "../tasks.$id";

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
    navigateMock.mockReset();
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

  it("renders the multi-agent live panel when the agents tab is activated", async () => {
    vi.mocked(getTaskTree).mockResolvedValue(treeWithDescendantFixture as never);

    renderRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-detail-tab-agents")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("tasks-detail-tab-agents"));

    await waitFor(() => expect(screen.getByTestId("tasks-multi-agent-panel")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-multi-agent-agent-task_abc")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-multi-agent-agent-task_child")).toBeInTheDocument();
    // The "N agents live" pill and interleaved-timeline banner have been
    // removed — summary text + tab badge carry the signal instead.
    expect(screen.getByTestId("tasks-multi-agent-summary")).toHaveTextContent(/running/);
    expect(screen.queryByTestId("tasks-multi-agent-live-count")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-multi-agent-timeline-live")).not.toBeInTheDocument();
  });

  it("falls back to the disconnected state when the tree read fails", async () => {
    vi.mocked(getTaskTree).mockRejectedValue(new Error("Tree stream disconnected"));

    renderRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-detail-tab-agents")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("tasks-detail-tab-agents"));

    await waitFor(() =>
      expect(screen.getByTestId("tasks-multi-agent-disconnected")).toBeInTheDocument()
    );
    expect(screen.getByTestId("tasks-multi-agent-disconnected")).toHaveTextContent(
      "Tree stream disconnected"
    );
  });
});
