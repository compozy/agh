import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { buildTaskRunDetailFixture } from "../../mocks/fixtures";
import { TaskRunRail } from "../task-run-rail";

type CostSummary = NonNullable<ReturnType<typeof buildTaskRunDetailFixture>["summary"]>;

function renderRail(cost: Partial<CostSummary>) {
  const base = buildTaskRunDetailFixture();
  const run = buildTaskRunDetailFixture({
    run: { ...base.run, status: "completed", ended_at: "2026-04-17T10:10:00Z" },
    summary: { ...base.summary, ...cost },
  });
  render(<TaskRunRail onInspect={vi.fn()} run={run} taskId="task_001" taskRuns={[]} />);
}

// Invariant: task-run cost presentation preserves the daemon's provenance instead of
// presenting estimates, included usage, or unknown values as measured spend.
describe("TaskRunRail cost provenance", () => {
  it("Should render actual cost as measured spend with its source", () => {
    renderRail({
      cost_status: "actual",
      cost_source: "agent_reported",
      total_cost: 0.18,
      cost_currency: "USD",
    });

    const cost = screen.getByTestId("task-run-detail-cost");
    expect(cost).toHaveTextContent("$0.180");
    expect(cost).toHaveTextContent("Reported by agent");
    expect(cost).not.toHaveTextContent("≈");
  });

  it("Should identify estimated cost with both the approximation cue and catalog source", () => {
    renderRail({
      cost_status: "estimated",
      cost_source: "catalog_config",
      total_cost: 0.18,
      cost_currency: "USD",
    });

    const cost = screen.getByTestId("task-run-detail-cost");
    expect(cost).toHaveTextContent("Est. cost");
    expect(cost).toHaveTextContent("≈ $0.180");
    expect(cost).toHaveTextContent("Estimated · Catalog rate");
  });

  it("Should render included and unavailable costs without a monetary amount", () => {
    const base = buildTaskRunDetailFixture();
    const { rerender } = render(
      <TaskRunRail
        onInspect={vi.fn()}
        run={buildTaskRunDetailFixture({
          run: { ...base.run, status: "completed", ended_at: "2026-04-17T10:10:00Z" },
          summary: {
            ...base.summary,
            cost_status: "included",
            cost_source: "none",
            total_cost: null,
          },
        })}
        taskId="task_001"
        taskRuns={[]}
      />
    );
    expect(screen.getByTestId("task-run-detail-cost")).toHaveTextContent("Included");

    rerender(
      <TaskRunRail
        onInspect={vi.fn()}
        run={buildTaskRunDetailFixture({
          run: { ...base.run, status: "completed", ended_at: "2026-04-17T10:10:00Z" },
          summary: { ...base.summary, cost_status: "unknown", total_cost: null },
        })}
        taskId="task_001"
        taskRuns={[]}
      />
    );
    const cost = screen.getByTestId("task-run-detail-cost");
    expect(cost).toHaveTextContent("Unavailable");
    expect(cost).not.toHaveTextContent("$");
  });

  it("Should omit the cost row when the daemon reports no cost provenance", () => {
    renderRail({ cost_status: undefined, cost_source: undefined, total_cost: 0.18 });
    expect(screen.queryByTestId("task-run-detail-cost")).not.toBeInTheDocument();
  });
});
