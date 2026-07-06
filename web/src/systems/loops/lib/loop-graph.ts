import type { LoopDefinition, LoopDefinitionGraph } from "../types";

/** The closed node-class vocabulary from the OpenAPI contract; color never encodes class. */
export type LoopNodeClass = LoopDefinitionGraph["nodes"][number]["class"];

const LOOP_NODE_CLASSES = [
  "action",
  "control",
  "source",
] as const satisfies readonly LoopNodeClass[];
const LOOP_NODE_CLASS_SET = new Set<LoopNodeClass>(LOOP_NODE_CLASSES);

/**
 * A read-only projection of one graph node for the Loop-detail body view.
 *
 * The daemon serializes a full `dsl.Node` (id / class / kind + the fan-out knobs,
 * gate verdict policy, etc.), but the generated OpenAPI types the graph as
 * `Record<string, never>[]` because the Go `Node` uses an inline `Extra` map the
 * codegen can't introspect. This projection is the hand-authored counterpart to
 * the generated OpenAPI graph shape and is read through runtime guards, so a
 * malformed node degrades gracefully instead of throwing.
 */
export interface LoopGraphNode {
  id: string;
  nodeClass: LoopNodeClass | null;
  kind: string;
  /** Fan-out knobs (present on `control` `fan-out` nodes). */
  batchSize?: number;
  maxParallel?: number;
  maxFanOut?: number;
  /** Gate metadata (present on `control` `gate` nodes). */
  isGate: boolean;
}

export interface LoopGraphEdge {
  from: string;
  to: string;
}

export interface LoopGraph {
  nodes: LoopGraphNode[];
  edges: LoopGraphEdge[];
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null ? (value as Record<string, unknown>) : null;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function asOptionalNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

export function toLoopNodeClass(value: unknown): LoopNodeClass | null {
  return typeof value === "string" && LOOP_NODE_CLASS_SET.has(value as LoopNodeClass)
    ? (value as LoopNodeClass)
    : null;
}

function projectNode(raw: unknown): LoopGraphNode | null {
  const record = asRecord(raw);
  if (!record) return null;
  const id = asString(record.id);
  if (id === "") return null;
  const kind = asString(record.kind);
  return {
    id,
    nodeClass: toLoopNodeClass(record.class),
    kind,
    batchSize: asOptionalNumber(record.batch_size),
    maxParallel: asOptionalNumber(record.max_parallel),
    maxFanOut: asOptionalNumber(record.max_fan_out),
    isGate: kind === "gate" || Boolean(record.criteria) || Boolean(record.verdict_policy),
  };
}

function projectEdge(raw: unknown): LoopGraphEdge | null {
  const record = asRecord(raw);
  if (!record) return null;
  const from = asString(record.from);
  const to = asString(record.to);
  if (from === "" || to === "") return null;
  return { from, to };
}

/**
 * Projects a Loop definition's graph into the typed read-only shape the detail
 * body view renders. Unreadable nodes/edges are dropped rather than surfaced as
 * empty rows.
 */
export function readLoopGraph(definition: Pick<LoopDefinition, "graph">): LoopGraph {
  const graph = definition.graph as unknown as { nodes?: unknown; edges?: unknown } | undefined;
  const rawNodes = Array.isArray(graph?.nodes) ? graph?.nodes : [];
  const rawEdges = Array.isArray(graph?.edges) ? graph?.edges : [];
  return {
    nodes: rawNodes.map(projectNode).filter((node): node is LoopGraphNode => node !== null),
    edges: rawEdges.map(projectEdge).filter((edge): edge is LoopGraphEdge => edge !== null),
  };
}

/** The node-class label rendered as neutral mono text (never color-coded). */
export function nodeClassLabel(node: LoopGraphNode): string {
  if (node.nodeClass === "control" && node.kind === "fan-out") return "control · fan-out";
  if (node.nodeClass === "control" && node.isGate) return "control · gate";
  return node.nodeClass ?? "node";
}

/** The fan-out summary line (`batch 1 · seq · ≤64`) shown under a fan-out node. */
export function fanOutSummary(node: LoopGraphNode): string | null {
  if (
    node.batchSize === undefined &&
    node.maxParallel === undefined &&
    node.maxFanOut === undefined
  ) {
    return null;
  }
  const parts: string[] = [];
  if (node.batchSize !== undefined) parts.push(`batch ${node.batchSize}`);
  if (node.maxParallel !== undefined)
    parts.push(node.maxParallel <= 1 ? "seq" : `×${node.maxParallel}`);
  if (node.maxFanOut !== undefined) parts.push(`≤${node.maxFanOut}`);
  return parts.join(" · ");
}
