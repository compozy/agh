import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AutomationListPanel } from "../automation-list-panel";

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
  event: "ext.github.push",
  filter: { "data.branch": "main" },
  scope: "workspace" as const,
  workspace_id: "ws_alpha",
  source: "dynamic" as const,
  enabled: true,
  retry: { strategy: "backoff" as const, max_retries: 4, base_delay: "5s" },
  fire_limit: { max: 12, window: "1h" },
  endpoint_slug: "push-review",
  webhook_id: "wbh_push_review",
  webhook_secret_present: false,
  created_at: "2026-04-11T08:00:00Z",
  updated_at: "2026-04-11T08:10:00Z",
};

describe("AutomationListPanel", () => {
  it("renders job items and highlights the selected record", () => {
    render(
      <AutomationListPanel
        activeWorkspaceName="alpha"
        jobs={[jobFixture]}
        kind="jobs"
        onSearchChange={vi.fn()}
        onSelect={vi.fn()}
        scopeFilter="workspace"
        searchQuery=""
        selectedId="job_daily_review"
        totalCount={1}
        triggers={[]}
      />
    );

    expect(screen.getByTestId("automation-item-job_daily_review")).toBeInTheDocument();
    expect(screen.getByTestId("automation-active-indicator")).toBeInTheDocument();
    expect(screen.getByTestId("automation-list-summary")).toHaveTextContent("1 job in alpha");
  });

  it("renders the loading fallback when isLoading=true and the list is empty", () => {
    render(
      <AutomationListPanel
        activeWorkspaceName="alpha"
        isLoading
        jobs={[]}
        kind="jobs"
        onSearchChange={vi.fn()}
        onSelect={vi.fn()}
        scopeFilter="all"
        searchQuery=""
        selectedId={null}
        totalCount={0}
        triggers={[]}
      />
    );

    expect(screen.getByTestId("automation-list-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("automation-list-empty")).not.toBeInTheDocument();
  });

  it("renders the error fallback when an errorMessage is provided and the list is empty", () => {
    render(
      <AutomationListPanel
        activeWorkspaceName="alpha"
        errorMessage="boom"
        jobs={[]}
        kind="jobs"
        onSearchChange={vi.fn()}
        onSelect={vi.fn()}
        scopeFilter="all"
        searchQuery=""
        selectedId={null}
        totalCount={0}
        triggers={[]}
      />
    );

    expect(screen.getByTestId("automation-list-error")).toHaveTextContent("boom");
  });

  it("filters trigger items from the search box", async () => {
    const user = userEvent.setup();
    let currentQuery = "";

    const onSearchChange = vi.fn((nextValue: string) => {
      currentQuery = nextValue;
      rerenderPanel();
    });

    const onSelect = vi.fn();

    const { rerender } = render(
      <AutomationListPanel
        activeWorkspaceName="alpha"
        jobs={[]}
        kind="triggers"
        onSearchChange={onSearchChange}
        onSelect={onSelect}
        scopeFilter="all"
        searchQuery={currentQuery}
        selectedId={null}
        totalCount={1}
        triggers={[triggerFixture]}
      />
    );

    function rerenderPanel() {
      rerender(
        <AutomationListPanel
          activeWorkspaceName="alpha"
          jobs={[]}
          kind="triggers"
          onSearchChange={onSearchChange}
          onSelect={onSelect}
          scopeFilter="all"
          searchQuery={currentQuery}
          selectedId={null}
          totalCount={1}
          triggers={currentQuery === "webhook-only" ? [] : [triggerFixture]}
        />
      );
    }

    await user.type(screen.getByTestId("automation-search-input"), "push");

    expect(onSearchChange).toHaveBeenCalled();
    expect(screen.getByTestId("automation-item-trg_push_review")).toBeInTheDocument();

    currentQuery = "webhook-only";
    rerenderPanel();

    expect(screen.getByTestId("automation-list-empty")).toBeInTheDocument();
  });
});
