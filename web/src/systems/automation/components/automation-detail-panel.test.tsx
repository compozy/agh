import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AutomationDetailPanel } from "./automation-detail-panel";
import { createAutomationJobDraft, createAutomationTriggerDraft } from "../lib/automation-drafts";

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
  session_id: "sess_001",
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
      activeWorkspaceId="ws_alpha"
      editor={null}
      error={null}
      isDeleting={false}
      isLoading={false}
      isTogglePending={false}
      isTriggerPending={false}
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
  it("renders loading, error, and empty states", () => {
    const loading = renderPanel({ isLoading: true, item: undefined });
    expect(screen.getByTestId("automation-detail-loading")).toBeInTheDocument();

    loading.onDelete.mockReset();

    renderPanel({ error: new Error("boom"), item: undefined });
    expect(screen.getByTestId("automation-detail-error")).toBeInTheDocument();

    renderPanel({ item: undefined });
    expect(screen.getByTestId("automation-detail-empty")).toBeInTheDocument();
  });

  it("renders editor variants for jobs and triggers", () => {
    const jobEditor = {
      kind: "jobs" as const,
      mode: "create" as const,
      draft: createAutomationJobDraft("ws_alpha"),
      isPending: false,
      onCancel: vi.fn(),
      onChange: vi.fn(),
      onSubmit: vi.fn(),
    };
    renderPanel({ editor: jobEditor, item: undefined });
    expect(screen.getByTestId("automation-job-form")).toBeInTheDocument();

    const triggerEditor = {
      kind: "triggers" as const,
      mode: "edit" as const,
      draft: createAutomationTriggerDraft("ws_alpha"),
      isPending: false,
      onCancel: vi.fn(),
      onChange: vi.fn(),
      onSubmit: vi.fn(),
    };
    renderPanel({ editor: triggerEditor, item: undefined });
    expect(screen.getByTestId("automation-trigger-form")).toBeInTheDocument();
  });

  it("renders dynamic job details and dispatches action callbacks", () => {
    const { onDelete, onEdit, onToggleEnabled, onTriggerNow } = renderPanel();

    expect(screen.getByTestId("automation-detail-panel")).toBeInTheDocument();
    expect(screen.getByText("daily-review")).toBeInTheDocument();
    expect(screen.getByText("Review recent changes.")).toBeInTheDocument();
    expect(screen.getByTestId("automation-run-run_001")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("toggle-automation-btn"));
    fireEvent.click(screen.getByTestId("edit-automation-btn"));
    fireEvent.click(screen.getByTestId("trigger-job-btn"));
    fireEvent.click(screen.getByTestId("delete-automation-btn"));

    expect(onToggleEnabled).toHaveBeenCalledWith(false);
    expect(onEdit).toHaveBeenCalledOnce();
    expect(onTriggerNow).toHaveBeenCalledOnce();
    expect(onDelete).toHaveBeenCalledOnce();
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
        "Config-sourced automation can only toggle enabled state from the UI. Definition changes stay in configuration files."
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Webhook id")).toBeInTheDocument();
    expect(screen.getByText("wbh_push_review")).toBeInTheDocument();
    expect(screen.queryByTestId("edit-automation-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("delete-automation-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("trigger-job-btn")).not.toBeInTheDocument();
  });
});
