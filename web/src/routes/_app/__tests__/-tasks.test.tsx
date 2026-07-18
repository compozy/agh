// Suite: Tasks route
// Invariant: The route distinguishes an unavailable task catalog from an empty catalog.
// Boundary IN: Tasks route query wiring and top-level surface selection.
// Boundary OUT: List presentation details owned by systems/tasks component suites.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { MouseEvent, ReactNode } from "react";
import { useEffect, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let childMatches: Array<{ id: string; params?: { id?: string } }> = [];
const navigateMock = vi.fn();

// Task mode navigation is a real `<Link search={{ mode }} to="/tasks">` (route
// chrome contract), not a local-state pill toggle. The mock keeps a listener
// registry so clicking a mode link updates the search TanStack Router would
// own, and every `useSearch()` reader re-renders off the same source.
const routerState = vi.hoisted(() => ({
  searchParams: {} as Record<string, unknown>,
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => Record<string, unknown>)
    | undefined,
}));

function getValidatedSearch() {
  return routerState.validateSearch
    ? routerState.validateSearch(routerState.searchParams)
    : routerState.searchParams;
}

function setSearchParams(next: Record<string, unknown>) {
  routerState.searchParams = next;
  const validated = getValidatedSearch();
  for (const listener of routerState.searchListeners) listener(validated);
}

function useMockedSearch() {
  const [search, setSearch] = useState(getValidatedSearch());
  useEffect(() => {
    routerState.searchListeners.add(setSearch);
    return () => {
      routerState.searchListeners.delete(setSearch);
    };
  }, []);
  return search;
}

const daemonStatusMockState = vi.hoisted(
  (): {
    data: { user_home_dir: string } | undefined;
    error: Error | null;
    isLoading: boolean;
    isPending: boolean;
  } => ({
    data: { user_home_dir: "/Users/operator" },
    error: null,
    isLoading: false,
    isPending: false,
  })
);

vi.mock("@/systems/status", () => ({
  useDaemonStatus: () => daemonStatusMockState,
}));

function normalizeSearch(value: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    if (entry !== undefined) out[key] = entry;
  }
  return out;
}

function searchMatchesExactly(
  current: Record<string, unknown>,
  next: Record<string, unknown>
): boolean {
  const currentNorm = normalizeSearch(current);
  const nextNorm = normalizeSearch(next);
  const nextKeys = Object.keys(nextNorm);
  return (
    Object.keys(currentNorm).length === nextKeys.length &&
    nextKeys.every(key => currentNorm[key] === nextNorm[key])
  );
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    activeOptions,
    children,
    search,
    ...rest
  }: {
    activeOptions?: { exact?: boolean; includeSearch?: boolean };
    children: ReactNode;
    search?: unknown;
  } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    const currentSearch = getValidatedSearch();
    const nextSearch =
      typeof search === "function"
        ? (search as (current: Record<string, unknown>) => Record<string, unknown>)(currentSearch)
        : ((search ?? {}) as Record<string, unknown>);
    const includeSearch = activeOptions?.includeSearch ?? true;
    const exact = activeOptions?.exact ?? false;
    const isActive =
      includeSearch && exact
        ? searchMatchesExactly(currentSearch, nextSearch)
        : includeSearch
          ? Object.keys(normalizeSearch(nextSearch)).every(
              key => normalizeSearch(currentSearch)[key] === normalizeSearch(nextSearch)[key]
            )
          : true;
    return (
      <a
        {...domRest}
        aria-current={isActive ? "page" : undefined}
        onClick={(event: MouseEvent<HTMLAnchorElement>) => {
          event.preventDefault();
          setSearchParams(nextSearch);
        }}
      >
        {children}
      </a>
    );
  },
  Outlet: () => <div data-testid="tasks-outlet" />,
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validateSearch = opts.validateSearch;
      return {
        component: opts.component,
        useSearch: () => useMockedSearch(),
      };
    },
  useChildMatches: () => childMatches,
  useNavigate: () => navigateMock,
  useSearch: () => useMockedSearch(),
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
const retryTaskRunMock = vi.fn();
const getSchedulerMock = vi.fn();
const getSchedulerBacklogMock = vi.fn();
const pauseSchedulerMock = vi.fn();
const resumeSchedulerMock = vi.fn();
const drainSchedulerMock = vi.fn();

