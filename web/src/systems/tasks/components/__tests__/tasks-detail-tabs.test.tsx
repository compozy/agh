import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksDetailTabs } from "../tasks-detail-tabs";

describe("TasksDetailTabs", () => {
  it("renders all tabs with counts and live indicator", () => {
    render(
      <TasksDetailTabs
        active="timeline"
        items={[
          { id: "overview", label: "Overview" },
          { id: "runs", label: "Runs", count: 3 },
          { id: "timeline", label: "Events", live: true },
          { id: "children", label: "Children", count: 2 },
          { id: "dependencies", label: "Dependencies", count: 1 },
        ]}
        onChange={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-tabs")).toBeInTheDocument();
    expect(within(screen.getByTestId("tasks-detail-tab-runs")).getByText("3")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("tasks-detail-tab-timeline")).getByText("Live")
    ).toHaveAttribute("aria-live", "polite");
    expect(screen.getByTestId("tasks-detail-tab-children")).toHaveAttribute(
      "aria-selected",
      "false"
    );
    expect(screen.getByTestId("tasks-detail-tab-timeline")).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  it("invokes onChange when a tab is clicked", () => {
    const onChange = vi.fn();
    render(
      <TasksDetailTabs
        active="overview"
        items={[
          { id: "overview", label: "Overview" },
          { id: "runs", label: "Runs", count: 3 },
        ]}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByTestId("tasks-detail-tab-runs"));
    expect(onChange).toHaveBeenCalledWith("runs");
  });
});
