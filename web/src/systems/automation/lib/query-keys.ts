import type {
  AutomationJobListFilter,
  AutomationRunHistoryFilter,
  AutomationRunListFilter,
  AutomationTriggerListFilter,
} from "../types";

function normalizeText(value?: string): string {
  return value ?? "";
}

function normalizeNumber(value?: number): string {
  return value == null ? "" : String(value);
}

export const automationKeys = {
  all: ["automation"] as const,

  jobs: () => [...automationKeys.all, "jobs"] as const,
  jobLists: () => [...automationKeys.jobs(), "list"] as const,
  jobList: (filters: AutomationJobListFilter = {}) =>
    [
      ...automationKeys.jobLists(),
      filters.scope ?? "",
      normalizeText(filters.workspace_id),
      filters.source ?? "",
      normalizeNumber(filters.limit),
    ] as const,
  jobDetails: () => [...automationKeys.jobs(), "detail"] as const,
  jobDetail: (id: string) => [...automationKeys.jobDetails(), id] as const,
  jobRunsRoot: () => [...automationKeys.jobs(), "runs"] as const,
  jobRuns: (id: string, filters: AutomationRunHistoryFilter = {}) =>
    [
      ...automationKeys.jobRunsRoot(),
      id,
      filters.status ?? "",
      normalizeText(filters.since),
      normalizeText(filters.until),
      normalizeNumber(filters.limit),
    ] as const,

  triggers: () => [...automationKeys.all, "triggers"] as const,
  triggerLists: () => [...automationKeys.triggers(), "list"] as const,
  triggerList: (filters: AutomationTriggerListFilter = {}) =>
    [
      ...automationKeys.triggerLists(),
      filters.scope ?? "",
      normalizeText(filters.workspace_id),
      filters.source ?? "",
      normalizeText(filters.event),
      normalizeNumber(filters.limit),
    ] as const,
  triggerDetails: () => [...automationKeys.triggers(), "detail"] as const,
  triggerDetail: (id: string) => [...automationKeys.triggerDetails(), id] as const,
  triggerRunsRoot: () => [...automationKeys.triggers(), "runs"] as const,
  triggerRuns: (id: string, filters: AutomationRunHistoryFilter = {}) =>
    [
      ...automationKeys.triggerRunsRoot(),
      id,
      filters.status ?? "",
      normalizeText(filters.since),
      normalizeText(filters.until),
      normalizeNumber(filters.limit),
    ] as const,

  runs: () => [...automationKeys.all, "runs"] as const,
  runLists: () => [...automationKeys.runs(), "list"] as const,
  runList: (filters: AutomationRunListFilter = {}) =>
    [
      ...automationKeys.runLists(),
      normalizeText(filters.job_id),
      normalizeText(filters.trigger_id),
      filters.status ?? "",
      normalizeText(filters.since),
      normalizeText(filters.until),
      normalizeNumber(filters.limit),
    ] as const,
};
