import { queryOptions } from "@tanstack/react-query";

import {
  getAutomationJob,
  getAutomationTrigger,
  listAutomationJobRuns,
  listAutomationJobs,
  listAutomationRuns,
  listAutomationTriggerRuns,
  listAutomationTriggers,
} from "../adapters/automation-api";
import { automationKeys } from "./query-keys";
import type {
  AutomationJobListFilter,
  AutomationRunHistoryFilter,
  AutomationRunListFilter,
  AutomationTriggerListFilter,
} from "../types";

const DEFAULT_STALE_TIME = 15_000;
const DEFAULT_REFETCH_INTERVAL = 30_000;
const RUNS_REFETCH_INTERVAL = 15_000;

export function automationJobsListOptions(filters: AutomationJobListFilter = {}) {
  return queryOptions({
    queryKey: automationKeys.jobList(filters),
    queryFn: ({ signal }) => listAutomationJobs(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
  });
}

export function automationJobDetailOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: automationKeys.jobDetail(id),
    queryFn: ({ signal }) => getAutomationJob(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function automationJobRunsOptions(
  id: string,
  filters: AutomationRunHistoryFilter = {},
  enabled = true
) {
  return queryOptions({
    queryKey: automationKeys.jobRuns(id, filters),
    queryFn: ({ signal }) => listAutomationJobRuns(id, filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: RUNS_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function automationTriggersListOptions(filters: AutomationTriggerListFilter = {}) {
  return queryOptions({
    queryKey: automationKeys.triggerList(filters),
    queryFn: ({ signal }) => listAutomationTriggers(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
  });
}

export function automationTriggerDetailOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: automationKeys.triggerDetail(id),
    queryFn: ({ signal }) => getAutomationTrigger(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function automationTriggerRunsOptions(
  id: string,
  filters: AutomationRunHistoryFilter = {},
  enabled = true
) {
  return queryOptions({
    queryKey: automationKeys.triggerRuns(id, filters),
    queryFn: ({ signal }) => listAutomationTriggerRuns(id, filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: RUNS_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function automationRunsListOptions(filters: AutomationRunListFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: automationKeys.runList(filters),
    queryFn: ({ signal }) => listAutomationRuns(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: RUNS_REFETCH_INTERVAL,
    enabled,
  });
}