vi.mock("@/systems/tasks/adapters/tasks-api", () => ({
  listTasks: (...args: unknown[]) => listTasksMock(...args),
  getTask: vi.fn().mockResolvedValue({}),
  listTaskRuns: vi.fn().mockResolvedValue([]),
  getTaskTimeline: vi.fn().mockResolvedValue([]),
  getTaskTree: vi.fn().mockResolvedValue({}),
  getTaskRun: vi.fn().mockResolvedValue({}),
  inspectTask: vi.fn().mockResolvedValue(null),
  inspectRun: vi.fn().mockResolvedValue(null),
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
  startTaskRun: vi.fn(),
  completeTaskRun: vi.fn(),
  failTaskRun: vi.fn(),
  forceFailTaskRun: vi.fn(),
  forceReleaseTaskRun: vi.fn(),
  retryTaskRun: (...args: unknown[]) => retryTaskRunMock(...args),
  markTaskRead: (...args: unknown[]) => markTaskReadMock(...args),
  archiveTask: (...args: unknown[]) => archiveTaskMock(...args),
  dismissTask: (...args: unknown[]) => dismissTaskMock(...args),
}));

vi.mock("@/systems/scheduler/adapters/scheduler-api", () => ({
  getScheduler: (...args: unknown[]) => getSchedulerMock(...args),
  getSchedulerBacklog: (...args: unknown[]) => getSchedulerBacklogMock(...args),
  pauseScheduler: (...args: unknown[]) => pauseSchedulerMock(...args),
  resumeScheduler: (...args: unknown[]) => resumeSchedulerMock(...args),
  drainScheduler: (...args: unknown[]) => drainSchedulerMock(...args),
}));

vi.mock("@/systems/workspace", async importOriginal => {
  const actual = await importOriginal<typeof import("@/systems/workspace")>();

  return {
    ...actual,
    useActiveWorkspace: () => ({
      activeWorkspace: { id: "ws_alpha", name: "Alpha" },
      activeWorkspaceId: "ws_alpha",
      error: null,
      hasHydrated: true,
      isLoading: false,
      isPending: false,
      workspaces: [
        {
          add_dirs: [],
          created_at: "2026-04-17T10:00:00Z",
          id: "ws_alpha",
          name: "Alpha",
          root_dir: "/workspace/alpha",
          updated_at: "2026-04-17T10:00:00Z",
        },
        {
          add_dirs: [],
          created_at: "2026-04-17T10:00:00Z",
          id: "ws_beta",
          name: "Beta",
          root_dir: "/workspace/beta",
          updated_at: "2026-04-17T10:00:00Z",
        },
      ],
    }),
  };
});

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { routeComponent } from "@/test/route-options";
import { Route } from "../tasks";

