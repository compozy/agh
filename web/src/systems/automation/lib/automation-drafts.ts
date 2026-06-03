import type {
  AutomationJob,
  AutomationRetry,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "../types";

const DEFAULT_RETRY_NONE = {
  strategy: "none" as const,
  max_retries: 0,
  base_delay: "",
};

const DEFAULT_RETRY_BACKOFF = {
  strategy: "backoff" as const,
  max_retries: 3,
  base_delay: "2s",
};

const DEFAULT_FIRE_LIMIT = {
  max: 12,
  window: "1h",
};

export function retryDraftForStrategy(
  strategy: AutomationRetry["strategy"],
  retry?: AutomationRetry
): AutomationRetry {
  if (strategy === "backoff") {
    return {
      strategy: "backoff",
      max_retries:
        retry?.strategy === "backoff" && retry.max_retries > 0
          ? retry.max_retries
          : DEFAULT_RETRY_BACKOFF.max_retries,
      base_delay:
        retry?.strategy === "backoff" && retry.base_delay.trim() !== ""
          ? retry.base_delay
          : DEFAULT_RETRY_BACKOFF.base_delay,
    };
  }

  return { ...DEFAULT_RETRY_NONE };
}

export function normalizeAutomationRetry(retry?: AutomationRetry): AutomationRetry {
  return retryDraftForStrategy(retry?.strategy ?? "none", retry);
}

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
    retry: normalizeAutomationRetry(),
    fire_limit: { ...DEFAULT_FIRE_LIMIT },
  };
}

export type JobOutputMode = "agent" | "task";

/** Derive the output mode from the draft: a task object means task mode. */
export function jobOutputMode(draft: CreateAutomationJobRequest): JobOutputMode {
  return draft.task ? "task" : "agent";
}

/** A blank task object for entering task mode. `network_channel` stays undefined. */
export function emptyJobTask(): NonNullable<CreateAutomationJobRequest["task"]> {
  return { title: "", description: "", owner: null };
}

/**
 * Switch the draft between agent and task output modes. The runtime requires
 * `retry.strategy === "none"` whenever a task is materialized, so task mode also
 * forces the retry policy to none.
 */
export function setJobOutputMode(
  draft: CreateAutomationJobRequest,
  mode: JobOutputMode
): CreateAutomationJobRequest {
  if (mode === "task") {
    return {
      ...draft,
      task: draft.task ?? emptyJobTask(),
      retry: retryDraftForStrategy("none"),
    };
  }
  return { ...draft, task: undefined };
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
    task: job.task ?? undefined,
    enabled: job.enabled,
    retry: normalizeAutomationRetry(job.retry),
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
    event: "session.stopped",
    filter: {},
    scope: workspaceId ? "workspace" : "global",
    workspace_id: workspaceId,
    enabled: true,
    retry: normalizeAutomationRetry(),
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
    retry: normalizeAutomationRetry(trigger.retry),
    fire_limit: {
      max: trigger.fire_limit.max,
      window: trigger.fire_limit.window,
    },
    endpoint_slug: trigger.endpoint_slug,
    webhook_id: trigger.webhook_id,
  };
}
