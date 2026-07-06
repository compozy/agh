import { useQuery } from "@tanstack/react-query";

import {
  loopAnnotationsOptions,
  loopConfigOptions,
  loopDetailOptions,
  loopRunDetailOptions,
  loopRunsOptions,
  loopsCatalogOptions,
} from "../lib/query-options";
import type { LoopRunsFilter } from "../types";

export function useLoops(workspaceId: string, enabled = true) {
  return useQuery(loopsCatalogOptions(workspaceId, enabled));
}

export function useLoop(workspaceId: string, name: string, enabled = true) {
  return useQuery(loopDetailOptions(workspaceId, name, enabled));
}

export function useLoopConfig(workspaceId: string, name: string, enabled = true) {
  return useQuery(loopConfigOptions(workspaceId, name, enabled));
}

export function useLoopAnnotations(workspaceId: string, name: string, enabled = true) {
  return useQuery(loopAnnotationsOptions(workspaceId, name, enabled));
}

export function useLoopRuns(workspaceId: string, filters: LoopRunsFilter = {}, enabled = true) {
  return useQuery(loopRunsOptions(workspaceId, filters, enabled));
}

export function useLoopRun(workspaceId: string, runId: string, enabled = true) {
  return useQuery(loopRunDetailOptions(workspaceId, runId, enabled));
}
