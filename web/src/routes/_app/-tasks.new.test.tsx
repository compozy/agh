import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let searchParams: { template?: string } = {};
const navigateMock = vi.fn();
const createMutateAsync = vi.fn();
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
  useEnqueueTaskRun: () => ({
    isPending: false,
    mutateAsync: enqueueMutateAsync,
  }),
}));

vi.mock("@/systems/tasks/components/task-editor-surface", () => ({
  TaskEditorSurface: (props: Record<string, unknown>) => (
    <div data-testid="task-editor-surface">
      <span data-testid="task-editor-template-prop">{String(props.templateId)}</span>
      <button
        data-testid="task-editor-template-change"
        onClick={() => (props.onTemplateChange as (templateId: string) => void)("recurring")}
        type="button"
      >
        template
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
    </div>
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { Route } from "./tasks.new";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const TaskCreateRoute = (Route as any).component as () => ReactNode;

describe("TaskCreateRoute", () => {
  beforeEach(() => {
    searchParams = {};
    navigateMock.mockReset();
    createMutateAsync.mockReset();
    enqueueMutateAsync.mockReset();
    createMutateAsync.mockResolvedValue({ id: "task_created" });
    enqueueMutateAsync.mockResolvedValue({ id: "run_001" });
  });

  it("passes the selected template from search into the editor surface", () => {
    searchParams = { template: "recurring" };
    render(<TaskCreateRoute />);
    expect(screen.getByTestId("task-editor-template-prop")).toHaveTextContent("recurring");
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
});
