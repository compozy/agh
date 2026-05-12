import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let searchParams: { template?: string } = {};
const navigateMock = vi.fn();
const createMutateAsync = vi.fn();
const createChildMutateAsync = vi.fn();
const enqueueMutateAsync = vi.fn();

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
  useNavigate: () => navigateMock,
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_alpha", name: "Alpha" },
    activeWorkspaceId: "ws_alpha",
  }),
}));

vi.mock("@/systems/tasks", () => ({
  useCreateTask: () => ({
    isPending: false,
    mutateAsync: createMutateAsync,
  }),
  useCreateChildTask: () => ({
    isPending: false,
    mutateAsync: createChildMutateAsync,
  }),
  useEnqueueTaskRun: () => ({
    isPending: false,
    mutateAsync: enqueueMutateAsync,
  }),
}));

vi.mock("@/systems/tasks/components/task-editor-modal", () => ({
  TaskEditorModal: (props: Record<string, unknown>) => (
    <div data-testid="task-editor-modal">
      <span data-testid="task-editor-mode">{String(props.mode)}</span>
      <span data-testid="task-editor-template-prop">{String(props.templateId)}</span>
      <span data-testid="task-editor-open">{String(props.open)}</span>
      <button
        data-testid="task-editor-template-change"
        onClick={() => (props.onTemplateChange as (templateId: string) => void)("recurring")}
        type="button"
      >
        template
      </button>
      <button
        data-testid="task-editor-close-trigger"
        onClick={() => (props.onOpenChange as (open: boolean) => void)(false)}
        type="button"
      >
        close
      </button>
      <button
        data-testid="task-editor-submit-trigger"
        onClick={() =>
          void (
            props.onSubmit as (draft: Record<string, unknown>, asDraft: boolean) => Promise<unknown>
          )({ ...(props.draft as Record<string, unknown>), title: "Create API contract" }, false)
        }
        type="button"
      >
        submit
      </button>
      <button
        data-testid="task-editor-submit-child-trigger"
        onClick={() =>
          void (
            props.onSubmit as (draft: Record<string, unknown>, asDraft: boolean) => Promise<unknown>
          )(
            {
              ...(props.draft as Record<string, unknown>),
              title: "Child task contract",
              parentTaskId: " task_parent_001 ",
            },
            false
          )
        }
        type="button"
      >
        submit child
      </button>
    </div>
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { routeComponent } from "@/test/route-options";
import { Route } from "../tasks.new";

const TaskCreateRoute = routeComponent(Route);

describe("TaskCreateRoute", () => {
  beforeEach(() => {
    searchParams = {};
    navigateMock.mockReset();
    createMutateAsync.mockReset();
    createChildMutateAsync.mockReset();
    enqueueMutateAsync.mockReset();
    createMutateAsync.mockResolvedValue({ id: "task_created" });
    createChildMutateAsync.mockResolvedValue({ id: "task_child_created" });
    enqueueMutateAsync.mockResolvedValue({ id: "run_001" });
  });

  it("mounts the editor modal in new mode and forwards the selected template", () => {
    searchParams = { template: "recurring" };
    render(<TaskCreateRoute />);
    expect(screen.getByTestId("task-editor-mode")).toHaveTextContent("new");
    expect(screen.getByTestId("task-editor-open")).toHaveTextContent("true");
    expect(screen.getByTestId("task-editor-template-prop")).toHaveTextContent("recurring");
  });

  it("navigates back to /tasks when the modal closes", () => {
    render(<TaskCreateRoute />);
    fireEvent.click(screen.getByTestId("task-editor-close-trigger"));
    expect(navigateMock).toHaveBeenCalledWith({ to: "/tasks" });
  });

  it("updates the route search when the editor switches template", () => {
    render(<TaskCreateRoute />);
    fireEvent.click(screen.getByTestId("task-editor-template-change"));
    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        search: expect.any(Function),
        to: "/tasks/new",
      })
    );
  });

  it("creates the task and navigates to the detail route after submit", async () => {
    render(<TaskCreateRoute />);
    fireEvent.click(screen.getByTestId("task-editor-submit-trigger"));

    await waitFor(() => expect(createMutateAsync).toHaveBeenCalled());
    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        params: { id: "task_created" },
        to: "/tasks/$id",
      })
    );
  });

  it("creates a child task through the child endpoint when a parent task id is provided", async () => {
    render(<TaskCreateRoute />);
    fireEvent.click(screen.getByTestId("task-editor-submit-child-trigger"));

    await waitFor(() =>
      expect(createChildMutateAsync).toHaveBeenCalledWith({
        parentId: "task_parent_001",
        data: expect.objectContaining({
          title: "Child task contract",
          workspace: "ws_alpha",
        }),
      })
    );
    expect(createMutateAsync).not.toHaveBeenCalled();
    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        params: { id: "task_child_created" },
        to: "/tasks/$id",
      })
    );
  });
});
