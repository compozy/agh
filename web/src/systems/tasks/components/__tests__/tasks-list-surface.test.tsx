import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
      <a
        href={typeof to === "string" ? to : "#"}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
      >
        {children as React.ReactNode}
      </a>
    ),
  };
});

const { TasksListSurface } = await import("../tasks-list-surface");
type TaskListItem = import("../../types").TaskListItem;

function buildTask(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    title: "Generate API client",
    identifier: "TASK-1",
    status: "ready",
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
    ...overrides,
  } as TaskListItem;
}

interface RenderOptions {
  tasks?: TaskListItem[];
  totalCount?: number;
  isLoading?: boolean;
  errorMessage?: string | null;
  statusFilter?: TaskListItem["status"] | null;
  workspaceName?: string | null;
  listUpdatedAt?: number;
}

function renderSurface(options: RenderOptions = {}) {
  return render(
    <UIProvider reducedMotion="always">
      <TasksListSurface
        errorMessage={options.errorMessage ?? null}
        isLoading={options.isLoading}
        listUpdatedAt={options.listUpdatedAt}
        onOwnerChange={() => {}}
        onPriorityChange={() => {}}
        onSearchQueryChange={() => {}}
        onSortChange={() => {}}
        onStatusChange={() => {}}
        ownerFilter={null}
        ownerOptions={[]}
        priorityFilter={null}
        searchQuery=""
        sortBy="recent"
        statusFilter={options.statusFilter ?? null}
        tasks={options.tasks ?? []}
        totalCount={options.totalCount ?? 0}
        workspaceName={options.workspaceName ?? "agh-runtime"}
      />
    </UIProvider>
  );
}

describe("TasksListSurface", () => {
  it("Should partition tasks into the canonical status group sections", () => {
    const tasks = [
      buildTask({ id: "a", title: "Active task", status: "in_progress" }),
      buildTask({ id: "b", title: "Blocked task", status: "blocked" }),
      buildTask({ id: "n", title: "Escalated task", status: "needs_attention" }),
      buildTask({ id: "c", title: "Queued task", status: "ready" }),
      buildTask({ id: "d", title: "Done task", status: "completed" }),
      buildTask({ id: "e", title: "Failed task", status: "failed" }),
    ];

    const { container } = renderSurface({ tasks, totalCount: tasks.length });

    expect(screen.getByTestId("task-group-active-label")).toHaveTextContent(/active/i);
    expect(screen.getByTestId("task-group-active-count")).toHaveTextContent("1");
    expect(screen.getByTestId("task-group-blocked")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-needs_attention")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-queued")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-done")).toBeInTheDocument();
    expect(screen.getByTestId("task-group-failed")).toBeInTheDocument();

    expect(screen.getByTestId("task-group-dot-active")).toHaveAttribute("data-tone", "accent");
    expect(screen.getByTestId("task-group-dot-active")).toHaveAttribute("data-variant", "ring");
    expect(screen.getByTestId("task-group-dot-blocked")).toHaveAttribute("data-tone", "danger");
    expect(screen.getByTestId("task-group-dot-needs_attention")).toHaveAttribute(
      "data-tone",
      "warning"
    );
    expect(screen.getByTestId("task-group-dot-queued")).toHaveAttribute("data-tone", "faint");
    expect(screen.getByTestId("task-group-dot-queued")).toHaveAttribute("data-variant", "ring");
    expect(screen.getByTestId("task-group-dot-done")).toHaveAttribute("data-tone", "faint");
    expect(screen.getByTestId("task-group-dot-done")).toHaveAttribute("data-variant", "solid");
    expect(screen.getByTestId("task-group-dot-failed")).toHaveAttribute("data-tone", "danger");

    const rowDots = container.querySelectorAll(
      '[data-slot="tasks-list-row"] [data-slot="status-dot"]'
    );
    expect(rowDots).toHaveLength(0);
  });

  it("Should link each list row to /tasks/$id", () => {
    renderSurface({
      tasks: [buildTask({ id: "task_777", title: "Linked task" })],
      totalCount: 1,
    });

    const row = screen.getByTestId("task-card-task_777");
    const link = row.querySelector("a");
    expect(link).not.toBeNull();
    expect(link).toHaveAttribute("href", "/tasks/$id");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ id: "task_777" }));
  });

  it("Should render search and forward list query changes", () => {
    const handleSearchQueryChange = vi.fn();
    render(
      <UIProvider reducedMotion="always">
        <TasksListSurface
          onOwnerChange={() => {}}
          onPriorityChange={() => {}}
          onSearchQueryChange={handleSearchQueryChange}
          onSortChange={() => {}}
          onStatusChange={() => {}}
          ownerFilter={null}
          ownerOptions={[]}
          priorityFilter={null}
          searchQuery="api"
          sortBy="recent"
          statusFilter={null}
          tasks={[buildTask({ id: "task_search" })]}
          totalCount={1}
        />
      </UIProvider>
    );

    const search = screen.getByTestId("tasks-list-search-input");
    expect(search).toHaveValue("api");
    fireEvent.change(search, { target: { value: "deploy" } });
    expect(handleSearchQueryChange).toHaveBeenCalledWith("deploy");
  });

  it("Should render the empty state when the list is empty", () => {
    renderSurface({ tasks: [], totalCount: 0 });
    expect(screen.getByTestId("tasks-list-surface-empty")).toBeInTheDocument();
  });

  it("Should render the loading skeleton when isLoading and no tasks", () => {
    renderSurface({ tasks: [], totalCount: 0, isLoading: true });
    expect(screen.getByTestId("tasks-list-surface-loading")).toBeInTheDocument();
  });

  it("Should render the error state when errorMessage is set and no tasks", () => {
    renderSurface({ tasks: [], totalCount: 0, errorMessage: "Unable to reach the daemon" });
    expect(screen.getByTestId("tasks-list-surface-error")).toBeInTheDocument();
    expect(screen.getByText(/unable to reach the daemon/i)).toBeInTheDocument();
  });

  it("Should render the page header with title, count, and workspace meta", () => {
    renderSurface({
      tasks: [buildTask({ id: "a", status: "ready" })],
      totalCount: 4,
      workspaceName: "agh-runtime",
      listUpdatedAt: Date.now() - 90_000,
    });

    expect(screen.getByTestId("tasks-list-page-title")).toHaveTextContent("Tasks");
    expect(screen.getByTestId("tasks-list-page-count")).toHaveTextContent("1 of 4");
    expect(screen.getByTestId("tasks-list-page-workspace")).toHaveTextContent("agh-runtime");
    expect(screen.getByTestId("tasks-list-page-synced")).toHaveTextContent(/synced/i);
  });
});
