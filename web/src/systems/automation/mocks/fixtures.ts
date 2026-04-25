import type { AutomationJob, AutomationRun, AutomationTrigger } from "../types";

export const automationJobFixtures: AutomationJob[] = [
  {
    id: "job_daily_review",
    name: "daily-review",
    agent_name: "reviewer",
    prompt: "Review recent changes.",
    scope: "workspace",
    workspace_id: "ws_storybook",
    source: "dynamic",
    enabled: true,
    schedule: {
      mode: "cron",
      expr: "0 9 * * *",
    },
    retry: {
      strategy: "none",
      max_retries: 0,
      base_delay: "",
    },
    fire_limit: {
      max: 12,
      window: "1h",
    },
    next_run: "2026-04-18T09:00:00Z",
    scheduler: {
      job_id: "job_daily_review",
      registered: true,
      next_run_at: "2026-04-18T09:00:00Z",
      last_run_at: "2026-04-17T09:00:01Z",
      last_scheduled_at: "2026-04-17T09:00:00Z",
      last_fire_id: "fire_daily_review_001",
      catch_up_policy: "skip_missed",
      misfire_grace_seconds: 0,
      misfire_count: 1,
      last_misfire_at: "2026-04-16T09:00:00Z",
      updated_at: "2026-04-17T09:00:01Z",
    },
    created_at: "2026-04-17T09:00:00Z",
    updated_at: "2026-04-17T11:00:00Z",
  },
  {
    id: "job_release_notes",
    name: "release-notes",
    agent_name: "writer",
    prompt: "Summarize the latest Storybook rollout changes.",
    scope: "global",
    source: "dynamic",
    enabled: false,
    retry: {
      strategy: "backoff",
      max_retries: 3,
      base_delay: "5s",
    },
    fire_limit: {
      max: 5,
      window: "6h",
    },
    created_at: "2026-04-16T12:00:00Z",
    updated_at: "2026-04-17T10:00:00Z",
  },
];

export const automationTriggerFixtures: AutomationTrigger[] = [
  {
    id: "trg_push_review",
    name: "push-review",
    agent_name: "reviewer",
    prompt: "Review push event {{ .Data.branch }}.",
    event: "ext.github.push",
    filter: {
      "data.branch": "main",
    },
    scope: "workspace",
    workspace_id: "ws_storybook",
    source: "dynamic",
    enabled: true,
    retry: {
      strategy: "backoff",
      max_retries: 4,
      base_delay: "5s",
    },
    fire_limit: {
      max: 12,
      window: "1h",
    },
    endpoint_slug: "push-review",
    webhook_id: "wbh_push_review",
    created_at: "2026-04-17T08:00:00Z",
    updated_at: "2026-04-17T08:10:00Z",
  },
];

export const automationRunFixtures: AutomationRun[] = [
  {
    id: "run_daily_review_001",
    status: "completed",
    attempt: 1,
    job_id: "job_daily_review",
    fire_id: "fire_daily_review_001",
    session_id: "sess-storybook",
    scheduled_at: "2026-04-17T09:00:00Z",
    started_at: "2026-04-17T09:00:00Z",
    ended_at: "2026-04-17T09:04:00Z",
  },
  {
    id: "run_push_review_002",
    status: "running",
    attempt: 1,
    trigger_id: "trg_push_review",
    session_id: "sess-reviewer",
    started_at: "2026-04-17T10:00:00Z",
  },
];

export const primaryAutomationJobFixture: AutomationJob = automationJobFixtures[0];
export const primaryAutomationTriggerFixture: AutomationTrigger = automationTriggerFixtures[0];
