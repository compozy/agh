import { fireEvent, render, screen } from "@testing-library/react";
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

import { AutomationDetailPanel } from "../automation-detail-panel";

const jobFixture = {
  id: "job_daily_review",
  name: "daily-review",
  agent_name: "reviewer",
  prompt: "Review recent changes.",
  scope: "workspace" as const,
  workspace_id: "ws_alpha",
  source: "dynamic" as const,
  enabled: true,
  schedule: { mode: "cron" as const, expr: "0 9 * * *" },
  retry: { strategy: "none" as const, max_retries: 3, base_delay: "2s" },
  fire_limit: { max: 12, window: "1h" },
  next_run: "2026-04-12T09:00:00Z",
  scheduler: {
    job_id: "job_daily_review",
    registered: true,
    next_run_at: "2026-04-12T09:00:00Z",
    last_run_at: "2026-04-11T09:00:01Z",
    last_scheduled_at: "2026-04-11T09:00:00Z",
    last_fire_id: "fire_daily_review_001",
    catch_up_policy: "skip_missed" as const,
    misfire_grace_seconds: 0,
    misfire_count: 1,
    last_misfire_at: "2026-04-10T09:00:00Z",
    updated_at: "2026-04-11T09:00:01Z",
  },
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:05:00Z",
};

const triggerFixture = {
  id: "trg_push_review",
  name: "push-review",
  agent_name: "reviewer",
  prompt: "Review push event {{ .Data.branch }}.",
  event: "webhook",
  filter: { "data.branch": "main" },
  scope: "workspace" as const,
  workspace_id: "ws_alpha",
  source: "config" as const,
  enabled: false,
  retry: { strategy: "backoff" as const, max_retries: 4, base_delay: "5s" },
  fire_limit: { max: 12, window: "1h" },
  endpoint_slug: "push-review",
  webhook_id: "wbh_push_review",
  created_at: "2026-04-11T08:00:00Z",
  updated_at: "2026-04-11T08:10:00Z",
};

const runFixture = {
  id: "run_001",
  status: "completed" as const,
  attempt: 1,
  job_id: "job_daily_review",
  fire_id: "fire_daily_review_001",
  session_id: "sess_001",
  scheduled_at: "2026-04-11T09:00:00Z",
  started_at: "2026-04-11T10:00:00Z",
  ended_at: "2026-04-11T10:05:00Z",
};

function renderPanel(overrides: Partial<Parameters<typeof AutomationDetailPanel>[0]> = {}) {
  const onDelete = vi.fn();
  const onEdit = vi.fn();
  const onToggleEnabled = vi.fn();
  const onTriggerNow = vi.fn();

  render(
    <AutomationDetailPanel
      emptyState={null}
      error={null}
      state={{
        isDeleting: false,
        isLoading: false,
        isTogglePending: false,
        isTriggerPending: false,
        ...overrides.state,
      }}
      item={jobFixture}
      kind="jobs"
      onDelete={onDelete}
      onEdit={onEdit}
      onToggleEnabled={onToggleEnabled}
      onTriggerNow={onTriggerNow}
      runs={[runFixture]}
      runsError={null}
      runsLoading={false}
      {...overrides}
    />
  );

  return { onDelete, onEdit, onToggleEnabled, onTriggerNow };
}

