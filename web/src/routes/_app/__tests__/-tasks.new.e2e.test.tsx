// Suite: tasks create modal route E2E
// Invariant: the tasks create route submits the visible modal draft through the real query/adapters stack with the selected workspace scope.
// Boundary IN: TaskCreateRoute, TaskEditorModal, workspace query hooks, task mutation hooks, and openapi-fetch.
// Boundary OUT: AGH daemon HTTP implementation, replaced by MSW handlers at the fetch boundary.

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse, type HttpHandler } from "msw";
import type { ReactNode } from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { buildCreatedTaskFixture, buildTaskRunRecordFixture } from "@/systems/tasks/mocks/fixtures";
import type { CreateTaskRequest } from "@/systems/tasks/types";
import type { WorkspacePayload } from "@/systems/workspace/types";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";
import {
  resetUserHomeDirStore,
  useUserHomeDirStore,
} from "@/systems/workspace/hooks/use-user-home-dir-store";
import { createMswFetch } from "@/test/msw-fetch";
import { routeComponent } from "@/test/route-options";

const { navigateMock, toast } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

let searchParams: { template?: string } = {};

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => ({
      component: opts.component,
      useSearch: () => (opts.validateSearch ? opts.validateSearch(searchParams) : searchParams),
    }),
  getRouteApi: () => ({
    useSearch: () => searchParams,
  }),
  useNavigate: () => navigateMock,
}));

vi.mock("sonner", () => ({
  toast,
}));

import { Route } from "../tasks.new";

const TaskCreatePage = routeComponent(Route);
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
let createTaskRequests: CreateTaskRequest[] = [];
let enqueuedTaskIds: string[] = [];

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

function taskCreateHandlers(): HttpHandler[] {
  return [
    http.get("/api/workspaces", () => HttpResponse.json({ workspaces })),
    http.post("/api/tasks", async ({ request }) => {
      const body = (await request.json()) as CreateTaskRequest;
      createTaskRequests.push(body);
      const task = {
        ...buildCreatedTaskFixture(body),
        draft: body.draft ?? false,
        scope: body.scope,
        workspace_id: body.workspace,
      };

      return HttpResponse.json({ task }, { status: 201 });
    }),
    http.post("/api/tasks/:id/runs", ({ params }) => {
      const taskId = String(params.id);
      enqueuedTaskIds.push(taskId);

      return HttpResponse.json(
        {
          run: buildTaskRunRecordFixture({
            id: `run_${taskId}`,
            task_id: taskId,
            status: "queued",
            started_at: null,
          }),
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
  handlers = taskCreateHandlers();
  createTaskRequests = [];
  enqueuedTaskIds = [];
  searchParams = {};
  navigateMock.mockReset();
  navigateMock.mockResolvedValue(undefined);
  toast.error.mockReset();
  toast.success.mockReset();
  resetUserHomeDirStore();
  useUserHomeDirStore.getState().setUserHomeDir("/home/operator");
  useActiveWorkspaceStore.setState({ selectedWorkspaceId: null });
});

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
      params: { id: "task_created" },
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
});
