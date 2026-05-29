import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TaskRunTimelinePanel } from "../task-run-timeline-panel";
import type { TaskRunDetailView, TaskTimelineItem } from "../../types";

function buildRun(overrides: Partial<TaskRunDetailView["run"]> = {}): TaskRunDetailView {
  return {
    run: {
      id: "run_001",
      task_id: "task_001",
      attempt: 2,
      status: "running",
      queued_at: "2026-04-11T14:30:00Z",
      started_at: "2026-04-11T14:37:45Z",
      origin: { kind: "cli", ref: "op" },
      session_id: "sess_jf8d21",
      ...overrides,
    },
    task: {
      id: "task_001",
      identifier: "TASK-42",
      status: "ready",
      scope: "workspace",
      title: "Summarize feedback",
    },
    summary: {},
    session: {
      session_id: "sess_jf8d21",
      created_at: "2026-04-11T14:30:00Z",
      updated_at: "2026-04-11T14:40:45Z",
      agent_name: "Coder",
    },
  } as unknown as TaskRunDetailView;
}

const eventA: TaskTimelineItem = {
  event_id: "evt_001",
  sequence: 12,
  event_type: "task.run_started",
  timestamp: "2026-04-11T14:37:45Z",
  payload: undefined,
  run: { id: "run_001", attempt: 2, status: "running" },
  origin: { kind: "cli", ref: "op" },
  task: { id: "task_001", identifier: "TASK-42" },
} as unknown as TaskTimelineItem;

const eventB: TaskTimelineItem = {
  event_id: "evt_002",
  sequence: 13,
  event_type: "task.run_progress",
  timestamp: "2026-04-11T14:40:45Z",
  payload: { message: "Halfway through" },
  run: { id: "run_001", attempt: 2, status: "running" },
  origin: { kind: "cli", ref: "op" },
  task: { id: "task_001", identifier: "TASK-42" },
} as unknown as TaskTimelineItem;

const eventOtherRun: TaskTimelineItem = {
  event_id: "evt_other",
  sequence: 9,
  event_type: "task.run_completed",
  timestamp: "2026-04-10T10:00:00Z",
  payload: undefined,
  run: { id: "run_999", attempt: 1, status: "completed" },
  origin: { kind: "cli", ref: "op" },
  task: { id: "task_001", identifier: "TASK-42" },
} as unknown as TaskTimelineItem;

const eventNeedsAttention: TaskTimelineItem = {
  event_id: "evt_needs_attention",
  sequence: 14,
  event_type: "task.run_needs_attention",
  timestamp: "2026-04-11T14:45:45Z",
  payload: { diagnostic: "No capable agent claimed this run before escalation." },
  run: { id: "run_001", attempt: 2, status: "needs_attention" },
  origin: { kind: "scheduler", ref: "starvation" },
  task: { id: "task_001", identifier: "TASK-42" },
} as unknown as TaskTimelineItem;

const eventRecoveredFromAttention: TaskTimelineItem = {
  event_id: "evt_recovered_from_attention",
  sequence: 15,
  event_type: "task.run_recovered_from_attention",
  timestamp: "2026-04-11T14:46:45Z",
  payload: { reason: "operator confirmed the dependency is now available" },
  run: { id: "run_001", attempt: 3, status: "queued" },
  origin: { kind: "cli", ref: "op" },
  task: { id: "task_001", identifier: "TASK-42" },
} as unknown as TaskTimelineItem;

describe("TaskRunTimelinePanel", () => {
  it("Should render the RunCard with the run id", () => {
    render(<TaskRunTimelinePanel items={[eventA, eventB]} run={buildRun()} />);
    expect(screen.getByTestId("tasks-run-detail-card")).toBeInTheDocument();
    const card = screen.getByTestId("tasks-run-detail-card");
    expect(card.textContent ?? "").toContain("run_001");
  });

  it("Should filter timeline events to the current run only", () => {
    render(<TaskRunTimelinePanel items={[eventA, eventB, eventOtherRun]} run={buildRun()} />);
    expect(screen.getByTestId("tasks-run-detail-timeline-item-evt_001")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-run-detail-timeline-item-evt_002")).toBeInTheDocument();
    expect(
      screen.queryByTestId("tasks-run-detail-timeline-item-evt_other")
    ).not.toBeInTheDocument();
  });

  it("Should render the empty state when no events match the run", () => {
    render(<TaskRunTimelinePanel items={[eventOtherRun]} run={buildRun()} />);
    expect(screen.getByTestId("tasks-run-detail-timeline-empty")).toBeInTheDocument();
  });

  it("Should render the loading state when no events have arrived yet", () => {
    render(<TaskRunTimelinePanel isLoading items={[]} run={buildRun()} />);
    expect(screen.getByTestId("tasks-run-detail-timeline-loading")).toBeInTheDocument();
  });

  it("Should describe needs_attention timeline diagnostics", () => {
    render(
      <TaskRunTimelinePanel
        items={[eventNeedsAttention]}
        run={buildRun({ status: "needs_attention" })}
      />
    );
    expect(
      screen.getByText("No capable agent claimed this run before escalation.")
    ).toBeInTheDocument();
  });

  it("Should describe recovered-from-attention timeline reasons", () => {
    render(<TaskRunTimelinePanel items={[eventRecoveredFromAttention]} run={buildRun()} />);
    expect(
      screen.getByText("operator confirmed the dependency is now available")
    ).toBeInTheDocument();
  });

  it("Should NOT render the old MetadataList Identity panel", () => {
    render(<TaskRunTimelinePanel items={[eventA, eventB]} run={buildRun()} />);
    expect(screen.queryByTestId("task-run-detail-identity")).not.toBeInTheDocument();
    expect(screen.queryByTestId("task-run-detail-identity-run")).not.toBeInTheDocument();
    expect(screen.queryByTestId("task-run-detail-identity-attempt")).not.toBeInTheDocument();
  });

  it("Should surface run errors as a danger warning strip on the card", () => {
    render(
      <TaskRunTimelinePanel
        items={[]}
        run={buildRun({ status: "failed", error: "partner export 429" })}
      />
    );
    const warning = document.querySelector("[data-slot='run-card-warning']");
    expect(warning).not.toBeNull();
    expect(warning?.textContent ?? "").toContain("partner export 429");
    expect(warning).toHaveAttribute("data-tone", "danger");
  });

  it("Should surface needs_attention diagnostics as a warning strip on the card", () => {
    render(
      <TaskRunTimelinePanel
        items={[]}
        run={buildRun({
          status: "needs_attention",
          error: "No capable agent claimed this run before escalation.",
        })}
      />
    );
    const warning = document.querySelector("[data-slot='run-card-warning']");
    expect(warning).not.toBeNull();
    expect(warning?.textContent ?? "").toContain("No capable agent claimed this run");
    expect(warning).toHaveAttribute("data-tone", "warning");
  });
});
