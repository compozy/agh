import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailPreviewPanel } from "./tasks-detail-preview-panel";
import type { TaskDetailView, TaskListItem } from "../types";

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
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    child_count: 1,
    dependency_count: 2,
    ...overrides,
  } as TaskListItem;
}

function buildDetail(task: TaskListItem, overrides: Partial<TaskDetailView> = {}): TaskDetailView {
  return {
    task: { ...task, description: "Build typed bindings and tests for payments-v3" },
    ...overrides,
  } as TaskDetailView;
}

describe("TasksDetailPreviewPanel", () => {
  it("renders an empty placeholder when no task is selected", () => {
    render(<TasksDetailPreviewPanel detail={null} task={null} />);
    expect(screen.getByTestId("tasks-detail-preview-empty")).toBeInTheDocument();
  });

  it("renders the loading state until detail data resolves", () => {
    render(<TasksDetailPreviewPanel detail={null} isLoading task={buildTask()} />);
    expect(screen.getByTestId("tasks-detail-preview-loading")).toBeInTheDocument();
  });

  it("renders the error state when the detail fetch fails", () => {
    render(<TasksDetailPreviewPanel detail={null} errorMessage="boom" task={buildTask()} />);
    expect(screen.getByTestId("tasks-detail-preview-error")).toHaveTextContent("boom");
  });

  it("renders enriched detail summary, counts, deep link, and actions", () => {
    const onDeleteTask = vi.fn();
    const onPublishTask = vi.fn();
    const onEnqueueRun = vi.fn();
    const task = buildTask();
    const detail = buildDetail(task, {
      children: [{ id: "child_1" } as never],
      dependency_references: [
        { depends_on_task_id: "dep1", task_id: task.id, kind: "blocks", created_at: "" } as never,
      ],
      runs: [{ id: "run_a" } as never, { id: "run_b" } as never],
    });

    render(
      <TasksDetailPreviewPanel
        detail={detail}
        onDeleteTask={onDeleteTask}
        onEnqueueRun={onEnqueueRun}
        onPublishTask={onPublishTask}
        task={task}
      />
    );

    expect(screen.getByTestId("tasks-detail-preview-title")).toHaveTextContent(
      "Generate API client"
    );
    expect(screen.getByTestId("tasks-detail-preview-counts-children")).toHaveTextContent("1");
    expect(screen.getByTestId("tasks-detail-preview-counts-deps")).toHaveTextContent("1");
    expect(screen.getByTestId("tasks-detail-preview-counts-runs")).toHaveTextContent("2");
    expect(screen.getByTestId("tasks-detail-preview-deeplink")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-detail-preview-enqueue"));
    expect(onEnqueueRun).toHaveBeenCalledWith(task.id);

    fireEvent.click(screen.getByTestId("tasks-detail-preview-delete"));
    fireEvent.click(screen.getByTestId("tasks-detail-preview-delete-confirm"));
    expect(onDeleteTask).toHaveBeenCalledWith(task.id);
  });

  it("wraps the task preview in CodeBlock with the yaml language when task.kind === 'yaml'", () => {
    const task = buildTask();
    const detail = buildDetail(task);
    (detail.task as unknown as { kind: string }).kind = "yaml";

    const { container } = render(<TasksDetailPreviewPanel detail={detail} task={task} />);

    const code = container.querySelector('[data-slot="code-block"]');
    expect(code).not.toBeNull();
    const language = container.querySelector('[data-slot="code-block-language"]');
    expect(language).toHaveTextContent("yaml");
  });

  it("offers a publish action for draft tasks", () => {
    const onPublishTask = vi.fn();
    const task = buildTask({ status: "draft", draft: true });
    render(
      <TasksDetailPreviewPanel
        detail={buildDetail(task)}
        onPublishTask={onPublishTask}
        task={task}
      />
    );

    const publish = screen.getByTestId("tasks-detail-preview-publish");
    expect(publish).toHaveTextContent("Publish");
    expect(publish).toHaveAttribute("title", expect.stringMatching(/coordinator handoff/i));
    fireEvent.click(publish);
    expect(onPublishTask).toHaveBeenCalledWith(task.id);

    expect(screen.getByTestId("tasks-detail-preview-lifecycle")).toHaveTextContent("Saved intent");
    expect(screen.getByTestId("tasks-detail-preview-lifecycle-hint")).toHaveTextContent(
      /saved intent/i
    );
  });

  it("labels the start-run button as the coordinator handoff boundary", () => {
    const task = buildTask({ status: "ready" });
    render(
      <TasksDetailPreviewPanel detail={buildDetail(task)} onEnqueueRun={() => {}} task={task} />
    );

    const button = screen.getByTestId("tasks-detail-preview-enqueue");
    expect(button).toHaveTextContent("Start run");
    expect(button).toHaveAttribute("title", expect.stringMatching(/coordinator handoff/i));
    expect(screen.getByTestId("tasks-detail-preview-lifecycle")).toHaveTextContent(
      "Ready to start"
    );
  });

  it("renders the coordination channel chip when the task has a coordinated active run", () => {
    const task = buildTask({
      status: "in_progress",
      active_run: {
        id: "run_42",
        task_id: "task_001",
        attempt: 1,
        status: "queued",
        queued_at: "2026-04-11T09:30:00Z",
        coordination_channel_id: "coord-task-001",
        coordination_channel: {
          id: "coord-task-001",
          display_name: "TASK-1 coordination",
          workspace_id: "ws_storybook",
          task_id: "task_001",
          run_id: "run_42",
          allowed_message_kinds: ["status", "request"],
        },
      } as TaskListItem["active_run"],
    });

    render(<TasksDetailPreviewPanel detail={buildDetail(task)} task={task} />);

    expect(screen.getByTestId("tasks-detail-preview-coordination")).toHaveTextContent(
      "Channel: TASK-1 coordination"
    );
    expect(screen.getByTestId("tasks-detail-preview-coordination")).toHaveAttribute(
      "title",
      expect.stringMatching(/channel messages support coordination only/i)
    );
    expect(screen.getByTestId("tasks-detail-preview-lifecycle")).toHaveTextContent(
      "Coordinator handoff"
    );
  });
});
