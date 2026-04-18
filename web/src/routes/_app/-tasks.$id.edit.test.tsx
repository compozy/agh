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

vi.mock("@/systems/tasks/components/task-editor-surface", () => ({
  TaskEditorSurface: (props: Record<string, unknown>) => (
    <div data-testid="task-editor-surface">
      <span data-testid="task-editor-mode">{String(props.mode)}</span>
      <span data-testid="task-editor-task-title">
        {String((props.task as { title: string }).title)}
      </span>
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

import { Route } from "./tasks.$id.edit";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskEditRoute = (Route as any).component as () => ReactNode;

describe("TaskEditRoute", () => {
  beforeEach(() => {
    routeParams = { id: "task_abc" };
    navigateMock.mockReset();
    updateMutateAsync.mockReset();
    updateMutateAsync.mockResolvedValue({ id: "task_abc" });
  });

  it("renders the editor in edit mode with the resolved task", async () => {
    render(<TaskEditRoute />);
    await waitFor(() => expect(screen.getByTestId("task-editor-surface")).toBeInTheDocument());
    expect(screen.getByTestId("task-editor-mode")).toHaveTextContent("edit");
    expect(screen.getByTestId("task-editor-task-title")).toHaveTextContent(
      "Summarize review feedback"
    );
  });

  it("updates the task and navigates back to detail after submit", async () => {
    render(<TaskEditRoute />);
    fireEvent.click(screen.getByTestId("task-editor-submit-trigger"));

    await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled());
    expect(navigateMock).toHaveBeenCalledWith({ params: { id: "task_abc" }, to: "/tasks/$id" });
  });
});
