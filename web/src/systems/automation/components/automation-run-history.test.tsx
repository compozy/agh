import { render, screen } from "@testing-library/react";
import type { AnchorHTMLAttributes } from "react";
import { describe, expect, it } from "vitest";

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

import { vi } from "vitest";

import { AutomationRunHistory } from "./automation-run-history";
import type { AutomationRun } from "../types";

const completedRun: AutomationRun = {
  id: "run_001",
  status: "completed",
  attempt: 1,
  job_id: "job_daily_review",
  session_id: "sess_001",
  started_at: "2026-04-11T10:00:00Z",
  ended_at: "2026-04-11T10:05:00Z",
};

const failedRun: AutomationRun = {
  id: "run_002",
  status: "failed",
  attempt: 2,
  job_id: "job_daily_review",
  started_at: "2026-04-11T11:00:00Z",
  ended_at: "2026-04-11T11:02:00Z",
  error: "timeout",
};

describe("AutomationRunHistory", () => {
  it("renders the empty state and never mounts the table when no runs are provided", () => {
    render(<AutomationRunHistory error={null} isLoading={false} runs={[]} />);

    expect(screen.getByTestId("automation-run-history-empty")).toBeInTheDocument();
    expect(screen.getByText("No runs recorded yet")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("renders the loading fallback instead of the table while a run query is pending", () => {
    render(<AutomationRunHistory error={null} isLoading runs={[]} />);

    expect(screen.getByTestId("automation-run-history-loading")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("renders the error state with the thrown message", () => {
    render(
      <AutomationRunHistory error={new Error("Failed to fetch runs")} isLoading={false} runs={[]} />
    );

    expect(screen.getByTestId("automation-run-history-error")).toBeInTheDocument();
    expect(screen.getByText("Failed to fetch runs")).toBeInTheDocument();
  });

  it("renders a table row per run with status, attempt, duration, and a session link", () => {
    render(
      <AutomationRunHistory error={null} isLoading={false} runs={[completedRun, failedRun]} />
    );

    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_002")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-session-link-run_001")).toHaveAttribute(
      "href",
      "/session/sess_001"
    );
    expect(screen.getByText("timeout")).toBeInTheDocument();
  });
});
