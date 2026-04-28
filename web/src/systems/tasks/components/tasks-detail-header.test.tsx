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
import type { TaskDetailView, TaskListItem } from "../types";

function buildDetail(
  overrides: Partial<TaskDetailView["task"]> = {},
  summaryOverrides: Partial<TaskDetailView["summary"]> = {}
): TaskDetailView {
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

  return {
    task,
    summary: {
      ...(task as unknown as TaskDetailView["summary"]),
      ...summaryOverrides,
    } as TaskDetailView["summary"],
  } as TaskDetailView;
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
    const dot = container.querySelector('[data-slot="pill-dot"]');
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

  it("renders the saved-intent lifecycle pill and hint for draft tasks without implying autonomy", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "draft", draft: true })}
        onPublish={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Saved intent");
    expect(screen.getByTestId("tasks-detail-lifecycle-hint")).toHaveTextContent(/saved intent/i);
    expect(screen.getByTestId("tasks-detail-publish")).toHaveTextContent("Publish");
    expect(screen.getByTestId("tasks-detail-publish")).toHaveAttribute(
      "title",
      expect.stringMatching(/coordinator handoff/i)
    );
    expect(screen.queryByTestId("tasks-detail-coordination")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
  });

  it("labels the start-run button as the coordinator handoff boundary", () => {
    render(<TasksDetailHeader detail={buildDetail({ status: "ready" })} onEnqueueRun={() => {}} />);

    const button = screen.getByTestId("tasks-detail-enqueue");
    expect(button).toHaveTextContent("Start run");
    expect(button).toHaveAttribute("title", expect.stringMatching(/coordinator handoff/i));
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Ready to start");
    expect(screen.getByTestId("tasks-detail-lifecycle-hint")).toHaveTextContent(
      /start enqueues a coordinator-handoff run/i
    );
  });

  it("surfaces the coordination channel chip when the active run is bound to a channel", () => {
    const activeRun = {
      id: "run_42",
      task_id: "task_001",
      attempt: 1,
      status: "queued",
      queued_at: "2026-04-11T09:30:00Z",
      coordination_channel_id: "coord-task-001",
      coordination_channel: {
        id: "coord-task-001",
        display_name: "TASK-42 coordination",
      },
    } as TaskListItem["active_run"];
    const detail = buildDetail({ status: "in_progress" }, { active_run: activeRun });

    render(<TasksDetailHeader detail={detail} onEnqueueRun={() => {}} />);

    expect(screen.getByTestId("tasks-detail-coordination")).toHaveTextContent(
      "Channel: TASK-42 coordination"
    );
    expect(screen.getByTestId("tasks-detail-coordination")).toHaveAttribute(
      "title",
      expect.stringMatching(/channel messages support coordination only/i)
    );
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Coordinator handoff");
  });
});
