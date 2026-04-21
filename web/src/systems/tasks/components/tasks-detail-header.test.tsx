import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailHeader } from "./tasks-detail-header";
import type { TaskDetailView } from "../types";

function buildDetail(overrides: Partial<TaskDetailView["task"]> = {}): TaskDetailView {
  const task = {
    id: "task_001",
    identifier: "TASK-42",
    title: "Summarize review feedback",
    status: "ready",
    scope: "workspace",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    ...overrides,
  } as TaskDetailView["task"];

  return { task, summary: task as unknown as TaskDetailView["summary"] } as TaskDetailView;
}

describe("TasksDetailHeader", () => {
  it("renders PageHeader with title, MonoBadge id, status pill, and action slot in DOM order", () => {
    const { container } = render(<TasksDetailHeader detail={buildDetail()} />);

    expect(screen.getByTestId("tasks-detail-title")).toHaveTextContent("Summarize review feedback");
    expect(screen.getByTestId("tasks-detail-id")).toHaveTextContent("TASK-42");
    expect(screen.getByTestId("tasks-detail-status")).toHaveTextContent("Ready");
    expect(screen.getByTestId("tasks-detail-actions")).toBeInTheDocument();

    // Breadcrumb surfaces the short identifier
    expect(screen.getByTestId("tasks-detail-breadcrumb")).toHaveTextContent("TASK-42");

    // Meta row contains priority pill + created-by
    expect(screen.getByTestId("tasks-detail-meta")).toHaveTextContent("High");
    expect(screen.getByTestId("tasks-detail-meta")).toHaveTextContent("pedro@");

    // Status dot rendered alongside title
    const dot = container.querySelector('[data-slot="status-dot"]');
    expect(dot).not.toBeNull();
  });

  it("fires cancel, publish, and enqueue callbacks", () => {
    const onCancel = vi.fn();
    const onEnqueueRun = vi.fn();

    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "in_progress" })}
        onCancel={onCancel}
        onEnqueueRun={onEnqueueRun}
      />
    );

    fireEvent.click(screen.getByTestId("tasks-detail-cancel"));
    fireEvent.click(screen.getByTestId("tasks-detail-enqueue"));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onEnqueueRun).toHaveBeenCalledTimes(1);
  });

  it("opens a confirmation dialog before deleting the task", () => {
    const onDelete = vi.fn();

    render(<TasksDetailHeader detail={buildDetail()} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId("tasks-detail-delete"));
    expect(screen.getByTestId("tasks-detail-delete-dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-detail-delete-confirm"));
    expect(onDelete).toHaveBeenCalledWith("task_001");
  });

  it("surfaces the publish button for draft tasks", () => {
    const onPublish = vi.fn();
    render(<TasksDetailHeader detail={buildDetail({ status: "draft" })} onPublish={onPublish} />);

    fireEvent.click(screen.getByTestId("tasks-detail-publish"));
    expect(onPublish).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
  });

  it("disables destructive actions while mutations are pending", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "in_progress" })}
        isCancelPending
        isDeletePending
        isEnqueuePending
        onDelete={() => {}}
        onCancel={() => {}}
        onEnqueueRun={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-delete")).toBeDisabled();
    expect(screen.getByTestId("tasks-detail-cancel")).toBeDisabled();
    expect(screen.getByTestId("tasks-detail-enqueue")).toBeDisabled();
  });
});
