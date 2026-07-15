import type { PillTone } from "@agh/ui";

import type { LoopDefinition, LoopRunGeneration } from "../types";
import { parseLoopActionFailure, type LoopActionFailure } from "./action-failure";
import { type LoopGraphNode, nodeClassLabel, readLoopGraph } from "./loop-graph";

/**
 * Per-node runtime states from `loop_generation_outputs.status`. `ready` and
 * `awaiting_child` are node-level only, never run statuses (ADR-021, §3.3):
 * `awaiting_child` marks a `run-loop await` node parked until its child terminates.
 * `reused`/`skipped` are the failed-only carry-forward states (§5.5).
 */
const NODE_STATUS_TONE: Record<string, PillTone> = {
  ready: "info",
  pending: "neutral",
  running: "accent",
  awaiting_child: "info",
  succeeded: "success",
  reused: "success",
  skipped: "neutral",
  "no-op": "neutral",
  blocked: "warning",
  failed: "danger",
};

/** Node states that pulse on the spine (live work), gated by reduced-motion at render. */
const NODE_STATUS_PULSE = new Set<string>(["running", "awaiting_child"]);

/** Explicit carry-forward states injected read-only under `failed-only` (§5.5). */
const CARRY_FORWARD_STATUSES = new Set<string>(["reused", "skipped"]);

export interface LoopTimelineNode {
  nodeId: string;
  /** Neutral mono class label (`action` / `control · gate` / `source`). */
  classLabel: string;
  kind: string;
  status: string;
  tone: PillTone;
  pulse: boolean;
  isGate: boolean;
  /** Carried forward read-only from a prior generation (failed-only re-attempt). */
  isCarriedForward: boolean;
  /** A `run-loop await` node's child run; links to the awaited child (round-10 N-004). */
  childLoopRunId?: string;
  /** Fan-out branch index when this output is one materialized branch. */
  itemIndex?: number;
  /** Sanitized durable failure detail for a failed action node. */
  failure?: LoopActionFailure;
}

export interface LoopTimelineGeneration {
  generation: number;
  nodes: LoopTimelineNode[];
  /** The newest generation, rendered expanded and border-accented when live. */
  isLatest: boolean;
}

function nodeTone(status: string): PillTone {
  return NODE_STATUS_TONE[status] ?? "neutral";
}

function projectTimelineNode(
  output: LoopRunGeneration["outputs"][number],
  graphNode: LoopGraphNode | undefined
): LoopTimelineNode {
  const status = output.status;
  return {
    nodeId: output.node_id,
    classLabel: graphNode ? nodeClassLabel(graphNode) : "node",
    kind: graphNode?.kind ?? "",
    status,
    tone: nodeTone(status),
    pulse: NODE_STATUS_PULSE.has(status),
    isGate: graphNode?.isGate ?? false,
    isCarriedForward: CARRY_FORWARD_STATUSES.has(status),
    childLoopRunId: output.child_loop_run_id,
    itemIndex: output.item_index,
    failure:
      status === "failed" ? (parseLoopActionFailure(output.output_ref) ?? undefined) : undefined,
  };
}

/**
 * Builds the collapsible generation timeline: each generation's node outputs enriched
 * with the node's class/kind from the definition graph (the run projection carries
 * only `node_id` + `status`). Generations render newest-first; the latest is flagged
 * so the route can expand it and accent a live border.
 */
export function buildRunTimeline(
  generations: readonly LoopRunGeneration[] | undefined,
  definition?: Pick<LoopDefinition, "graph">
): LoopTimelineGeneration[] {
  if (!generations || generations.length === 0) return [];
  const graph = definition ? readLoopGraph(definition) : null;
  const nodeById = new Map(graph?.nodes.map(node => [node.id, node]) ?? []);
  const latest = generations.reduce((max, gen) => Math.max(max, gen.generation), 0);
  return [...generations]
    .sort((a, b) => b.generation - a.generation)
    .map(gen => ({
      generation: gen.generation,
      isLatest: gen.generation === latest,
      nodes: gen.outputs.map(output => projectTimelineNode(output, nodeById.get(output.node_id))),
    }));
}

/**
 * The materialized fan-out breadth of the latest generation: the count of distinct
 * `item_index` branches (§4.4 Breadth meter). Zero when the tick fanned out nothing.
 */
export function latestGenerationBreadth(
  generations: readonly LoopRunGeneration[] | undefined
): number {
  if (!generations || generations.length === 0) return 0;
  const latest = generations.reduce((max, gen) => Math.max(max, gen.generation), 0);
  const indices = new Set<number>();
  for (const gen of generations) {
    if (gen.generation !== latest) continue;
    for (const output of gen.outputs) {
      if (typeof output.item_index === "number") indices.add(output.item_index);
    }
  }
  return indices.size;
}

export { NODE_STATUS_TONE };