describe("AutomationDetailPanel", () => {
  it("renders loading state", () => {
    renderPanel({
      state: {
        isDeleting: false,
        isLoading: true,
        isTogglePending: false,
        isTriggerPending: false,
      },
      item: undefined,
    });
    expect(screen.getByTestId("automation-detail-loading")).toBeInTheDocument();
  });

  it("renders error state", () => {
    renderPanel({ error: new Error("boom"), item: undefined });
    expect(screen.getByTestId("automation-detail-error")).toBeInTheDocument();
  });

  it("renders route-level empty state", () => {
    renderPanel({
      emptyState: {
        actionLabel: "Create Job",
        description: "Create the first job.",
        icon: "jobs",
        onAction: vi.fn(),
        title: "No jobs configured",
      },
      item: undefined,
    });
    expect(screen.getByTestId("automation-detail-empty")).toBeInTheDocument();
    expect(screen.getByText("No jobs configured")).toBeInTheDocument();
  });

  it("renders dynamic job details and dispatches action callbacks", () => {
    const { onDelete, onEdit, onToggleEnabled, onTriggerNow } = renderPanel();

    expect(screen.getByTestId("automation-detail-panel")).toBeInTheDocument();
    expect(screen.getByText("daily-review")).toBeInTheDocument();
    expect(screen.getByText("Review recent changes.")).toBeInTheDocument();
    expect(screen.getByTestId("automation-job-scheduler")).toHaveTextContent("skip_missed");
    expect(screen.getByTestId("automation-job-scheduler")).toHaveTextContent(
      "fire_daily_review_001"
    );
    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_001")).toHaveAttribute(
      "href",
      "/session/sess_001"
    );

    fireEvent.click(screen.getByTestId("toggle-automation-btn"));
    fireEvent.click(screen.getByTestId("edit-automation-btn"));
    fireEvent.click(screen.getByTestId("trigger-job-btn"));
    fireEvent.click(screen.getByTestId("delete-automation-btn"));

    expect(onToggleEnabled).toHaveBeenCalledWith(false);
    expect(onEdit).toHaveBeenCalledOnce();
    expect(onTriggerNow).toHaveBeenCalledOnce();
    expect(onDelete).toHaveBeenCalledOnce();
  });

  it("Should render the 24 px DetailHeader anatomy with the job name as H1", () => {
    renderPanel();

    const header = screen.getByTestId("automation-detail-header");
    expect(header).toHaveAttribute("data-slot", "detail-header");
    const heading = screen.getByRole("heading", { level: 1, name: "daily-review" });
    expect(heading.className).toContain("text-[length:var(--text-detail-h1)]");
  });

  it("renders manual jobs without implying a cron schedule", () => {
    renderPanel({
      item: {
        ...jobFixture,
        schedule: undefined,
      },
    });

    expect(screen.getByText("manual")).toBeInTheDocument();
    expect(screen.getAllByText("Manual")).toHaveLength(2);
    expect(screen.queryByText("Cron schedule")).not.toBeInTheDocument();
  });

  it("renders the job success-rate Metric with the computed percentage from the fixture runs", () => {
    renderPanel({
      item: jobFixture,
      runs: [
        runFixture,
        { ...runFixture, id: "run_002", status: "completed" as const },
        { ...runFixture, id: "run_003", status: "failed" as const },
      ],
    });

    const successRate = screen.getByTestId("automation-job-metric-success-rate");
    expect(successRate).toHaveTextContent("67%");
  });

  it("renders the trigger hook Section with a KindChip for the source", () => {
    renderPanel({
      item: { ...triggerFixture, source: "dynamic" as const, event: "ext.github.push" },
      kind: "triggers",
      runs: [],
    });

    const kindChip = document.querySelector('[data-slot="kind-chip"][data-kind="ext.github.push"]');
    expect(kindChip).not.toBeNull();
  });

  it("renders config trigger details without mutable actions", () => {
    renderPanel({
      item: triggerFixture,
      kind: "triggers",
      runs: [
        { ...runFixture, id: "run_trigger", trigger_id: "trg_push_review", job_id: undefined },
      ],
    });

    expect(
      screen.getByText(
        "This automation is defined in configuration files. Only the enabled state can be toggled from the UI."
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Webhook id")).toBeInTheDocument();
    expect(screen.getByText("wbh_push_review")).toBeInTheDocument();
    expect(screen.queryByTestId("edit-automation-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("delete-automation-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-job-btn")).not.toBeInTheDocument();
  });
});
