import { render, screen, within } from "@testing-library/react";
import type { AnchorHTMLAttributes } from "react";
import { describe, expect, it, vi } from "vitest";

interface MockLinkParams {
  id?: string;
}

interface MockLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  params?: MockLinkParams;
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, params, ...props }: MockLinkProps) => (
    <a href={`/session/${params?.id ?? ""}`} {...props}>
      {children}
    </a>
  ),
}));

import { AutomationRunHistory } from "../automation-run-history";
import type { AutomationRun } from "../../types";

const completedRun: AutomationRun = {
  id: "run_001",
  status: "completed",
  attempt: 1,
  job_id: "job_daily_review",
  fire_id: "fire_daily_review_001",
  session_id: "sess_001",
  scheduled_at: "2026-04-11T09:00:00Z",
  started_at: "2026-04-11T10:00:00Z",
  ended_at: "2026-04-11T10:05:00Z",
};

const failedRun: AutomationRun = {
  id: "run_002",
  status: "failed",
  attempt: 2,
  job_id: "job_daily_review",
  session_id: "sess_002",
  started_at: "2026-04-11T11:00:00Z",
  ended_at: "2026-04-11T11:02:00Z",
  error: "timeout",
  delivery_error: "dispatcher unavailable",
};

const pendingRun: AutomationRun = {
  id: "run_003",
  status: "scheduled",
  attempt: 1,
  job_id: "job_daily_review",
  scheduled_at: "2026-04-11T12:00:00Z",
};

describe("AutomationRunHistory", () => {
  it("Should render the empty state when no runs are provided", () => {
    render(<AutomationRunHistory error={null} isLoading={false} runs={[]} />);

    expect(screen.getByTestId("automation-run-history-empty")).toBeInTheDocument();
    expect(screen.getByText("No runs recorded yet")).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });

  it("Should render the loading fallback while a run query is pending", () => {
    render(<AutomationRunHistory error={null} isLoading runs={[]} />);

    expect(screen.getByTestId("automation-run-history-loading")).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });

  it("Should render the error state with the thrown message", () => {
    render(
      <AutomationRunHistory error={new Error("Failed to fetch runs")} isLoading={false} runs={[]} />
    );

    expect(screen.getByTestId("automation-run-history-error")).toBeInTheDocument();
    expect(screen.getByText("Failed to fetch runs")).toBeInTheDocument();
  });

  it("Should render every run row as a whole-row Link with a trailing ChevronRight when a session is present", () => {
    render(
      <AutomationRunHistory error={null} isLoading={false} runs={[completedRun, failedRun]} />
    );

    const completedRow = screen.getByTestId("automation-run-run_001");
    expect(completedRow.tagName).toBe("A");
    expect(completedRow).toHaveAttribute("href", "/session/sess_001");
    const trailingChevron = completedRow.querySelector("svg");
    expect(trailingChevron).not.toBeNull();
    expect(trailingChevron).toHaveAttribute("aria-hidden", "true");

    const failedRow = screen.getByTestId("automation-run-run_002");
    expect(failedRow.tagName).toBe("A");
    expect(within(failedRow).getByText("timeout")).toBeInTheDocument();
    expect(within(failedRow).getByText("Delivery: dispatcher unavailable")).toBeInTheDocument();
    expect(within(completedRow).getByText("fire_daily_review_001")).toBeInTheDocument();
    expect(screen.getByText(/scheduled Apr 11, 2026/)).toBeInTheDocument();
  });

  it("Should render rows without a session as static rows that surface a pending hint", () => {
    render(<AutomationRunHistory error={null} isLoading={false} runs={[pendingRun]} />);

    const row = screen.getByTestId("automation-run-run_003");
    expect(row.tagName).toBe("DIV");
    expect(within(row).getByText("pending")).toBeInTheDocument();
  });
});
