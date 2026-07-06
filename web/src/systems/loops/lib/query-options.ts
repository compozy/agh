import { queryOptions } from "@tanstack/react-query";

import {
  getLoop,
  getLoopAnnotations,
  getLoopConfig,
  getLoopRun,
  listLoopRuns,
  listLoops,
} from "../adapters/loops-api";
import { isTerminalLoopStatus } from "./loop-formatters";
import { loopsKeys } from "./query-keys";
import type { LoopRunsFilter } from "../types";

const DEFAULT_STALE_TIME = 15_000;
const DEFAULT_REFETCH_INTERVAL = 30_000;
const LIVE_STALE_TIME = 5_000;
const LIVE_REFETCH_INTERVAL = 15_000;

export function loopsCatalogOptions(workspaceId: string, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.catalog(workspaceId),
    queryFn: ({ signal }) => listLoops(workspaceId, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function loopDetailOptions(workspaceId: string, name: string, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.detail(workspaceId, name),
    queryFn: ({ signal }) => getLoop(workspaceId, name, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && Boolean(name) && enabled,
  });
}

export function loopConfigOptions(workspaceId: string, name: string, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.config(workspaceId, name),
    queryFn: ({ signal }) => getLoopConfig(workspaceId, name, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && Boolean(name) && enabled,
  });
}

export function loopAnnotationsOptions(workspaceId: string, name: string, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.annotations(workspaceId, name),
    queryFn: ({ signal }) => getLoopAnnotations(workspaceId, name, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && Boolean(name) && enabled,
  });
}

export function loopRunsOptions(workspaceId: string, filters: LoopRunsFilter = {}, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.runs(workspaceId, filters),
    queryFn: ({ signal }) => listLoopRuns(workspaceId, filters, signal),
    staleTime: LIVE_STALE_TIME,
    refetchInterval: LIVE_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function loopRunDetailOptions(workspaceId: string, runId: string, enabled = true) {
  return queryOptions({
    queryKey: loopsKeys.runDetail(workspaceId, runId),
    queryFn: ({ signal }) => getLoopRun(workspaceId, runId, signal),
    staleTime: LIVE_STALE_TIME,
    // Poll only while the run is live; a terminal run's projection is immutable, so
    // the run page stops refetching once it reaches a terminal state (contract-lane
    // risk, task-18 review) instead of polling a finished run forever.
    refetchInterval: query =>
      isTerminalLoopStatus(query.state.data?.run.status) ? false : LIVE_REFETCH_INTERVAL,
    enabled: Boolean(workspaceId) && Boolean(runId) && enabled,
  });
}
