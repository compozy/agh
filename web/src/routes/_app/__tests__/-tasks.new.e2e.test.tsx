// Suite: tasks create modal route E2E
// Invariant: the tasks create route submits the visible modal draft through the real query/adapters stack with the selected workspace scope.
// Boundary IN: TaskCreateRoute, TaskEditorModal, workspace query hooks, task mutation hooks, and openapi-fetch.
// Boundary OUT: AGH daemon HTTP implementation, replaced by MSW handlers at the fetch boundary.

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { delay, http, HttpResponse, type HttpHandler } from "msw";
import type { AnchorHTMLAttributes, ReactNode } from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import {
  buildCreatedTaskFixture,
  buildDetailFixture,
  buildTaskExecutionProfileFixture,
  buildTaskRunRecordFixture,
  buildTaskTreeNodeFixture,
  buildTaskTreeFixture,
} from "@/systems/tasks/mocks/fixtures";
import { buildTaskCatalogResponse } from "@/systems/tasks/mocks/query-responses";
import { SIMPLE_TASK_TEMPLATE_IDS, TASK_TEMPLATES } from "@/systems/tasks/lib/task-templates";
import { useTask, useTasks } from "@/systems/tasks";
import type {
  CreateTaskRequest,
  TaskDetailView,
  TaskExecutionProfile,
  TaskListItem,
  TaskRecord,
  TaskRun,
  UpdateTaskRequest,
} from "@/systems/tasks/types";
import type { WorkspacePayload } from "@/systems/workspace/types";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";
import {
  resetUserHomeDirStore,
  useUserHomeDirStore,
} from "@/systems/workspace/hooks/use-user-home-dir-store";
import { createMswFetch, createStatefulMswStore } from "@/test/msw-fetch";
import { buildLiveNetworkParticipationFixture } from "@/test/network-participation-fixtures";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";

const { navigateMock, toast } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

let searchParams: { template?: string } = {};
let routeParams: { id?: string } = {};
let childMatches: Array<{ id: string; params?: { id?: string } }> = [];

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: Record<string, unknown>;
  to?: string;
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, params, to, ...props }: MockLinkProps) => (
    <a href={String(to ?? "")} data-params={JSON.stringify(params ?? {})} {...props}>
      {children}
    </a>
  ),
  Outlet: () => <div data-testid="tasks-detail-outlet" />,
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => ({
      component: opts.component,
      useSearch: () => (opts.validateSearch ? opts.validateSearch(searchParams) : searchParams),
      useParams: () => routeParams,
    }),
  getRouteApi: () => ({
    useParams: () => routeParams,
    useSearch: () => searchParams,
  }),
  useChildMatches: () => childMatches,
  useNavigate: () => navigateMock,
  useRouter: () => ({ history: { back: vi.fn() } }),
}));

vi.mock("sonner", () => ({
  toast,
}));

import { Route as TaskCreateRoute } from "../tasks.new";
import { Route as TaskDetailRoute } from "../tasks.$id";
import { Route as TaskEditRoute } from "../tasks.$id.edit";

const TaskCreatePage = routeComponent(TaskCreateRoute);
const TaskDetailPage = routeComponent(TaskDetailRoute);
const TaskEditPage = routeComponent(TaskEditRoute);
const originalFetch = globalThis.fetch;

const workspaces: WorkspacePayload[] = [
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
];

let handlers: HttpHandler[] = [];
type StatefulTask = TaskRecord & TaskListItem;
let taskStore = createStatefulMswStore<StatefulTask>([]);
let taskExecutionProfiles = new Map<string, TaskExecutionProfile>();
let taskRuns: TaskRun[] = [];
let createTaskRequests: CreateTaskRequest[] = [];
let updateTaskRequests: UpdateTaskRequest[] = [];
let enqueuedTaskIds: string[] = [];
let createTaskResponseOverride: ((body: CreateTaskRequest) => Response | Promise<Response>) | null =
  null;
let updateTaskResponseOverride: ((body: UpdateTaskRequest) => Response | Promise<Response>) | null =
  null;

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
}

function renderTaskCreatePage() {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TaskCreatePage />
    </QueryClientProvider>
  );
}

function renderTaskDetailPage(taskId: string) {
  routeParams = { id: taskId };
  childMatches = [];
  const queryClient = createQueryClient();
  return renderWithTopbar(
    <QueryClientProvider client={queryClient}>
      <TaskDetailPage />
    </QueryClientProvider>
  );
}

