import { useQuery } from "@tanstack/react-query";

import {
  automationJobDetailOptions,
  automationJobRunsOptions,
  automationJobsListOptions,
  automationRunsListOptions,
  automationTriggerDetailOptions,
  automationTriggerRunsOptions,
  automationTriggersListOptions,
} from "../lib/query-options";
import type {
  AutomationJobListFilter,
  AutomationRunHistoryFilter,
  AutomationRunListFilter,
  AutomationTriggerListFilter,
} from "../types";

interface QueryHookOptions {
  enabled?: boolean;
}

export function useAutomationJobs(filters: AutomationJobListFilter = {}) {
  return useQuery(automationJobsListOptions(filters));
}

export function useAutomationJob(id: string, options: QueryHookOptions = {}) {
  return useQuery(automationJobDetailOptions(id, options.enabled ?? true));
}

export function useAutomationJobRuns(
  id: string,
  filters: AutomationRunHistoryFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery(automationJobRunsOptions(id, filters, options.enabled ?? true));
}

export function useAutomationTriggers(filters: AutomationTriggerListFilter = {}) {
  return useQuery(automationTriggersListOptions(filters));
}

export function useAutomationTrigger(id: string, options: QueryHookOptions = {}) {
  return useQuery(automationTriggerDetailOptions(id, options.enabled ?? true));
}

export function useAutomationTriggerRuns(
  id: string,
  filters: AutomationRunHistoryFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery(automationTriggerRunsOptions(id, filters, options.enabled ?? true));
}

export function useAutomationRuns(
  filters: AutomationRunListFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery(automationRunsListOptions(filters, options.enabled ?? true));
}
