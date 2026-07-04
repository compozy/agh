import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksDetailBlockedReasons } from "../tasks-detail-blocked-reasons";
import type { TaskBlockedReason } from "../../types";

describe("TasksDetailBlockedReasons", () => {
  it("Should render exactly one chip per blocked_reasons entry, in order, with source + kind", () => {
    const reasons: TaskBlockedReason[] = [
      { source: "dependency", depends_on_task_ids: ["task_dep"] },
      { source: "approval", reason: "Pending review" },
      { source: "block", kind: "transient", reason: "External API down", block_id: "block_1" },
    ];

    render(<TasksDetailBlockedReasons reasons={reasons} />);

    const chips = screen.getAllByTestId("tasks-detail-blocked-reason");
    expect(chips).toHaveLength(3);
    expect(chips.map(chip => chip.getAttribute("data-source"))).toEqual([
      "dependency",
      "approval",
      "block",
    ]);
    expect(chips[2]).toHaveTextContent("Block · Transient");
    expect(chips[2]).toHaveTextContent("External API down");
    expect(chips[0]).toHaveTextContent("Waiting on task_dep");
  });

  it("Should render nothing when the task carries no blocking causes", () => {
    const { container } = render(<TasksDetailBlockedReasons reasons={[]} />);
    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByTestId("tasks-detail-blocked-reasons")).not.toBeInTheDocument();
  });
});
