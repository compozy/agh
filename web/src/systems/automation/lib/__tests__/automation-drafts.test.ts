import { describe, expect, it } from "vitest";

import {
  automationJobToDraft,
  automationTriggerToDraft,
  createAutomationJobDraft,
  createAutomationTriggerDraft,
  emptyJobTask,
  jobOutputMode,
  normalizeAutomationRetry,
  retryDraftForStrategy,
  setJobOutputMode,
} from "../automation-drafts";

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
  retry: { strategy: "backoff" as const, max_retries: 4, base_delay: "5s" },
  fire_limit: { max: 8, window: "30m" },
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
  source: "dynamic" as const,
  enabled: false,
  retry: { strategy: "none" as const, max_retries: 3, base_delay: "2s" },
  fire_limit: { max: 12, window: "1h" },
  endpoint_slug: "push-review",
  webhook_id: "wbh_push_review",
  webhook_secret_present: true,
  created_at: "2026-04-11T08:00:00Z",
  updated_at: "2026-04-11T08:10:00Z",
};

describe("automation draft helpers", () => {
  it("creates a workspace-scoped job draft when an active workspace exists", () => {
    expect(createAutomationJobDraft("ws_alpha")).toEqual({
      name: "",
      agent_name: "",
      prompt: "",
      schedule: { mode: "cron", expr: "0 9 * * *" },
      scope: "workspace",
      workspace_id: "ws_alpha",
      enabled: true,
      retry: { strategy: "none", max_retries: 0, base_delay: "" },
      fire_limit: { max: 12, window: "1h" },
    });
  });

  it("falls back to a global job draft without an active workspace", () => {
    expect(createAutomationJobDraft()).toMatchObject({
      scope: "global",
      workspace_id: undefined,
    });
  });

  it("maps an existing job back into an editable draft", () => {
    expect(automationJobToDraft(jobFixture)).toEqual({
      name: "daily-review",
      agent_name: "reviewer",
      prompt: "Review recent changes.",
      schedule: { mode: "cron", expr: "0 9 * * *" },
      scope: "workspace",
      workspace_id: "ws_alpha",
      enabled: true,
      retry: { strategy: "backoff", max_retries: 4, base_delay: "5s" },
      fire_limit: { max: 8, window: "30m" },
    });
  });

  it("creates trigger drafts with the expected defaults", () => {
    expect(createAutomationTriggerDraft()).toMatchObject({
      event: "session.stopped",
      filter: {},
      scope: "global",
      workspace_id: undefined,
      enabled: true,
      retry: { strategy: "none", max_retries: 0, base_delay: "" },
    });
    expect(createAutomationTriggerDraft("ws_alpha")).toMatchObject({
      scope: "workspace",
      workspace_id: "ws_alpha",
      retry: { strategy: "none", max_retries: 0, base_delay: "" },
    });
  });

  it("maps an existing trigger back into an editable draft", () => {
    expect(automationTriggerToDraft(triggerFixture)).toEqual({
      name: "push-review",
      agent_name: "reviewer",
      prompt: "Review push event {{ .Data.branch }}.",
      event: "webhook",
      filter: { "data.branch": "main" },
      scope: "workspace",
      workspace_id: "ws_alpha",
      enabled: false,
      retry: { strategy: "none", max_retries: 0, base_delay: "" },
      fire_limit: { max: 12, window: "1h" },
      endpoint_slug: "push-review",
      webhook_id: "wbh_push_review",
    });
  });

  it("normalizes retry payloads for none and backoff strategies", () => {
    expect(normalizeAutomationRetry()).toEqual({
      strategy: "none",
      max_retries: 0,
      base_delay: "",
    });
    expect(
      normalizeAutomationRetry({ strategy: "none", max_retries: 9, base_delay: "7s" })
    ).toEqual({
      strategy: "none",
      max_retries: 0,
      base_delay: "",
    });
    expect(retryDraftForStrategy("backoff")).toEqual({
      strategy: "backoff",
      max_retries: 3,
      base_delay: "2s",
    });
  });
});

describe("job output mode helpers", () => {
  it("Should derive the output mode from the presence of a task object", () => {
    const agentDraft = createAutomationJobDraft("ws_alpha");
    expect(jobOutputMode(agentDraft)).toBe("agent");
    expect(jobOutputMode({ ...agentDraft, task: emptyJobTask() })).toBe("task");
  });

  it("Should produce a blank unassigned task for entering task mode", () => {
    expect(emptyJobTask()).toEqual({ title: "", description: "", owner: null });
  });

  it("Should force retry to none when switching to task mode (Go requires it)", () => {
    const backoffDraft: ReturnType<typeof createAutomationJobDraft> = {
      ...createAutomationJobDraft("ws_alpha"),
      retry: { strategy: "backoff", max_retries: 5, base_delay: "5s" },
    };
    const taskDraft = setJobOutputMode(backoffDraft, "task");
    expect(taskDraft.task).toEqual({ title: "", description: "", owner: null });
    expect(taskDraft.retry).toEqual({ strategy: "none", max_retries: 0, base_delay: "" });
  });

  it("Should preserve an existing task object when re-entering task mode", () => {
    const existing = { title: "review", description: "do it", owner: null };
    const draft = { ...createAutomationJobDraft("ws_alpha"), task: existing };
    expect(setJobOutputMode(draft, "task").task).toBe(existing);
  });

  it("Should clear the task object when switching back to agent mode", () => {
    const draft = { ...createAutomationJobDraft("ws_alpha"), task: emptyJobTask() };
    expect(setJobOutputMode(draft, "agent").task).toBeUndefined();
  });

  it("Should round-trip a task-delegating job back into a task-mode draft", () => {
    const taskJob = {
      ...jobFixture,
      task: { title: "delegated-review", description: "review changes", owner: null },
    };
    const draft = automationJobToDraft(taskJob);
    expect(jobOutputMode(draft)).toBe("task");
    expect(draft.task).toEqual({
      title: "delegated-review",
      description: "review changes",
      owner: null,
    });
  });
});
