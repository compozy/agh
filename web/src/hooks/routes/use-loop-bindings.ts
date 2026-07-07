import { useMemo } from "react";

import { useAutomationJobs, useAutomationTriggers } from "@/systems/automation";
import type { LoopBindingKind, LoopBindingRow } from "@/systems/loops";

import { buildLoopBindingIndex } from "./loop-bindings-map";

/** Upper bound on automations scanned to build the catalog badge index. */
const BINDING_SCAN_LIMIT = 200;

const EMPTY_ROWS: readonly LoopBindingRow[] = [];

export interface LoopBindingIndexResult {
  /** Attached-automation kinds per loop name, for the catalog binding badge. */
  byLoop: Map<string, LoopBindingKind[]>;
  isLoading: boolean;
}

/**
 * Scans the workspace's triggers + jobs and indexes which Loops have at least one
 * attached loop-target automation (catalog binding badge, §9.14). Automation-off
 * or errored queries degrade to an empty index (no badges), never a thrown route.
 */
export function useLoopBindingIndex(workspaceId: string): LoopBindingIndexResult {
  const enabled = workspaceId !== "";
  const triggersQuery = useAutomationTriggers({ limit: BINDING_SCAN_LIMIT }, { enabled });
  const jobsQuery = useAutomationJobs({ limit: BINDING_SCAN_LIMIT }, { enabled });
  const triggers = triggersQuery.data ?? [];
  const jobs = jobsQuery.data ?? [];
  const byLoop = useMemo(() => {
    if (!workspaceId) return new Map<string, LoopBindingKind[]>();
    const index = buildLoopBindingIndex(triggers, jobs, workspaceId);
    return new Map([...index].map(([name, entry]) => [name, entry.kinds]));
  }, [triggers, jobs, workspaceId]);
  return { byLoop, isLoading: triggersQuery.isLoading || jobsQuery.isLoading };
}

export interface LoopBindingsResult {
  rows: readonly LoopBindingRow[];
  isLoading: boolean;
}

/**
 * The attached loop-target automations for one Loop, using the `loop=<name>`
 * automation list filter (detail Start-bindings panel).
 */
export function useLoopBindings(workspaceId: string, loopName: string): LoopBindingsResult {
  const enabled = workspaceId !== "" && loopName !== "";
  const triggersQuery = useAutomationTriggers(
    { loop: loopName, limit: BINDING_SCAN_LIMIT },
    { enabled }
  );
  const jobsQuery = useAutomationJobs({ loop: loopName, limit: BINDING_SCAN_LIMIT }, { enabled });
  const triggers = triggersQuery.data ?? [];
  const jobs = jobsQuery.data ?? [];
  const rows = useMemo(() => {
    if (!workspaceId || !loopName) return EMPTY_ROWS;
    return buildLoopBindingIndex(triggers, jobs, workspaceId).get(loopName)?.rows ?? EMPTY_ROWS;
  }, [triggers, jobs, workspaceId, loopName]);
  return { rows, isLoading: triggersQuery.isLoading || jobsQuery.isLoading };
}
