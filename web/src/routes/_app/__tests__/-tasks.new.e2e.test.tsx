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
  buildTaskRunRecordFixture,
  buildTaskTreeNodeFixture,
  buildTaskTreeFixture,
} from "@/systems/tasks/mocks/fixtures";
import { TASK_TEMPLATES } from "@/systems/tasks/lib/task-templates";
import { useTask, useTasks } from "@/systems/tasks";
import type {
  CreateTaskRequest,
  TaskDetailView,
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
let taskRuns: TaskRun[] = [];
let createTaskRequests: CreateTaskRequest[] = [];
let updateTaskRequests: UpdateTaskRequest[] = [];
let enqueuedTaskIds: string[] = [];
let createTaskResponseOverride: ((body: CreateTaskRequest) => Response | Promise<Response>) | null =
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
  return render(
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
    network_channel: body.network_channel,
    origin: { kind: "web", ref: "operator" },
    owner: body.owner ?? undefined,
    scope: body.scope,
    status: body.draft ? "draft" : "ready",
    title: body.title.trim(),
    updated_at: "2026-04-17T10:00:00Z",
    workspace_id: body.scope === "workspace" ? body.workspace : undefined,
  };

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

function listTasksForRequest(request: Request) {
  const url = new URL(request.url);
  const scope = url.searchParams.get("scope");
  const workspace = url.searchParams.get("workspace");
  const includeDrafts = url.searchParams.get("include_drafts") !== "false";
  const status = url.searchParams.get("status");

  return taskStore
    .listScoped({ scope, workspace })
    .filter(task => includeDrafts || !task.draft)
    .filter(task => !status || task.status === status);
}

function taskCreateHandlers(): HttpHandler[] {
  return [
    http.get("/api/workspaces", () => HttpResponse.json({ workspaces })),
    http.get("/api/tasks", ({ request }) =>
      HttpResponse.json({ tasks: listTasksForRequest(request) })
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

      return HttpResponse.json({ task: updated }, { status: 200 });
    }),
    http.delete("/api/tasks/:id", ({ params }) => {
      if (!taskStore.delete(String(params.id))) {
        return HttpResponse.json(
          { error: `Task not found: ${String(params.id)}` },
          { status: 404 }
        );
      }

      return new HttpResponse(null, { status: 204 });
    }),
    http.post("/api/tasks/:id/runs", ({ params }) => {
      const taskId = String(params.id);
      enqueuedTaskIds.push(taskId);
      const run = buildTaskRunRecordFixture({
        id: `run_${taskId}`,
        task_id: taskId,
        status: "queued",
        started_at: null,
      });
      taskRuns = [run, ...taskRuns];
      taskStore.patch(taskId, {
        active_run: {
          attempt: run.attempt,
          claimed_by: run.claimed_by,
          claim_token_hash: run.claim_token_hash,
          coordination_channel_id: run.coordination_channel_id,
          error: run.error,
          id: run.id,
          queued_at: run.queued_at,
          session_id: run.session_id,
          started_at: run.started_at,
          status: run.status,
          task_id: run.task_id,
        },
        status: "in_progress",
      } as Partial<StatefulTask>);

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
  taskRuns = [];
  handlers = taskCreateHandlers();
  createTaskRequests = [];
  updateTaskRequests = [];
  enqueuedTaskIds = [];
  createTaskResponseOverride = null;
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
    fireEvent.change(screen.getByTestId("task-network-input"), {
      target: { value: "launch-room" },
    });
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
        network_channel: "launch-room",
        owner: { kind: "agent_session", ref: "writer" },
        priority: "urgent",
        scope: "workspace",
        title: "Create API contract",
        workspace: "ws_beta",
      })
    );
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
    fireEvent.click(screen.getByTestId("tasks-detail-delete"));
    fireEvent.click(await screen.findByTestId("tasks-detail-delete-confirm"));

    await waitFor(() => {
      expect(taskStore.get("task_continuity_task")).toBeUndefined();
    });
    detail.unmount();
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
