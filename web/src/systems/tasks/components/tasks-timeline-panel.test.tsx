import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksTimelinePanel } from "./tasks-timeline-panel";
import type { TaskTimelineItem } from "../types";

function buildItem(overrides: Partial<TaskTimelineItem> = {}): TaskTimelineItem {
  return {
    event_id: "evt_1",
    event_type: "task.run_started",
    sequence: 1,
    timestamp: "2026-04-11T14:37:45Z",
    actor: { kind: "daemon", ref: "daemon" },
    origin: { kind: "cli", ref: "op" },
    task: {
      id: "task_001",
      identifier: "TASK-42",
      status: "ready",
      scope: "workspace",
      title: "Summarize review feedback",
    },
    run: {
      id: "run_abc",
      attempt: 2,
      status: "running",
      queued_at: "2026-04-11T14:32:12Z",
      task_id: "task_001",
      max_attempts: 3,
    } as unknown as TaskTimelineItem["run"],
    ...overrides,
  } as TaskTimelineItem;
}

describe("TasksTimelinePanel", () => {
  it("renders loader while items load", () => {
    render(<TasksTimelinePanel isLoading items={[]} />);
    expect(screen.getByTestId("tasks-timeline-loading")).toBeInTheDocument();
  });

  it("renders the error state when loading fails", () => {
    render(<TasksTimelinePanel errorMessage="boom" items={[]} />);
    expect(screen.getByTestId("tasks-timeline-error")).toHaveTextContent("boom");
  });

  it("renders an empty state when no events exist", () => {
    render(<TasksTimelinePanel items={[]} />);
    expect(screen.getByTestId("tasks-timeline-empty")).toBeInTheDocument();
  });

  it("renders event messages with live indicator for in-progress runs", () => {
    render(
      <TasksTimelinePanel
        isLive
        items={[
          buildItem({
            event_id: "evt_progress",
            event_type: "task.run_progress",
            sequence: 7,
            payload: { message: "Calling tool github.post_review_comment" },
          }),
          buildItem({
            event_id: "evt_failed",
            event_type: "task.run_failed",
            sequence: 4,
            run: {
              ...buildItem().run,
              error: "rate-limited",
              status: "failed",
            } as unknown as TaskTimelineItem["run"],
          }),
        ]}
      />
    );

    expect(screen.getByTestId("tasks-timeline-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-timeline-event-type-evt_progress")).toHaveTextContent(
      "task.run_progress"
    );
    expect(screen.getByTestId("tasks-timeline-message-evt_progress")).toHaveTextContent(
      "Calling tool"
    );
    expect(screen.getByTestId("tasks-timeline-live-evt_progress")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-timeline-message-evt_failed")).toHaveTextContent(
      "rate-limited"
    );
  });

  it("fires onLoadMore when the cursor is saturated", () => {
    const onLoadMore = vi.fn();
    render(<TasksTimelinePanel canLoadMore items={[buildItem()]} onLoadMore={onLoadMore} />);

    fireEvent.click(screen.getByTestId("tasks-timeline-load-more"));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });
});
