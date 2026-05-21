import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  schedulerBacklogFixture,
  schedulerPausedStatusFixture,
  schedulerStatusFixture,
} from "../../mocks";
import { SchedulerControlsPanel } from "../scheduler-controls-panel";

describe("SchedulerControlsPanel", () => {
  it("renders scheduler status and backlog pressure", () => {
    render(
      <SchedulerControlsPanel backlog={schedulerBacklogFixture} status={schedulerStatusFixture} />
    );

    expect(screen.getByTestId("scheduler-controls-state")).toHaveTextContent("Running");
    expect(screen.getByTestId("scheduler-controls-meta")).toHaveTextContent("1 active claims");
    expect(screen.getByTestId("scheduler-backlog-total")).toHaveTextContent("2");
    expect(screen.getByTestId("scheduler-backlog-row-run_014")).toBeInTheDocument();
  });

  it("requires a reason before pausing scheduler dispatch", async () => {
    const onPause = vi.fn().mockResolvedValue(undefined);
    render(
      <SchedulerControlsPanel
        backlog={schedulerBacklogFixture}
        onPause={onPause}
        status={schedulerStatusFixture}
      />
    );

    fireEvent.click(screen.getByTestId("scheduler-controls-pause"));
    fireEvent.click(screen.getByTestId("scheduler-controls-pause-confirm"));
    expect(screen.getByTestId("scheduler-controls-pause-error")).toHaveTextContent(
      "Provide a pause reason."
    );

    fireEvent.change(screen.getByTestId("scheduler-controls-pause-reason"), {
      target: { value: "provider incident" },
    });
    fireEvent.click(screen.getByTestId("scheduler-controls-pause-confirm"));
    await waitFor(() => {
      expect(onPause).toHaveBeenCalledWith("provider incident");
    });
  });

  it("renders resume action while paused and forwards drain", () => {
    const onResume = vi.fn();
    const onDrain = vi.fn();
    render(
      <SchedulerControlsPanel
        backlog={schedulerBacklogFixture}
        onDrain={onDrain}
        onResume={onResume}
        status={schedulerPausedStatusFixture}
      />
    );

    fireEvent.click(screen.getByTestId("scheduler-controls-resume"));
    fireEvent.click(screen.getByTestId("scheduler-controls-drain"));

    expect(onResume).toHaveBeenCalledTimes(1);
    expect(onDrain).toHaveBeenCalledTimes(1);
  });
});
