import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailOverviewPanel } from "../tasks-detail-overview-panel";
import type { TaskDetailView } from "../../types";

function buildDetail(overrides: Partial<TaskDetailView> = {}): TaskDetailView {
  return {
    task: {
      id: "task_001",
      identifier: "TASK-42",
      title: "Summarize review feedback",
      status: "in_progress",
      scope: "workspace",
      origin: { kind: "cli", ref: "op" },
      created_at: "2026-04-11T09:00:00Z",
      updated_at: "2026-04-11T09:00:00Z",
      created_by: { kind: "human", ref: "pedro@" },
      owner: { kind: "agent_session", ref: "Coder" },
      priority: "high",
      description: "Pull CodeRabbit review on PR 341 and post a summary.",
    },
    summary: {
      id: "task_001",
      title: "Summarize review feedback",
      status: "in_progress",
      scope: "workspace",
      origin: { kind: "cli", ref: "op" },
      created_at: "2026-04-11T09:00:00Z",
      updated_at: "2026-04-11T09:00:00Z",
      created_by: { kind: "human", ref: "pedro@" },
      active_run: {
        id: "run_active",
        attempt: 2,
        max_attempts: 3,
        status: "running",
        queued_at: "2026-04-11T09:00:00Z",
        started_at: "2026-04-11T09:00:30Z",
        task_id: "task_001",
        session_id: "sess_a",
      },
      child_count: 1,
      dependency_count: 2,
    },
    children: [
      {
        id: "child_001",
        identifier: "TASK-43",
        status: "ready",
        scope: "workspace",
        title: "Write migration",
        priority: "medium",
      } as never,
    ],
    dependency_references: [
      {
        task_id: "task_001",
        depends_on_task_id: "dep_001",
        kind: "blocks",
        created_at: "2026-04-11T09:00:00Z",
        depends_on: {
          id: "dep_001",
          identifier: "TASK-19",
          status: "completed",
          scope: "workspace",
          title: "Write tests",
        },
      } as never,
    ],
    runs: [{ id: "run_active" } as never, { id: "run_older" } as never],
    ...overrides,
  } as TaskDetailView;
}

describe("TasksDetailOverviewPanel", () => {
  it("renders counts, active run link, and description", () => {
    render(<TasksDetailOverviewPanel detail={buildDetail()} />);

    expect(screen.getByTestId("tasks-detail-overview")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-overview-children")).toHaveTextContent("1");
    expect(screen.getByTestId("tasks-detail-overview-dependencies")).toHaveTextContent("1");
    expect(screen.getByTestId("tasks-detail-overview-runs")).toHaveTextContent("2");
    expect(screen.getByTestId("tasks-detail-active-run")).toHaveTextContent("run_active");
    expect(screen.getByTestId("tasks-detail-active-run-link")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-description")).toHaveTextContent("Pull CodeRabbit");
  });

  it("renders empty description state when task has no description", () => {
    const detail = buildDetail();
    detail.task.description = "";
    render(<TasksDetailOverviewPanel detail={detail} />);
    expect(screen.getByTestId("tasks-detail-description")).toHaveTextContent(
      "No description provided."
    );
  });

  it("surfaces the coordination channel chip on coordinated active runs", () => {
    const detail = buildDetail();
    detail.summary!.active_run = {
      ...detail.summary!.active_run!,
      coordination_channel_id: "coord-task-001",
      coordination_channel: {
        id: "coord-task-001",
        display_name: "TASK-42 coordination",
        workspace_id: "ws_storybook",
        task_id: detail.task.id,
        run_id: detail.summary!.active_run!.id,
        allowed_message_kinds: ["status", "request"],
      },
    } as TaskDetailView["summary"]["active_run"];

    render(<TasksDetailOverviewPanel detail={detail} />);

    expect(screen.getByTestId("tasks-detail-active-run-channel")).toHaveTextContent(
      "Channel: TASK-42 coordination"
    );
    expect(screen.getByTestId("tasks-detail-active-run-channel")).toHaveAttribute(
      "title",
      expect.stringMatching(/channel messages support coordination only/i)
    );
  });

  it("renders an empty execution section with saved-intent hint when there is no active run", () => {
    const detail = buildDetail();
    detail.task.status = "draft";
    detail.task.draft = true;
    detail.summary!.active_run = null;

    render(<TasksDetailOverviewPanel detail={detail} />);

    expect(screen.queryByTestId("tasks-detail-active-run")).not.toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-active-run-empty")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-active-run-empty-hint")).toHaveTextContent(
      /saved intent/i
    );
  });
});
