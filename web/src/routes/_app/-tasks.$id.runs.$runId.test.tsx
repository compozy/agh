import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc", runId: "run_001" };

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
}));

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: vi.fn().mockResolvedValue([]),
  getTask: vi.fn(),
  listTaskRuns: vi.fn().mockResolvedValue([]),
  getTaskTimeline: vi.fn().mockResolvedValue([]),
  getTaskTree: vi.fn().mockResolvedValue({}),
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

import { getTask, getTaskRun } from "@/systems/tasks/adapters/tasks-api";

import { Route } from "./tasks.$id.runs.$runId";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskRunDetailRoute = (Route as any).component as () => ReactNode;

function renderRoute() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TaskRunDetailRoute />
    </QueryClientProvider>
  );
}

const runFixture = {
  run: {
    id: "run_001",
    task_id: "task_abc",
    attempt: 2,
    status: "running",
    queued_at: "2026-04-11T14:30:00Z",
    started_at: "2026-04-11T14:37:45Z",
    origin: { kind: "cli", ref: "op" },
    session_id: "sess_jf8d21",
    claimed_by: { kind: "agent_session", ref: "Coder" },
  },
  task: {
    id: "task_abc",
    identifier: "TASK-42",
    status: "ready",
    scope: "workspace",
    title: "Summarize review feedback",
  },
  summary: {
    last_activity_at: "2026-04-11T14:40:45Z",
    tool_call_count: 4,
    input_tokens: 14281,
    output_tokens: 3046,
    total_tokens: 17327,
  },
  session: {
    session_id: "sess_jf8d21",
    created_at: "2026-04-11T14:30:00Z",
    updated_at: "2026-04-11T14:40:45Z",
    agent_name: "Coder",
  },
};

const taskFixture = {
  task: {
    id: "task_abc",
    identifier: "TASK-42",
    status: "ready",
    scope: "workspace",
    title: "Summarize review feedback",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
  },
  summary: {
    id: "task_abc",
    title: "Summarize review feedback",
    status: "ready",
    scope: "workspace",
    created_by: { kind: "human", ref: "pedro@" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    origin: { kind: "cli", ref: "op" },
  },
};

describe("TaskRunDetailRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc", runId: "run_001" };
    vi.mocked(getTaskRun).mockResolvedValue(runFixture as never);
    vi.mocked(getTask).mockResolvedValue(taskFixture as never);
  });

  it("renders the run header, identity, progress, and activity", async () => {
    renderRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-run-detail-content")).toBeInTheDocument());
    expect(screen.getByTestId("task-run-detail-title")).toHaveTextContent("Run run_001");
    expect(screen.getByTestId("task-run-detail-identity-run")).toHaveTextContent("run_001");
    expect(screen.getByTestId("task-run-detail-progress-input-tokens")).toHaveTextContent("14,281");
    expect(screen.getByTestId("task-run-detail-activity")).toBeInTheDocument();
  });

  it("renders a not-found state when the run cannot be fetched", async () => {
    vi.mocked(getTaskRun).mockRejectedValue(new Error("Task run not found: run_001"));
    renderRoute();
    await waitFor(() =>
      expect(screen.getByTestId("tasks-run-detail-not-found")).toBeInTheDocument()
    );
  });
});