const TasksRoute = routeComponent(Route);
import {
  buildDashboardFixture,
  buildInboxFixture,
  buildInboxItemFixture,
} from "@/systems/tasks/components/test-fixtures";
import { buildTaskFixture } from "@/systems/tasks/mocks/fixtures";

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
    routerState.searchParams = {};
    daemonStatusMockState.data = { user_home_dir: "/Users/operator" };
    daemonStatusMockState.error = null;
    daemonStatusMockState.isLoading = false;
    daemonStatusMockState.isPending = false;
    navigateMock.mockReset();
    listTasksMock.mockReset();
    listTasksMock.mockResolvedValue({
      facets: { owners: [], statuses: [] },
      page: { has_more: false, limit: 50, total: 0 },
      tasks: [],
    });
    getTaskDashboardMock.mockReset();
    getTaskDashboardMock.mockResolvedValue(buildDashboardFixture());
    getTaskInboxMock.mockReset();
    getTaskInboxMock.mockResolvedValue(
      buildInboxFixture({
        page: { has_more: false, limit: 50, total: 1 },
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
    retryTaskRunMock.mockReset();
    retryTaskRunMock.mockResolvedValue({ id: "run_retry" });
    getSchedulerMock.mockReset();
    getSchedulerMock.mockResolvedValue({
      active_claim_count: 0,
      as_of: "2026-04-17T10:00:00Z",
      paused: false,
      paused_task_count: 0,
      queued_run_count: 1,
    });
    getSchedulerBacklogMock.mockReset();
    getSchedulerBacklogMock.mockResolvedValue({ runs: [], total: 0 });
    pauseSchedulerMock.mockReset();
    pauseSchedulerMock.mockResolvedValue({});
    resumeSchedulerMock.mockReset();
    resumeSchedulerMock.mockResolvedValue({});
    drainSchedulerMock.mockReset();
    drainSchedulerMock.mockResolvedValue({
      completed: true,
      completed_at: "2026-04-17T10:00:01Z",
      remaining_claims: 0,
      scheduler: {
        active_claim_count: 0,
        as_of: "2026-04-17T10:00:01Z",
        paused: true,
        paused_task_count: 0,
        queued_run_count: 1,
      },
      started_at: "2026-04-17T10:00:00Z",
    });
  });

  it("renders the shared tasks shell body container", () => {
    renderTasksRoute();
    expect(screen.getByTestId("tasks-shell")).toBeInTheDocument();
    // Full-width route shell (PageShell density="route") replaces the legacy
    // SplitPane body wrapper
    expect(screen.getByTestId("tasks-shell")).toHaveAttribute("data-density", "route");
  });

  it("renders mode pills, the create button, and the empty state when no tasks exist", async () => {
    renderTasksRoute();
    expect(screen.getByTestId("tasks-mode-nav")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-list")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-kanban")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-inbox")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-mode-list")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("tasks-mode-kanban")).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("tasks-empty-state")).toBeInTheDocument());
    expect(screen.getByTestId("tasks-empty-template-one_shot")).toBeInTheDocument();
  });

  it("Should mark only the selected tasks mode with aria-current", async () => {
    renderTasksRoute();

    expect(screen.getByTestId("tasks-mode-list")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("tasks-mode-kanban")).not.toHaveAttribute("aria-current");

    fireEvent.click(screen.getByTestId("tasks-mode-kanban"));

    await waitFor(() => {
      expect(screen.getByTestId("tasks-mode-kanban")).toHaveAttribute("aria-current", "page");
    });
    expect(screen.getByTestId("tasks-mode-list")).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("tasks-mode-dashboard")).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("tasks-mode-inbox")).not.toHaveAttribute("aria-current");
  });

  it("keeps the task count unknown until the catalog returns an authoritative total", async () => {
    let resolveCatalog: ((value: unknown) => void) | undefined;
    listTasksMock.mockImplementationOnce(() => new Promise(resolve => (resolveCatalog = resolve)));

    renderTasksRoute();

    expect(await screen.findByTestId("tasks-list-surface-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-list-page-count")).not.toBeInTheDocument();

    await act(async () => {
      resolveCatalog?.({
        facets: { owners: [], statuses: [] },
        page: { has_more: false, limit: 50, total: 0 },
        tasks: [],
      });
      await Promise.resolve();
    });

    // A resolved-empty catalog swaps in the definitive TasksEmptyState (no list
    // surface, no page-head count chip) rather than showing a "0" count.
    expect(await screen.findByTestId("tasks-empty-state")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-list-page-count")).not.toBeInTheDocument();
  });

  it("renders the catalog error instead of the empty state when the initial list fails", async () => {
    listTasksMock.mockRejectedValueOnce(new Error("task catalog unavailable"));

    renderTasksRoute();

    expect(await screen.findByTestId("tasks-list-surface-error")).toHaveTextContent(
      "task catalog unavailable"
    );
    expect(screen.queryByTestId("tasks-list-page-count")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-empty-state")).not.toBeInTheDocument();
  });

  it("renders daemon status failure without a task count or catalog request", async () => {
    daemonStatusMockState.data = undefined;
    daemonStatusMockState.error = new Error("daemon status unavailable");

    renderTasksRoute();

    expect(await screen.findByTestId("tasks-scope-error")).toHaveTextContent(
      "daemon status unavailable"
    );
    expect(screen.queryByTestId("tasks-list-page-count")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-empty-state")).not.toBeInTheDocument();
    expect(listTasksMock).not.toHaveBeenCalled();
    expect(getTaskInboxMock).not.toHaveBeenCalled();
    expect(getTaskDashboardMock).not.toHaveBeenCalled();
  });

  it("renders the outlet inside the shell when a child route is active", () => {
    childMatches = [{ id: "/_app/tasks/$id", params: { id: "task_abc" } }];
    renderTasksRoute();
    expect(screen.getByTestId("tasks-outlet")).toBeInTheDocument();
    // The parent cedes the topbar slot to the child route (route chrome T2:
    // detail = breadcrumb + detail actions); publishing the mode nav here
    // would steal the child's slot in the single-publisher store.
    expect(screen.queryByTestId("tasks-mode-nav")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-open-create")).not.toBeInTheDocument();
    // The detail child route takes over the full canvas; the list panel
    // is no longer rendered side-by-side with the detail (no SplitPane).
    expect(screen.queryByTestId("tasks-list-surface")).not.toBeInTheDocument();
  });

  it("shows the exact unread badge before the inbox surface is opened", async () => {
    renderTasksRoute();

    await waitFor(() => {
      const inboxTab = screen.getByTestId("tasks-mode-inbox");
      expect(inboxTab.querySelector('[data-slot="route-nav-count"]')).toHaveTextContent("1");
    });
    expect(getTaskInboxMock).toHaveBeenCalledWith(
      expect.objectContaining({ limit: 1, scope: "workspace", workspace: "ws_alpha" }),
      expect.any(AbortSignal)
    );
    expect(screen.queryByTestId("tasks-inbox-view")).not.toBeInTheDocument();
  });

  it("switches to the dashboard view and renders the cards + queue/health sections", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-dashboard"));

    await waitFor(() => {
      expect(getTaskDashboardMock).toHaveBeenCalled();
    });

    expect(await screen.findByTestId("tasks-dashboard-view")).toBeInTheDocument();
    expect(screen.getByTestId("scheduler-controls-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-cards")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-dashboard-queue-health")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-list-surface")).not.toBeInTheDocument();
  });

  it("switches to the inbox view, renders the approvals lane, and triggers approve action", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-inbox"));

    await waitFor(() => expect(getTaskInboxMock).toHaveBeenCalled());

    expect(await screen.findByTestId("tasks-inbox-view")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument();
    await waitFor(() => {
      const inboxTab = screen.getByTestId("tasks-mode-inbox");
      expect(inboxTab.querySelector('[data-slot="route-nav-count"]')).toHaveTextContent("1");
    });
    // Approval items now live under the `Needs review` UI group
    expect(screen.getByTestId("tasks-inbox-group-needs_review")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-inbox-item-approve-task_apr"));
    await waitFor(() => {
      expect(approveTaskMock).toHaveBeenCalledWith("task_apr");
    });
  });

  it("disables only the pending run retry and blocks duplicate submission", async () => {
    let resolveRetry: ((value: unknown) => void) | undefined;
    retryTaskRunMock.mockImplementationOnce(() => new Promise(resolve => (resolveRetry = resolve)));
    getTaskInboxMock.mockResolvedValue(
      buildInboxFixture({
        page: { has_more: false, limit: 50, total: 2 },
        unread_total: 2,
        groups: [
          {
            lane: "failed_runs",
            count: 2,
            unread_count: 2,
            items: [
              buildInboxItemFixture({
                lane: "failed_runs",
                task: { id: "task_a", scope: "workspace", status: "failed", title: "Task A" },
                run: {
                  attempt: 1,
                  id: "run_a",
                  max_attempts: 3,
                  queued_at: "2026-04-17T09:55:00Z",
                  status: "failed",
                  task_id: "task_a",
                },
              }),
              buildInboxItemFixture({
                lane: "failed_runs",
                task: { id: "task_b", scope: "workspace", status: "failed", title: "Task B" },
                run: {
                  attempt: 1,
                  id: "run_b",
                  max_attempts: 3,
                  queued_at: "2026-04-17T09:56:00Z",
                  status: "failed",
                  task_id: "task_b",
                },
              }),
            ],
          },
        ],
      })
    );

    renderTasksRoute();
    fireEvent.click(screen.getByTestId("tasks-mode-inbox"));
    const retryA = await screen.findByTestId("tasks-inbox-item-retry-task_a");
    const retryB = screen.getByTestId("tasks-inbox-item-retry-task_b");

    fireEvent.click(retryA);
    await waitFor(() => expect(retryA).toBeDisabled());
    expect(retryA).toHaveAttribute("aria-busy", "true");
    expect(retryB).toBeEnabled();
    fireEvent.click(retryA);
    expect(retryTaskRunMock).toHaveBeenCalledTimes(1);
    expect(retryTaskRunMock).toHaveBeenCalledWith("run_a", {});

    await act(async () => {
      resolveRetry?.({ id: "run_retry" });
      await Promise.resolve();
    });
    await waitFor(() => expect(retryA).toBeEnabled());
  });

  it("sends lane filters to the backend inbox query", async () => {
    renderTasksRoute();

    fireEvent.click(screen.getByTestId("tasks-mode-inbox"));
    await waitFor(() => expect(getTaskInboxMock).toHaveBeenCalled());

    fireEvent.click(screen.getByTestId("tasks-inbox-filter-trigger"));
    fireEvent.click(await screen.findByRole("option", { name: "Lane" }));
    fireEvent.click(await screen.findByRole("option", { name: /Approvals/ }));
    await waitFor(() => {
      expect(getTaskInboxMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ lane: "approvals" }),
        expect.any(AbortSignal)
      );
    });
  });

  it("navigates to the route-based editor when the create action is clicked", async () => {
    listTasksMock.mockResolvedValue({
      facets: { owners: [], statuses: [{ count: 1, status: "draft" }] },
      page: { has_more: false, limit: 50, total: 1 },
      tasks: [
        buildTaskFixture({
          active_run: null,
          id: "task_abc",
          title: "Create API contract",
          status: "draft",
        }),
      ],
    });

    renderTasksRoute();
    await waitFor(() => expect(screen.getByTestId("tasks-open-create")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("tasks-open-create"));

    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({ search: expect.any(Function), to: "/tasks/new" })
    );
  });
});
