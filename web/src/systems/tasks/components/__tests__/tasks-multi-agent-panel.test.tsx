import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import type { MultiAgentAgent } from "@/hooks/routes/use-task-detail-page";

import { TasksMultiAgentPanel } from "../tasks-multi-agent-panel";
import type { TaskTimelineItem, TaskTreeNode } from "../../types";

type TaskTreeNodeFixtureOverrides = Omit<Partial<TaskTreeNode>, "task"> & {
  task?: Partial<TaskTreeNode["task"]>;
};
type TaskTimelineItemFixtureOverrides = Omit<Partial<TaskTimelineItem>, "task"> & {
  task?: Partial<TaskTimelineItem["task"]>;
};

function buildNode(overrides: TaskTreeNodeFixtureOverrides = {}): TaskTreeNode {
  const { task, ...nodeOverrides } = overrides;
  return {
    depth: 0,
    task: {
      id: "task_001",
      identifier: "TASK-38",
      latest_event_seq: 1,
      status: "in_progress",
      scope: "workspace",
      title: "Triage new crash reports",
      owner: { kind: "agent_session", ref: "Researcher" },
      ...task,
    },
    active_run: {
      id: "run_a1b2",
      attempt: 1,
      max_attempts: 3,
      queued_at: "2026-04-17T10:00:00Z",
      status: "running",
      task_id: "task_001",
      session_id: "sess_a",
    },
    child_count: 2,
    last_activity_at: "2026-04-17T10:01:00Z",
    ...nodeOverrides,
  } as TaskTreeNode;
}

function buildAgent(overrides: Partial<MultiAgentAgent> = {}): MultiAgentAgent {
  const node = overrides.node ?? buildNode();
  return {
    node,
    isRoot: true,
    isPrimary: true,
    isLive: true,
    label: "Researcher",
    ...overrides,
  };
}

function buildTimelineItem(overrides: TaskTimelineItemFixtureOverrides = {}): TaskTimelineItem {
  const { task, ...itemOverrides } = overrides;
  return {
    event_id: "evt_1",
    sequence: 42,
    event_type: "task.run_progress",
    timestamp: "2026-04-17T10:00:30Z",
    task: { id: "task_001", identifier: "TASK-38", latest_event_seq: 42, ...task },
    run: {
      id: "run_a1b2",
      attempt: 1,
      status: "running",
    },
    ...itemOverrides,
  } as TaskTimelineItem;
}