function renderTaskEditPage(taskId: string) {
  routeParams = { id: taskId };
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TaskEditPage />
    </QueryClientProvider>
  );
}

function renderTaskListProbe(workspace: string) {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TaskListProbe workspace={workspace} />
    </QueryClientProvider>
  );
}

function renderTaskDetailProbe(taskId: string) {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TaskDetailProbe taskId={taskId} />
    </QueryClientProvider>
  );
}

function TaskListProbe({ workspace }: { workspace: string }) {
  const query = useTasks({ include_drafts: true, scope: "workspace", workspace });
  const tasks = query.data ?? [];

  return (
    <div data-testid={`task-list-probe-${workspace}`}>
      {tasks.map(task => (
        <article data-testid={`task-probe-item-${task.id}`} key={task.id}>
          {task.title}
        </article>
      ))}
    </div>
  );
}

function TaskDetailProbe({ taskId }: { taskId: string }) {
  const query = useTask(taskId);
  const detail = query.data;

  return (
    <div data-testid="task-detail-probe">
      {detail ? (
        <>
          <h1>{detail.task.title}</h1>
          <p>{detail.task.description}</p>
        </>
      ) : null}
    </div>
  );
}

function taskIdForTitle(title: string): string {
  const normalized = title
    .trim()
    .replace(/[^a-zA-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toLowerCase();
  return `task_${normalized || "created"}`;
}

function createdTaskFromBody(body: CreateTaskRequest): StatefulTask {
  const id = taskIdForTitle(body.title);
  const task = {
    ...buildCreatedTaskFixture(body),
    active_run: null,
    approval_policy: body.approval_policy,
    auto_enqueue_on_ready: body.auto_enqueue_on_ready,
    created_at: "2026-04-17T10:00:00Z",
    draft: body.draft ?? false,
    id,
    identifier: body.identifier ?? "TASK-NEW",
    last_activity_at: "2026-04-17T10:00:00Z",
    latest_event_seq: 1,
    max_attempts: body.max_attempts ?? 1,
    origin: { kind: "web", ref: "operator" },
    owner: body.owner ?? undefined,
    scope: body.scope,
    status: body.draft ? "draft" : "ready",
    title: body.title.trim(),
    updated_at: "2026-04-17T10:00:00Z",
    workspace_id: body.scope === "workspace" ? body.workspace : undefined,
  };

  taskExecutionProfiles.set(
    id,
    buildTaskExecutionProfileFixture({
      task_id: id,
      network_participation: body.network_participation,
    })
  );

  return task as StatefulTask;
}

function detailForTask(task: StatefulTask): TaskDetailView {
  const runs = taskRuns.filter(run => run.task_id === task.id);

  return buildDetailFixture({
    children: [],
    dependency_references: [],
    runs,
    summary: {
      ...task,
      active_run: task.active_run ?? null,
      child_count: task.child_count ?? 0,
      dependency_count: task.dependency_count ?? 0,
    },
    task,
  });
}

function taskCreateHandlers(): HttpHandler[] {
  return [
    http.get("/api/workspaces", () => HttpResponse.json({ workspaces })),
    http.get("/api/tasks", ({ request }) =>
      HttpResponse.json(buildTaskCatalogResponse(taskStore.all(), new URL(request.url)))
    ),
    http.get("/api/tasks/:id", ({ params }) => {
      const task = taskStore.get(String(params.id));
      if (!task) {
        return HttpResponse.json(
          { error: `Task not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      return HttpResponse.json({ task: detailForTask(task) });
    }),
    http.get("/api/tasks/:id/execution-profile", ({ params }) => {
      const taskId = String(params.id);
      if (!taskStore.get(taskId)) {
        return HttpResponse.json({ error: `Task not found: ${taskId}` }, { status: 404 });
      }
      const profile =
        taskExecutionProfiles.get(taskId) ?? buildTaskExecutionProfileFixture({ task_id: taskId });
      return HttpResponse.json({ profile });
    }),
    http.get("/api/tasks/:id/runs", ({ params }) =>
      HttpResponse.json({ runs: taskRuns.filter(run => run.task_id === String(params.id)) })
    ),
    http.get("/api/tasks/:id/timeline", () => HttpResponse.json({ timeline: [] })),
    http.get("/api/tasks/:id/tree", ({ params }) => {
      const task = taskStore.get(String(params.id));
      if (!task) {
        return HttpResponse.json(
          { error: `Task not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      return HttpResponse.json({
        tree: buildTaskTreeFixture({
          descendants: [],
          root: buildTaskTreeNodeFixture({
            active_run: task.active_run ?? null,
            child_count: 0,
            task,
          }),
        }),
      });
    }),
    http.get("/api/tasks/:id/inspect", () => HttpResponse.json({ inspect: null })),
    http.post("/api/tasks", async ({ request }) => {
      const body = (await request.json()) as CreateTaskRequest;
      createTaskRequests.push(body);
      if (createTaskResponseOverride) {
        return createTaskResponseOverride(body);
      }

      const task = taskStore.prepend(createdTaskFromBody(body));

      return HttpResponse.json({ task }, { status: 201 });
    }),
    http.patch("/api/tasks/:id", async ({ params, request }) => {
      const body = (await request.json()) as UpdateTaskRequest;
      updateTaskRequests.push(body);
      if (updateTaskResponseOverride) {
        return updateTaskResponseOverride(body);
      }
      const task = taskStore.get(String(params.id));
      if (!task) {
        return HttpResponse.json(
          { error: `Task not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      const updated = taskStore.patch(task.id, {
        ...body,
        description: body.description ?? task.description,
        owner: body.clear_owner ? undefined : (body.owner ?? task.owner),
        updated_at: "2026-04-17T10:05:00Z",
      } as Partial<StatefulTask>);
      if (body.network_participation) {
        const currentProfile =
          taskExecutionProfiles.get(task.id) ??
          buildTaskExecutionProfileFixture({ task_id: task.id });
        taskExecutionProfiles.set(task.id, {
          ...currentProfile,
          network_participation: body.network_participation,
          updated_at: "2026-04-17T10:05:00Z",
        });
      }

      return HttpResponse.json({ task: updated }, { status: 200 });
    }),
    http.delete("/api/tasks/:id", ({ params }) => {
      const taskId = String(params.id);
      if (!taskStore.delete(taskId)) {
        return HttpResponse.json({ error: `Task not found: ${taskId}` }, { status: 404 });
      }
      taskExecutionProfiles.delete(taskId);

      return new HttpResponse(null, { status: 204 });
    }),
    http.post("/api/tasks/:id/runs", ({ params }) => {
      const taskId = String(params.id);
      const task = taskStore.get(taskId);
      if (!task) {
        return HttpResponse.json({ error: `Task not found: ${taskId}` }, { status: 404 });
      }
      if (task.max_attempts === undefined) {
        return HttpResponse.json({ error: `Task has no retry policy: ${taskId}` }, { status: 500 });
      }
      enqueuedTaskIds.push(taskId);
      const run = buildTaskRunRecordFixture({
        id: `run_${taskId}`,
        task_id: taskId,
        status: "queued",
        started_at: null,
      });
      taskRuns = [run, ...taskRuns];
      const activeRun: NonNullable<TaskListItem["active_run"]> = {
        attempt: run.attempt,
        claimed_by: run.claimed_by,
        resolved_network_participation: run.resolved_network_participation,
        error: run.error,
        id: run.id,
        max_attempts: task.max_attempts,
        queued_at: run.queued_at,
        session_id: run.session_id,
        started_at: run.started_at,
        status: run.status,
        task_id: run.task_id,
      };
      taskStore.patch(taskId, {
        active_run: activeRun,
        status: "in_progress",
      });

      return HttpResponse.json(
        {
          run,
        },
        { status: 201 }
      );
    }),
  ];
}

beforeAll(() => {
  globalThis.fetch = createMswFetch(() => handlers);
});

afterAll(() => {
  globalThis.fetch = originalFetch;
});

beforeEach(() => {
  vi.spyOn(useActiveWorkspaceStore.persist, "hasHydrated").mockReturnValue(true);
  taskStore.reset([]);
  taskExecutionProfiles = new Map();
  taskRuns = [];
  handlers = taskCreateHandlers();
  createTaskRequests = [];
  updateTaskRequests = [];
  enqueuedTaskIds = [];
  createTaskResponseOverride = null;
  updateTaskResponseOverride = null;
  searchParams = {};
  routeParams = {};
  childMatches = [];
  navigateMock.mockReset();
  navigateMock.mockResolvedValue(undefined);
  toast.error.mockReset();
  toast.success.mockReset();
  delete (window as { __agh_xss?: boolean }).__agh_xss;
  resetUserHomeDirStore();
  useUserHomeDirStore.getState().setUserHomeDir("/home/operator");
  useActiveWorkspaceStore.setState({ selectedWorkspaceId: null });
});

function fillRequiredTaskFields(title: string, description?: string) {
  fireEvent.change(screen.getByTestId("task-title-input"), {
    target: { value: title },
  });
  if (description !== undefined) {
    fireEvent.change(screen.getByTestId("task-description-input"), {
      target: { value: description },
    });
  }
}

async function waitForTaskWorkspace(name = "Alpha") {
  await waitFor(() => {
    expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent(name);
  });
}

describe("TaskCreateRoute create modal", () => {
  it("Should open, fill, and submit a workspace-scoped draft through MSW", async () => {
    renderTaskCreatePage();

    expect(await screen.findByTestId("task-editor-modal")).toBeInTheDocument();
    expect(screen.getByTestId("task-editor-modal-title")).toHaveTextContent("Create task");
    await waitFor(() => {
      expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("Alpha");
    });

    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    expect(screen.getByTestId("task-parent-input")).toBeInTheDocument();
    expect(screen.getByTestId("task-execution-toggle")).toBeInTheDocument();
    expect(screen.queryByTestId("task-network-input")).not.toBeInTheDocument();

    fireEvent.change(screen.getByTestId("task-title-input"), {
      target: { value: "Create API contract" },
    });
    fireEvent.change(screen.getByTestId("task-description-input"), {
      target: { value: "Draft the contract payload." },
    });
    fireEvent.click(screen.getByTestId("task-priority-urgent"));
    fireEvent.click(screen.getByTestId("task-workspace-select"));
    fireEvent.click(screen.getByTestId("task-workspace-item-ws_beta"));
    expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("Beta");

    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "agent_session" } });
    fireEvent.change(screen.getByTestId("task-owner-ref"), { target: { value: "writer" } });
    fireEvent.click(screen.getByTestId("task-execution-toggle"));
    fireEvent.click(screen.getByTestId("task-save-draft-toggle"));
    expect(screen.getByTestId("task-editor-modal-submit")).toHaveTextContent("Save draft");

    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    expect(createTaskRequests[0]).toEqual(
      expect.objectContaining({
        description: "Draft the contract payload.",
        draft: true,
        owner: { kind: "agent_session", ref: "writer" },
        priority: "urgent",
        scope: "workspace",
        title: "Create API contract",
        workspace: "ws_beta",
      })
    );
    expect(createTaskRequests[0]).not.toHaveProperty("network_channel");
    expect(enqueuedTaskIds).toEqual([]);
    expect(navigateMock).toHaveBeenCalledWith({
      params: { id: "task_create_api_contract" },
      to: "/tasks/$id",
    });
    expect(toast.success).toHaveBeenCalledWith('Saved draft "Create API contract".');
  });

  it("Should route template selection through search params and render the selected card", async () => {
    const { rerender } = renderTaskCreatePage();

    expect(await screen.findByTestId("task-template-human_in_loop")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("task-template-human_in_loop"));

    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        search: expect.any(Function),
        to: "/tasks/new",
      })
    );
    expect(screen.getByTestId("task-template-human_in_loop")).toHaveAttribute(
      "aria-checked",
      "false"
    );

    searchParams = { template: "human_in_loop" };
    rerender(
      <QueryClientProvider client={createQueryClient()}>
        <TaskCreatePage />
      </QueryClientProvider>
    );

    expect(screen.getByTestId("task-template-human_in_loop")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  it("Should preserve the authored draft across every Simple template and mode round trip", async () => {
    const view = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields(
      "Preserve authored contract",
      "Keep this description through every template transition."
    );

    const assertAuthoredContract = () => {
      expect(screen.getByTestId("task-title-input")).toHaveValue("Preserve authored contract");
      expect(screen.getByTestId("task-description-input")).toHaveValue(
        "Keep this description through every template transition."
      );
    };
    const assertAuthoredAdvancedFields = () => {
      expect(screen.getByTestId("task-parent-input")).toHaveValue("task-parent-release");
      expect(screen.getByTestId("task-owner-kind")).toHaveValue("human");
      expect(screen.getByTestId("task-owner-ref")).toHaveValue("release-operator");
    };
    const rerenderWithTemplate = (template: string | undefined) => {
      searchParams = template ? { template } : {};
      view.rerender(
        <QueryClientProvider client={createQueryClient()}>
          <TaskCreatePage />
        </QueryClientProvider>
      );
    };

    expect(SIMPLE_TASK_TEMPLATE_IDS).toEqual(["one_shot", "human_in_loop", "epic"]);
    assertAuthoredContract();

    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    fireEvent.change(screen.getByTestId("task-parent-input"), {
      target: { value: "task-parent-release" },
    });
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "human" } });
    fireEvent.change(screen.getByTestId("task-owner-ref"), {
      target: { value: "release-operator" },
    });
    fireEvent.click(screen.getByTestId("task-priority-urgent"));
    fireEvent.click(screen.getByTestId("task-attempts-5"));
    fireEvent.click(screen.getByTestId("task-approval-manual"));
    assertAuthoredContract();
    assertAuthoredAdvancedFields();
    fireEvent.click(screen.getByTestId("task-mode-simple"));
    assertAuthoredContract();
    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    assertAuthoredAdvancedFields();
    expect(screen.getByTestId("task-priority-urgent")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("task-attempts-5")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("task-approval-manual")).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByTestId("task-mode-simple"));

    fireEvent.click(screen.getByTestId("task-template-human_in_loop"));
    rerenderWithTemplate("human_in_loop");
    assertAuthoredContract();
    expect(screen.getByTestId("task-priority-high")).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    assertAuthoredAdvancedFields();
    expect(screen.getByTestId("task-attempts-1")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("task-approval-manual")).toHaveAttribute("aria-pressed", "true");
    assertAuthoredContract();
    fireEvent.click(screen.getByTestId("task-mode-simple"));

    fireEvent.click(screen.getByTestId("task-template-epic"));
    rerenderWithTemplate("epic");
    assertAuthoredContract();
    expect(screen.getByTestId("task-priority-high")).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    assertAuthoredAdvancedFields();
    expect(screen.getByTestId("task-attempts-1")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("task-approval-none")).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByTestId("task-mode-simple"));
    assertAuthoredContract();

    fireEvent.click(screen.getByTestId("task-template-one_shot"));
    rerenderWithTemplate(undefined);
    assertAuthoredContract();
    expect(screen.getByTestId("task-priority-medium")).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByTestId("task-mode-advanced"));
    assertAuthoredAdvancedFields();
    expect(screen.getByTestId("task-attempts-1")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("task-approval-none")).toHaveAttribute("aria-pressed", "true");
  });

  it("Should block empty-title submission and surface the validation toast", async () => {
    renderTaskCreatePage();

    expect(await screen.findByTestId("task-editor-modal-submit")).toBeDisabled();

    fireEvent.submit(screen.getByTestId("task-editor-modal-form"));

    expect(createTaskRequests).toEqual([]);
    expect(toast.error).toHaveBeenCalledWith("Provide a title before creating the task.");
  });

  it("Should cancel without creating a task", async () => {
    renderTaskCreatePage();

    fireEvent.click(await screen.findByTestId("task-editor-modal-cancel"));

    expect(createTaskRequests).toEqual([]);
    expect(navigateMock).toHaveBeenCalledWith({ to: "/tasks" });
  });

  it("Should create the default task, enqueue one run, and render the created detail destination", async () => {
    const created = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Run launch checklist", "Start the launch checklist now.");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(enqueuedTaskIds).toEqual(["task_run_launch_checklist"]);
    });
    expect(createTaskRequests[0]).toEqual(
      expect.objectContaining({
        draft: false,
        scope: "workspace",
        title: "Run launch checklist",
        workspace: "ws_alpha",
      })
    );
    expect(toast.success).toHaveBeenCalledWith('Created task "Run launch checklist".');
    expect(navigateMock).toHaveBeenCalledWith({
      params: { id: "task_run_launch_checklist" },
      to: "/tasks/$id",
    });

    created.unmount();
    renderTaskDetailPage("task_run_launch_checklist");

    expect(await screen.findByTestId("tasks-detail-content")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-title")).toHaveTextContent("Run launch checklist");
    expect(screen.getByTestId("tasks-detail-active-run-card")).toHaveTextContent(
      "run_task_run_launch_checklist"
    );
  });

  it("Should keep workspace-created tasks isolated across workspace list reads", async () => {
    const created = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Alpha isolated task");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    created.unmount();

    const beta = renderTaskListProbe("ws_beta");
    await screen.findByTestId("task-list-probe-ws_beta");
    expect(
      screen.queryByTestId("task-probe-item-task_alpha_isolated_task")
    ).not.toBeInTheDocument();
    beta.unmount();

    renderTaskListProbe("ws_alpha");
    expect(await screen.findByTestId("task-probe-item-task_alpha_isolated_task")).toHaveTextContent(
      "Alpha isolated task"
    );
  });

  it("Should preserve task input after a create failure and retry without a duplicate task", async () => {
    let attempts = 0;
    createTaskResponseOverride = () => {
      attempts += 1;
      if (attempts === 1) {
        return HttpResponse.json({ error: "task store unavailable" }, { status: 500 });
      }
      const body = createTaskRequests.at(-1);
      if (!body) {
        return HttpResponse.json({ error: "missing task body" }, { status: 500 });
      }
      const task = taskStore.prepend(createdTaskFromBody(body));
      return HttpResponse.json({ task }, { status: 201 });
    };
    renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Retryable task", "Keep this text after failure.");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("task store unavailable");
    });
    expect(screen.getByTestId("task-title-input")).toHaveValue("Retryable task");
    expect(screen.getByTestId("task-description-input")).toHaveValue(
      "Keep this text after failure."
    );
    expect(navigateMock).not.toHaveBeenCalledWith(expect.objectContaining({ to: "/tasks/$id" }));
    expect(taskStore.all()).toHaveLength(0);

    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(2);
    });
    expect(taskStore.all()).toHaveLength(1);
    expect(taskStore.get("task_retryable_task")?.title).toBe("Retryable task");
  });

  it("Should ignore a rapid second task submit while create and enqueue are pending", async () => {
    createTaskResponseOverride = async body => {
      await delay(50);
      const task = taskStore.prepend(createdTaskFromBody(body));
      return HttpResponse.json({ task }, { status: 201 });
    };
    renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Single submit task");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("task-editor-modal-submit")).toBeDisabled();
    });
    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    await waitFor(() => {
      expect(enqueuedTaskIds).toEqual(["task_single_submit_task"]);
    });
  });

  it("Should escape pasted HTML in the created task detail while preserving payload text", async () => {
    const unsafeDescription = `${"<script>window.__agh_xss = true</script>"}${"x".repeat(5_000)}`;
    const created = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("HTML paste task", unsafeDescription);
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    expect(createTaskRequests[0]?.description).toBe(unsafeDescription);
    created.unmount();

    renderTaskDetailPage("task_html_paste_task");

    const description = await screen.findByTestId("tasks-detail-description-card");
    expect(description).toHaveTextContent("x".repeat(100));
    expect(document.querySelector("script")).toBeNull();
    expect((window as { __agh_xss?: boolean }).__agh_xss).toBeUndefined();
  });

  it("Should re-fetch a created task after remount with a fresh query client", async () => {
    const created = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Refresh visible task", "Re-fetch after remount.");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    created.unmount();

    const firstRead = renderTaskDetailProbe("task_refresh_visible_task");
    expect(await screen.findByText("Refresh visible task")).toBeInTheDocument();
    firstRead.unmount();

    renderTaskDetailProbe("task_refresh_visible_task");
    expect(await screen.findByText("Refresh visible task")).toBeInTheDocument();
  });

  it("Should create, edit, and delete the same task through stateful GET/PATCH/DELETE reads", async () => {
    const created = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields("Continuity task", "Original task body.");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    created.unmount();

    const edit = renderTaskEditPage("task_continuity_task");
    expect(await screen.findByTestId("task-title-input")).toHaveValue("Continuity task");
    expect(screen.getByTestId("task-description-input")).toHaveValue("Original task body.");

    fireEvent.change(screen.getByTestId("task-title-input"), {
      target: { value: "Continuity task edited" },
    });
    fireEvent.change(screen.getByTestId("task-description-input"), {
      target: { value: "Edited task body." },
    });
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(updateTaskRequests).toHaveLength(1);
    });
    expect(updateTaskRequests[0]).toEqual(
      expect.objectContaining({
        description: "Edited task body.",
        title: "Continuity task edited",
      })
    );
    edit.unmount();

    const detail = renderTaskDetailPage("task_continuity_task");
    expect(await screen.findByTestId("tasks-detail-title")).toHaveTextContent(
      "Continuity task edited"
    );
    fireEvent.click(screen.getByTestId("tasks-detail-overflow"));
    fireEvent.click(screen.getByTestId("tasks-detail-delete"));
    fireEvent.click(await screen.findByTestId("tasks-detail-delete-confirm"));

    await waitFor(() => {
      expect(taskStore.get("task_continuity_task")).toBeUndefined();
    });
    detail.unmount();
  });

  it("Should clear an exact-session owner through PATCH and show Unassigned on a fresh edit read", async () => {
    const task = createdTaskFromBody({
      approval_policy: "manual",
      auto_enqueue_on_ready: true,
      description: "Preserve every mutable field while clearing ownership.",
      draft: false,
      max_attempts: 3,
      network_participation: {
        mode: "live",
        channel_id: "launch-room",
        channel_strategy: "named",
      },
      owner: { kind: "agent_session", ref: "sess-exact-owner" },
      priority: "urgent",
      scope: "workspace",
      title: "Release exact owner",
      workspace: "ws_alpha",
    });
    task.resolved_network_participation = buildLiveNetworkParticipationFixture({
      workspaceId: "ws_alpha",
      channelId: "launch-room",
    });
    taskStore.reset([task]);

    const edit = renderTaskEditPage(task.id);
    expect(await screen.findByTestId("task-owner-kind")).toHaveValue("agent_session");
    expect(screen.getByTestId("task-owner-ref")).toHaveValue("sess-exact-owner");

    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "" } });

    expect(screen.getByTestId("task-owner-kind")).toHaveValue("");
    expect(screen.getByTestId("task-owner-ref")).toBeDisabled();
    expect(screen.getByTestId("task-owner-ref")).toHaveValue("");
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(updateTaskRequests).toHaveLength(1);
    });
    expect(updateTaskRequests[0]).toEqual({
      approval_policy: "manual",
      auto_enqueue_on_ready: true,
      clear_owner: true,
      description: "Preserve every mutable field while clearing ownership.",
      max_attempts: 3,
      network_participation: {
        mode: "live",
        channel_id: "launch-room",
        channel_strategy: "named",
      },
      priority: "urgent",
      title: "Release exact owner",
    });
    expect(taskStore.get(task.id)?.owner).toBeUndefined();
    edit.unmount();

    renderTaskEditPage(task.id);
    expect(await screen.findByTestId("task-owner-kind")).toHaveValue("");
    expect(screen.getByTestId("task-owner-ref")).toBeDisabled();
    expect(screen.getByTestId("task-owner-ref")).toHaveValue("");
    expect(screen.getByTestId("task-title-input")).toHaveValue("Release exact owner");
    expect(screen.getByTestId("task-description-input")).toHaveValue(
      "Preserve every mutable field while clearing ownership."
    );
  });

  it("Should preserve a non-clear owner edit and omit the clear operation", async () => {
    const task = createdTaskFromBody({
      owner: { kind: "agent_session", ref: "sess-original" },
      scope: "workspace",
      title: "Reassign exact owner",
      workspace: "ws_alpha",
    });
    taskStore.reset([task]);
    renderTaskEditPage(task.id);

    expect(await screen.findByTestId("task-owner-kind")).toHaveValue("agent_session");
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "human" } });
    fireEvent.change(screen.getByTestId("task-owner-ref"), { target: { value: "pedro" } });
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(updateTaskRequests).toHaveLength(1);
    });
    expect(updateTaskRequests[0]?.owner).toEqual({ kind: "human", ref: "pedro" });
    expect(updateTaskRequests[0]).not.toHaveProperty("clear_owner");
    expect(taskStore.get(task.id)?.owner).toEqual({ kind: "human", ref: "pedro" });
  });

  it("Should cancel an owner clear without sending a PATCH", async () => {
    const task = createdTaskFromBody({
      owner: { kind: "agent_session", ref: "sess-keep-owner" },
      scope: "workspace",
      title: "Keep owner on cancel",
      workspace: "ws_alpha",
    });
    taskStore.reset([task]);
    renderTaskEditPage(task.id);

    expect(await screen.findByTestId("task-owner-kind")).toHaveValue("agent_session");
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "" } });
    fireEvent.click(screen.getByTestId("task-editor-modal-cancel"));

    expect(updateTaskRequests).toEqual([]);
    expect(taskStore.get(task.id)?.owner).toEqual({
      kind: "agent_session",
      ref: "sess-keep-owner",
    });
    expect(navigateMock).toHaveBeenCalledWith({ params: { id: task.id }, to: "/tasks/$id" });
  });

  it("Should preserve the edited draft and owner-clear intent after a PATCH error", async () => {
    const task = createdTaskFromBody({
      description: "Persisted description.",
      owner: { kind: "agent_session", ref: "sess-error-owner" },
      scope: "workspace",
      title: "Retry owner clear",
      workspace: "ws_alpha",
    });
    taskStore.reset([task]);
    updateTaskResponseOverride = () =>
      HttpResponse.json({ error: "task store unavailable" }, { status: 500 });
    renderTaskEditPage(task.id);

    expect(await screen.findByTestId("task-owner-kind")).toHaveValue("agent_session");
    fireEvent.change(screen.getByTestId("task-title-input"), {
      target: { value: "Retry owner clear after failure" },
    });
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "" } });
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("task store unavailable");
    });
    expect(updateTaskRequests[0]).toEqual(expect.objectContaining({ clear_owner: true }));
    expect(updateTaskRequests[0]).not.toHaveProperty("owner");
    expect(screen.getByTestId("task-editor-modal")).toBeInTheDocument();
    expect(screen.getByTestId("task-title-input")).toHaveValue("Retry owner clear after failure");
    expect(screen.getByTestId("task-owner-kind")).toHaveValue("");
    expect(screen.getByTestId("task-owner-ref")).toHaveValue("");
    expect(taskStore.get(task.id)?.title).toBe("Retry owner clear");
    expect(taskStore.get(task.id)?.owner).toEqual({
      kind: "agent_session",
      ref: "sess-error-owner",
    });
    expect(navigateMock).not.toHaveBeenCalledWith(
      expect.objectContaining({ params: { id: task.id }, to: "/tasks/$id" })
    );
  });

  it("Should submit task owner kind/ref combinations without dropping an explicit null owner", async () => {
    const ownerNull = renderTaskCreatePage();

    await waitForTaskWorkspace();
    fireEvent.click(await screen.findByTestId("task-mode-advanced"));
    fillRequiredTaskFields("Owner kind only");
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "human" } });
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    expect(createTaskRequests[0]?.owner).toBeNull();
    ownerNull.unmount();

    const ownerSpecific = renderTaskCreatePage();
    await waitForTaskWorkspace();
    fireEvent.click(await screen.findByTestId("task-mode-advanced"));
    fillRequiredTaskFields("Owner specific");
    fireEvent.change(screen.getByTestId("task-owner-kind"), { target: { value: "human" } });
    fireEvent.change(screen.getByTestId("task-owner-ref"), { target: { value: "pedro" } });
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(2);
    });
    expect(createTaskRequests[1]?.owner).toEqual({ kind: "human", ref: "pedro" });
    ownerSpecific.unmount();
  });

  it.each(TASK_TEMPLATES)("Should preserve $id template draft/enqueue behavior", async template => {
    searchParams = { template: template.id };
    renderTaskCreatePage();

    await waitForTaskWorkspace();
    fillRequiredTaskFields(`Template ${template.id}`);
    fireEvent.click(screen.getByTestId("task-editor-modal-submit"));

    await waitFor(() => {
      expect(createTaskRequests).toHaveLength(1);
    });
    const expectedDraft = template.id === "recurring";
    const expectedEnqueueCount = expectedDraft || !template.preview.enqueueOnSubmit ? 0 : 1;
    expect(createTaskRequests[0]?.draft).toBe(expectedDraft);
    expect(enqueuedTaskIds).toHaveLength(expectedEnqueueCount);
    if (template.id === "human_in_loop") {
      expect(createTaskRequests[0]?.approval_policy).toBe("manual");
    }
  });
});
