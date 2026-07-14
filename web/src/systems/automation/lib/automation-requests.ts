import { automationTargetMode, normalizeAutomationRetry } from "./automation-drafts";
import { localInputToDate, toRfc3339 } from "./cron-engine";
import type { CreateAutomationJobRequest, CreateAutomationTriggerRequest } from "../types";

export type AutomationEditorMode = "create" | "edit";

export interface AutomationRequestProjection<T> {
  method: "PATCH" | "POST";
  path: string;
  payload: T;
}

export function normalizeAutomationSchedule(
  schedule: CreateAutomationJobRequest["schedule"]
): CreateAutomationJobRequest["schedule"] {
  if (schedule.mode !== "at") {
    return schedule;
  }

  return {
    ...schedule,
    time: toRfc3339(localInputToDate(schedule.time ?? "")),
  };
}

/** Exact Job request normalization shared by displayed preview and route submission. */
export function buildAutomationJobRequest(
  draft: CreateAutomationJobRequest
): CreateAutomationJobRequest {
  return {
    ...draft,
    retry: normalizeAutomationRetry(draft.retry ?? undefined),
    schedule: normalizeAutomationSchedule(draft.schedule),
  };
}

/** Exact Trigger request normalization shared by displayed preview and route submission. */
export function buildAutomationTriggerRequest(
  draft: CreateAutomationTriggerRequest
): CreateAutomationTriggerRequest {
  const loopTarget =
    automationTargetMode(draft) === "loop" && draft.loop_target
      ? {
          ...draft.loop_target,
          workspace_id:
            draft.scope === "workspace"
              ? (draft.workspace_id ?? "")
              : draft.loop_target.workspace_id,
        }
      : draft.loop_target;

  return {
    ...draft,
    loop_target: loopTarget,
    retry: normalizeAutomationRetry(draft.retry ?? undefined),
  };
}

export function projectAutomationJobRequest(
  draft: CreateAutomationJobRequest,
  mode: AutomationEditorMode
): AutomationRequestProjection<CreateAutomationJobRequest> {
  return {
    method: mode === "create" ? "POST" : "PATCH",
    path: mode === "create" ? "/api/automation/jobs" : "/api/automation/jobs/{id}",
    payload: buildAutomationJobRequest(draft),
  };
}

export function projectAutomationTriggerRequest(
  draft: CreateAutomationTriggerRequest,
  mode: AutomationEditorMode
): AutomationRequestProjection<CreateAutomationTriggerRequest> {
  return {
    method: mode === "create" ? "POST" : "PATCH",
    path: mode === "create" ? "/api/automation/triggers" : "/api/automation/triggers/{id}",
    payload: buildAutomationTriggerRequest(draft),
  };
}
