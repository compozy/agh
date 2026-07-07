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

export function useAutomationJobs(
  filters: AutomationJobListFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery({ ...automationJobsListOptions(filters), enabled: options.enabled ?? true });
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

export function useAutomationTriggers(
  filters: AutomationTriggerListFilter = {},
  options: QueryHookOptions = {}
) {
  return useQuery({ ...automationTriggersListOptions(filters), enabled: options.enabled ?? true });
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
