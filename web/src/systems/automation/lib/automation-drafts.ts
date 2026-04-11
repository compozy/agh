import type {
  AutomationJob,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "../types";

const DEFAULT_RETRY = {
  strategy: "none" as const,
  max_retries: 3,
  base_delay: "2s",
};

const DEFAULT_FIRE_LIMIT = {
  max: 12,
  window: "1h",
};

export function createAutomationJobDraft(
  activeWorkspaceId?: string | null
): CreateAutomationJobRequest {
  const workspaceId = activeWorkspaceId ?? undefined;

  return {
    name: "",
    agent_name: "",
    prompt: "",
    schedule: {
      mode: "cron",
      expr: "0 9 * * *",
    },
    scope: workspaceId ? "workspace" : "global",
    workspace_id: workspaceId,
    enabled: true,
    retry: { ...DEFAULT_RETRY },
    fire_limit: { ...DEFAULT_FIRE_LIMIT },
  };
}

export function automationJobToDraft(job: AutomationJob): CreateAutomationJobRequest {
  return {
    name: job.name,
    agent_name: job.agent_name,
    prompt: job.prompt,
    schedule: job.schedule ?? {
      mode: "cron",
      expr: "0 9 * * *",
    },
    scope: job.scope,
    workspace_id: job.workspace_id,
    enabled: job.enabled,
    retry: {
      strategy: job.retry.strategy,
      max_retries: job.retry.max_retries,
      base_delay: job.retry.base_delay,
    },
    fire_limit: {
      max: job.fire_limit.max,
      window: job.fire_limit.window,
    },
  };
}

export function createAutomationTriggerDraft(
  activeWorkspaceId?: string | null
): CreateAutomationTriggerRequest {
  const workspaceId = activeWorkspaceId ?? undefined;

  return {
    name: "",
    agent_name: "",
    prompt: "",
    event: "webhook",
    filter: {},
    scope: workspaceId ? "workspace" : "global",
    workspace_id: workspaceId,
    enabled: true,
    retry: { ...DEFAULT_RETRY },
    fire_limit: { ...DEFAULT_FIRE_LIMIT },
  };
}

export function automationTriggerToDraft(
  trigger: AutomationTrigger
): CreateAutomationTriggerRequest {
  return {
    name: trigger.name,
    agent_name: trigger.agent_name,
    prompt: trigger.prompt,
    event: trigger.event,
    filter: trigger.filter ?? {},
    scope: trigger.scope,
    workspace_id: trigger.workspace_id,
    enabled: trigger.enabled,
    retry: {
      strategy: trigger.retry.strategy,
      max_retries: trigger.retry.max_retries,
      base_delay: trigger.retry.base_delay,
    },
    fire_limit: {
      max: trigger.fire_limit.max,
      window: trigger.fire_limit.window,
    },
    endpoint_slug: trigger.endpoint_slug,
    webhook_id: trigger.webhook_id,
  };
}