describe("TasksMultiAgentPanel", () => {
  it("renders a loading placeholder when tree state is loading", () => {
    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[]}
        descendantCount={0}
        liveCount={0}
        state="loading"
        timeline={[]}
      />
    );
    expect(screen.getByTestId("tasks-multi-agent-loading")).toBeInTheDocument();
  });

  it("renders a disconnected fallback with the error message", () => {
    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[]}
        descendantCount={0}
        errorMessage="Stream disconnected"
        liveCount={0}
        state="disconnected"
        timeline={[]}
      />
    );
    expect(screen.getByTestId("tasks-multi-agent-disconnected")).toHaveTextContent(
      "Stream disconnected"
    );
  });

  it("renders an empty state when the task has no descendants", () => {
    const root = buildAgent({
      isLive: false,
      node: buildNode({ active_run: null, child_count: 0 }),
    });
    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root]}
        descendantCount={0}
        liveCount={0}
        state="no-descendants"
        timeline={[]}
      />
    );
    expect(screen.getByTestId("tasks-multi-agent-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("tasks-multi-agent-agents")).not.toBeInTheDocument();
  });

  it("renders a no-active banner alongside agents when no agents are live", () => {
    const root = buildAgent({ isLive: false });
    const child = buildAgent({
      isRoot: false,
      isPrimary: false,
      isLive: false,
      label: "Coder",
      node: buildNode({
        depth: 1,
        parent_task_id: "task_001",
        task: {
          id: "task_002",
          identifier: "TASK-39",
          status: "ready",
          scope: "workspace",
          title: "Child",
          owner: { kind: "agent_session", ref: "Coder" },
        },
        active_run: null,
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root, child]}
        descendantCount={1}
        liveCount={0}
        state="no-active"
        timeline={[]}
      />
    );

    expect(screen.getByTestId("tasks-multi-agent-no-active")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-multi-agent-agent-task_001")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-multi-agent-agent-task_002")).toBeInTheDocument();
  });

  it("shows the Agents header with a running/idle summary, not a live pill", () => {
    const root = buildAgent();
    const child = buildAgent({
      isRoot: false,
      isPrimary: false,
      isLive: false,
      label: "Coder",
      node: buildNode({
        depth: 1,
        parent_task_id: "task_001",
        task: {
          id: "task_002",
          identifier: "TASK-39",
          status: "ready",
          scope: "workspace",
          title: "Reproduce top-3 crashes",
          owner: { kind: "agent_session", ref: "Coder" },
        },
        active_run: null,
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root, child]}
        descendantCount={1}
        liveCount={1}
        state="ready"
        timeline={[]}
      />
    );

    expect(screen.getByTestId("tasks-multi-agent-header")).toHaveTextContent("Agents");
    expect(screen.getByTestId("tasks-multi-agent-summary")).toHaveTextContent("1 running · 1 idle");
    // The old top-right "N AGENTS LIVE" pill has been deleted — hierarchy
    // lives in the subtitle now.
    expect(screen.queryByTestId("tasks-multi-agent-live-count")).not.toBeInTheDocument();
  });

  it("flags live agents with a 2px accent left-rail (no perimeter border)", () => {
    const root = buildAgent();
    const child = buildAgent({
      isRoot: false,
      isPrimary: false,
      isLive: false,
      label: "Coder",
      node: buildNode({
        depth: 1,
        parent_task_id: "task_001",
        task: {
          id: "task_002",
          identifier: "TASK-39",
          status: "ready",
          scope: "workspace",
          title: "Reproduce",
          owner: { kind: "agent_session", ref: "Coder" },
        },
        active_run: null,
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root, child]}
        descendantCount={1}
        liveCount={1}
        state="ready"
        timeline={[]}
      />
    );

    const liveCard = screen.getByTestId("tasks-multi-agent-agent-task_001");
    expect(liveCard).toHaveAttribute("data-is-live", "true");
    expect(liveCard.className).toContain("border-l-2");
    expect(liveCard.className).toContain("border-l-accent");

    const idleCard = screen.getByTestId("tasks-multi-agent-agent-task_002");
    expect(idleCard).toHaveAttribute("data-is-live", "false");
    expect(idleCard.className).toContain("border-l-transparent");
  });

  it("renders the per-agent event strip when timeline data is available", () => {
    const root = buildAgent();
    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root]}
        descendantCount={0}
        liveCount={1}
        state="ready"
        timeline={[buildTimelineItem()]}
      />
    );
    expect(screen.getByTestId("tasks-multi-agent-agent-events-task_001")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-multi-agent-agent-event-evt_1")).toHaveTextContent(
      "task.run_progress"
    );
  });

  it("renders session and run drill-down links for agents with an active run", () => {
    const root = buildAgent();
    const child = buildAgent({
      isRoot: false,
      isPrimary: false,
      isLive: true,
      label: "Coder",
      node: buildNode({
        depth: 1,
        parent_task_id: "task_001",
        task: {
          id: "task_002",
          identifier: "TASK-39",
          status: "in_progress",
          scope: "workspace",
          title: "Reproduce",
          owner: { kind: "agent_session", ref: "Coder" },
        },
        active_run: {
          id: "run_c3d4",
          attempt: 1,
          max_attempts: 2,
          queued_at: "2026-04-17T10:00:10Z",
          status: "running",
          task_id: "task_002",
          session_id: "sess_b",
        },
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={1}
        agents={[root, child]}
        descendantCount={1}
        liveCount={2}
        state="ready"
        timeline={[]}
      />
    );

    const sessionLink = screen.getByTestId("tasks-multi-agent-agent-session-task_002");
    expect(sessionLink).toHaveTextContent("Open session");

    const runLink = screen.getByTestId("tasks-multi-agent-agent-run-task_002");
    expect(runLink).toHaveTextContent("Open run");

    const taskLink = screen.getByTestId("tasks-multi-agent-agent-task-task_002");
    expect(taskLink).toHaveTextContent("Open task");
  });

  it("does not render a session link for agents without an active run", () => {
    const root = buildAgent();
    const child = buildAgent({
      isRoot: false,
      isPrimary: false,
      isLive: false,
      label: "Writer",
      node: buildNode({
        depth: 1,
        parent_task_id: "task_001",
        task: {
          id: "task_003",
          identifier: "TASK-40",
          status: "ready",
          scope: "workspace",
          title: "Idle",
          owner: { kind: "agent_session", ref: "Writer" },
        },
        active_run: null,
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root, child]}
        descendantCount={1}
        liveCount={1}
        state="ready"
        timeline={[]}
      />
    );

    expect(
      screen.queryByTestId("tasks-multi-agent-agent-session-task_003")
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-multi-agent-agent-run-task_003")).not.toBeInTheDocument();
    expect(screen.getByTestId("tasks-multi-agent-agent-task-task_003")).toBeInTheDocument();
  });

  it("renders a failure summary for agents with a run error", () => {
    const root = buildAgent({
      isLive: false,
      node: buildNode({
        active_run: {
          id: "run_failed",
          attempt: 2,
          max_attempts: 3,
          queued_at: "2026-04-17T10:00:00Z",
          started_at: "2026-04-17T10:00:10Z",
          ended_at: "2026-04-17T10:00:30Z",
          status: "failed",
          task_id: "task_001",
          error: "tool execution failed",
        },
      }),
    });

    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root]}
        descendantCount={0}
        liveCount={0}
        state="no-active"
        timeline={[]}
      />
    );

    expect(screen.getByTestId("tasks-multi-agent-agent-error-task_001")).toHaveTextContent(
      "tool execution failed"
    );
  });

  it("does not render the deprecated interleaved-timeline section", () => {
    const root = buildAgent();
    render(
      <TasksMultiAgentPanel
        activeDescendants={0}
        agents={[root]}
        descendantCount={0}
        liveCount={1}
        state="ready"
        timeline={[]}
      />
    );
    expect(screen.queryByTestId("tasks-multi-agent-timeline")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-multi-agent-timeline-live")).not.toBeInTheDocument();
  });
});
