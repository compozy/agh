import { queryOptions } from "@tanstack/react-query";

import {
  getTask,
  getTaskDashboard,
  getTaskInbox,
  getTaskRun,
  getTaskTimeline,
  getTaskTree,
  listTaskRuns,
  listTasks,
} from "../adapters/tasks-api";
import { tasksKeys } from "./query-keys";
import type {
  TaskDashboardFilter,
  TaskInboxFilter,
  TaskListFilter,
  TaskRunsFilter,
  TaskTimelineFilter,
} from "../types";

const DEFAULT_STALE_TIME = 15_000;
const DEFAULT_REFETCH_INTERVAL = 30_000;
const LIVE_STALE_TIME = 5_000;
const LIVE_REFETCH_INTERVAL = 15_000;

export function tasksListOptions(filters: TaskListFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.list(filters),
    queryFn: ({ signal }) => listTasks(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled,
  });
}

export function taskDetailOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.detail(id),
    queryFn: ({ signal }) => getTask(id, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function taskRunsOptions(id: string, filters: TaskRunsFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.runs(id, filters),
    queryFn: ({ signal }) => listTaskRuns(id, filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: LIVE_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function taskTimelineOptions(id: string, filters: TaskTimelineFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.timeline(id, filters),
    queryFn: ({ signal }) => getTaskTimeline(id, filters, signal),
    staleTime: LIVE_STALE_TIME,
    refetchInterval: LIVE_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function taskTreeOptions(id: string, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.tree(id),
    queryFn: ({ signal }) => getTaskTree(id, signal),
    staleTime: LIVE_STALE_TIME,
    refetchInterval: LIVE_REFETCH_INTERVAL,
    enabled: Boolean(id) && enabled,
  });
}

export function taskRunDetailOptions(runId: string, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.runDetail(runId),
    queryFn: ({ signal }) => getTaskRun(runId, signal),
    staleTime: LIVE_STALE_TIME,
    refetchInterval: LIVE_REFETCH_INTERVAL,
    enabled: Boolean(runId) && enabled,
  });
}

export function taskDashboardOptions(filters: TaskDashboardFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.dashboard(filters),
    queryFn: ({ signal }) => getTaskDashboard(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled,
  });
}

export function taskInboxOptions(filters: TaskInboxFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: tasksKeys.inbox(filters),
    queryFn: ({ signal }) => getTaskInbox(filters, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled,
  });
}
