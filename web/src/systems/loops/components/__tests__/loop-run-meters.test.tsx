import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LoopRunMeters } from "../run-page/loop-run-meters";
import { buildRunMeters } from "../../lib/loop-run-meters";
import { loopRunDetailByRunId } from "../../mocks/fixtures";
import type { LoopRunRecord } from "../../types";

const base = loopRunDetailByRunId.get("looprun_running")!.run;

function run(overrides: Partial<LoopRunRecord>): LoopRunRecord {
  return { ...base, ...overrides };
}

describe("LoopRunMeters", () => {
  it("Should render all 5 meters", () => {
    render(<LoopRunMeters meters={buildRunMeters(base, 3)} />);
    for (const key of ["attempts", "tokens", "wall", "cost", "breadth"]) {
      expect(screen.getByTestId(`loop-run-meter-${key}`)).toBeInTheDocument();
    }
  });

  it("Should warn-tint a budget meter only near the ceiling", () => {
    const { rerender } = render(
      <LoopRunMeters
        meters={buildRunMeters(run({ tokens_used: 300_000, budget_tokens: 1_000_000 }))}
      />
    );
    expect(screen.getByTestId("loop-run-meter-tokens")).toHaveAttribute("data-tone", "neutral");
    rerender(
      <LoopRunMeters
        meters={buildRunMeters(run({ tokens_used: 950_000, budget_tokens: 1_000_000 }))}
      />
    );
    expect(screen.getByTestId("loop-run-meter-tokens")).toHaveAttribute("data-tone", "warn");
    expect(screen.getByTestId("loop-run-meter-bar-tokens")).toBeInTheDocument();
  });

  it("Should keep the cost meter display-only with no bar and no cap control", () => {
    render(<LoopRunMeters meters={buildRunMeters(base, 3)} />);
    const cost = screen.getByTestId("loop-run-meter-cost");
    expect(cost).toHaveAttribute("data-display-only", "true");
    expect(cost).toHaveAttribute("data-tone", "neutral");
    expect(within(cost).queryByRole("spinbutton")).not.toBeInTheDocument();
    expect(within(cost).queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.queryByTestId("loop-run-meter-bar-cost")).not.toBeInTheDocument();
  });
});
