import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AgentStatsGrid } from "../agent-stats-grid";

describe("AgentStatsGrid", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders exact counted metrics instead of aggregates from the loaded page", () => {
    vi.spyOn(Date, "now").mockReturnValue(Date.parse("2026-07-11T12:05:00Z"));

    render(
      <AgentStatsGrid total={205} active={7} resumable={13} lastActivityAt="2026-07-11T12:00:00Z" />
    );

    expect(within(screen.getByTestId("agent-stat-active")).getByText("7")).toBeInTheDocument();
    expect(within(screen.getByTestId("agent-stat-total")).getByText("205")).toBeInTheDocument();
    expect(within(screen.getByTestId("agent-stat-resumable")).getByText("13")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("agent-stat-last-activity")).getByText("5m ago")
    ).toBeInTheDocument();
    expect(screen.queryByText("Total runtime")).not.toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
  });
});
