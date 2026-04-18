import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let childMatches: Array<{ id: string; params?: { id?: string } }> = [];
const navigateMock = vi.fn();

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
  useNavigate: () => navigateMock,
}));

const listTasksMock = vi.fn();
const getTaskDashboardMock = vi.fn();
const getTaskInboxMock = vi.fn();
const approveTaskMock = vi.fn();
const rejectTaskMock = vi.fn();
const archiveTaskMock = vi.fn();
const markTaskReadMock = vi.fn();
const dismissTaskMock = vi.fn();
const enqueueTaskRunMock = vi.fn();

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: (...args: unknown[]) => listTasksMock(...args),
  getTask: vi.fn().mockResolvedValue({}),
  listTaskRuns: vi.fn().mockResolvedValue([]),
  getTaskTimeline: vi.fn().mockResolvedValue([]),
  getTaskTree: vi.fn().mockResolvedValue({}),
  getTaskRun: vi.fn().mockResolvedValue({}),
  getTaskDashboard: (...args: unknown[]) => getTaskDashboardMock(...args),
  getTaskInbox: (...args: unknown[]) => getTaskInboxMock(...args),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  publishTask: vi.fn(),
  cancelTask: vi.fn(),
  approveTask: (...args: unknown[]) => approveTaskMock(...args),
  rejectTask: (...args: unknown[]) => rejectTaskMock(...args),
  createChildTask: vi.fn(),
  addTaskDependency: vi.fn(),
  removeTaskDependency: vi.fn(),
  enqueueTaskRun: (...args: unknown[]) => enqueueTaskRunMock(...args),
  attachTaskRunSession: vi.fn(),
  cancelTaskRun: vi.fn(),
  claimTaskRun: vi.fn(),
  startTaskRun: vi.fn(),
  completeTaskRun: vi.fn(),
  failTaskRun: vi.fn(),
  markTaskRead: (...args: unknown[]) => markTaskReadMock(...args),
  archiveTask: (...args: unknown[]) => archiveTaskMock(...args),
  dismissTask: (...args: unknown[]) => dismissTaskMock(...args),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_alpha", name: "Alpha" },
    activeWorkspaceId: "ws_alpha",
  }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { Route } from "./tasks";
import {
  buildDashboardFixture,
  buildInboxFixture,
  buildInboxItemFixture,
} from "@/systems/tasks/components/test-fixtures";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TasksRoute = (Route as any).component as () => ReactNode;

function renderTasksRoute() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <TasksRoute />
    </QueryClientProvider>
  );
}

describe("TasksRoute", () => {
  beforeEach(() => {
    childMatches = [];
    navigateMock.mockReset();
    listTasksMock.mockReset();
    listTasksMock.mockResolvedValue([]);
    getTaskDashboardMock.mockReset();
    getTaskDashboardMock.mockResolvedValue(buildDashboardFixture());
    getTaskInboxMock.mockReset();
    getTaskInboxMock.mockResolvedValue(
      buildInboxFixture({
        total: 1,
        unread_total: 1,
        groups: [
          {
            lane: "approvals",
            count: 1,
            unread_count: 1,
            items: [
              buildInboxItemFixture({
                lane: "approvals",
                approval_policy: "manual",
                approval_state: "pending",
                task: {
                  id: "task_apr",
                  identifier: "TASK-33",
                  scope: "workspace",
                  status: "pending",
                  title: "Rotate keys",
                },
                triage: {
                  actor: { kind: "human", ref: "op" },
                  archived: false,
                  dismissed: false,
                  read: false,
                  task_id: "task_apr",
                  updated_at: "2026-04-17T10:00:00Z",
                },
              }),
            ],
          },
        ],
      })
    );
    approveTaskMock.mockReset();
    approveTaskMock.mockResolvedValue({ id: "task_apr" });
    rejectTaskMock.mockReset();
    archiveTaskMock.mockReset();
    markTaskReadMock.mockReset();
    dismissTaskMock.mockReset();
    enqueueTaskRunMock.mockReset();
  });

  it("renders the shared tasks shell with the Tasks title", () => {
    renderTasksRoute();
    expect(screen.getByTestId("tasks-shell-title")).toHaveTextContent("Tasks");
    expect(screen.getByTestId("tasks-shell-icon")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-shell-body")).toBeInTheDocument();
  });

  it("renders mode pills, the create button, and the empty state when no tasks exist", async () => {
    renderTasksRoute();
    expect(screen.getByTestId("tasks-mode-pills")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-list")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-kanban")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-inbox")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("tasks-empty-state")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-empty-template-one_shot")).toBeInTheDocument();
  });

  it("renders the outlet inside the shell when a child route is active", () => {
    childMatches = [{ id: "/_app/tasks/$id", params: { id: "task_abc" } }];
    renderTasksRoute();
    expect(screen.getByTestId("tasks-outlet")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-pills")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-list-panel")).toBeInTheDocument();
  });

  it("switches to the dashboard view and renders the cards + queue/health sections", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-dashboard"));

    await waitFor(() => {
      expect(getTaskDashboardMock).toHaveBeenCalled();
    });

    expect(await screen.findByTestId("tasks-dashboard-view")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-cards")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-queue-health")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-list-panel")).not.toBeInTheDocument();
  });

  it("switches to the inbox view, renders the approvals lane, and triggers approve action", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-inbox"));

    await waitFor(() => expect(getTaskInboxMock).toHaveBeenCalled());

    expect(await screen.findByTestId("tasks-inbox-view")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument();
    const inboxTab = screen.getByTestId("tasks-mode-inbox");
    expect(inboxTab.querySelector('[data-slot="pills-badge"]')).toHaveTextContent("1");
    expect(screen.getByTestId("tasks-inbox-group-approvals")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-inbox-item-approve-task_apr"));
    await waitFor(() => {
      expect(approveTaskMock).toHaveBeenCalledWith("task_apr");
    });
  });

  it("changes lane filter and re-queries the inbox endpoint", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-inbox"));
    await waitFor(() => expect(getTaskInboxMock).toHaveBeenCalled());

    fireEvent.click(screen.getByTestId("tasks-inbox-lane-approvals"));

    await waitFor(() => {
      expect(getTaskInboxMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ lane: "approvals" }),
        expect.any(AbortSignal)
      );
    });
  });

  it("navigates to the route-based editor when the create action is clicked", async () => {
    listTasksMock.mockResolvedValue([
      {
        id: "task_abc",
        title: "Create API contract",
        status: "draft",
        scope: "workspace",
        updated_at: "2026-04-17T10:00:00Z",
        created_at: "2026-04-17T09:00:00Z",
        created_by: { kind: "human", ref: "pedro@" },
        origin: { kind: "web", ref: "agh-web" },
      },
    ]);

    renderTasksRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("tasks-open-create"));

    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({ search: expect.any(Function), to: "/tasks/new" })
    );
  });
});
