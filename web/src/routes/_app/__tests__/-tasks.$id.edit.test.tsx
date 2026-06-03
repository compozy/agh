import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let routeParams = { id: "task_abc" };
const navigateMock = vi.fn();
const updateMutateAsync = vi.fn();
const taskDetailFixture = {
  task: {
    id: "task_abc",
    identifier: "TASK-42",
    title: "Summarize review feedback",
    status: "draft",
    scope: "workspace",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    workspace_id: "ws_alpha",
  },
};

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => routeParams,
  }),
  getRouteApi: () => ({
    useParams: () => routeParams,
  }),
  useNavigate: () => navigateMock,
}));

vi.mock("@/systems/tasks", () => ({
  useTask: () => ({
    data: taskDetailFixture,
    isLoading: false,
  }),
  useUpdateTask: () => ({
    isPending: false,
    mutateAsync: updateMutateAsync,
  }),
}));

vi.mock("@/systems/tasks/components/task-editor-modal", () => ({
  TaskEditorModal: (props: Record<string, unknown>) => (
    <div data-testid="task-editor-modal">
      <span data-testid="task-editor-mode">{String(props.mode)}</span>
      <span data-testid="task-editor-open">{String(props.open)}</span>
      <span data-testid="task-editor-task-title">
        {String((props.task as { title: string }).title)}
      </span>
      <button
        data-testid="task-editor-close-trigger"
        onClick={() => (props.onOpenChange as (open: boolean) => void)(false)}
        type="button"
      >
        close
      </button>
      <button
        data-testid="task-editor-submit-trigger"
        onClick={() => void (props.onSubmit as (draft: unknown) => Promise<unknown>)(props.draft)}
        type="button"
      >
        submit
      </button>
    </div>
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { routeComponent } from "@/test/route-options";
import { Route } from "../tasks.$id.edit";

const TaskEditRoute = routeComponent(Route);

describe("TaskEditRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc" };
    navigateMock.mockReset();
    updateMutateAsync.mockReset();
    updateMutateAsync.mockResolvedValue({ id: "task_abc" });
  });

  it("renders the editor modal in edit mode with the resolved task", async () => {
    render(<TaskEditRoute />);
    await waitFor(() => expect(screen.getByTestId("task-editor-modal")).toBeInTheDocument());
    expect(screen.getByTestId("task-editor-mode")).toHaveTextContent("edit");
    expect(screen.getByTestId("task-editor-open")).toHaveTextContent("true");
    expect(screen.getByTestId("task-editor-task-title")).toHaveTextContent(
      "Summarize review feedback"
    );
  });

  it("navigates back to the detail page when the modal closes", () => {
    render(<TaskEditRoute />);
    fireEvent.click(screen.getByTestId("task-editor-close-trigger"));
    expect(navigateMock).toHaveBeenCalledWith({ params: { id: "task_abc" }, to: "/tasks/$id" });
  });

  it("updates the task and navigates back to detail after submit", async () => {
    render(<TaskEditRoute />);
    fireEvent.click(screen.getByTestId("task-editor-submit-trigger"));

    await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled());
    expect(navigateMock).toHaveBeenCalledWith({ params: { id: "task_abc" }, to: "/tasks/$id" });
  });
});
