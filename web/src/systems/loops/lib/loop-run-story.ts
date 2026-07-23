import type { PillTone } from "@agh/ui";

import type { LoopRunEventFrame, LoopRunGeneration } from "../types";
import { parseLoopActionFailure } from "./action-failure";
import { isTerminalLoopStatus } from "./loop-formatters";
import { type LoopGraph, findWatchNode, goalNodeIds, topoOrder } from "./loop-graph";

/**
 * The event → story adapter (redesign spec §5.3): folds the replayed SSE frames
 * into plain-language story rows, the "Happening now" projection, and the
 * "What happens next" note. Every string is a template over real payload fields
 * — domain flavor comes only from node ids, inputs, and payloads (§5.1); the
 * runtime identifiers survive verbatim in the mono `micro` labels.
 */

export type LoopStoryIcon =
  | "round"
  | "check-pass"
  | "check-warn"
  | "node-done"
  | "node-failed"
  | "approval"
  | "watching"
  | "paused"
  | "resumed"
  | "started"
  | "stopped"
  | "done";

export interface LoopStoryIssue {
  id: string;
  note?: string;
}

export interface LoopStoryTaskLink {
  taskId: string;
  taskRunId: string;
}

export interface LoopStoryRow {
  key: string;
  kind: string;
  seq: number;
  at: string;
  tone: PillTone;
  icon: LoopStoryIcon;
  title: string;
  sub?: string;
  /** Blocking issues rendered after `sub` with mono ids (`gate_verdict · revise`). */
  issues?: LoopStoryIssue[];
  /** Verbatim runtime identifier trail (`gate_verdict · revise`). */
  micro: string;
  /** Source node id when the row belongs to one node (Turns disclosure anchor). */
  nodeId?: string;
  generation?: number;
}

export interface LoopStoryNow {
  nodeId: string;
  /** Humanized node label (`fix_batches` → "fix batches"). */
  label: string;
  itemIndex?: number;
  generation: number;
  startedAt: string;
  taskLink?: LoopStoryTaskLink;
  /** Set while a `run-loop` node awaits its child run (links `/loop-runs/$runId`). */
  childRunId?: string;
  isGoalNode: boolean;
}

export interface LoopRunStory {
  /** Newest-first story rows. */
  rows: LoopStoryRow[];
  /** The newest running node without a terminal event; null when nothing runs. */
  now: LoopStoryNow | null;
}

export interface LoopRunStoryContext {
  status?: string | null;
  reattemptStrategy?: string;
  graph: LoopGraph | null;
  generations?: readonly LoopRunGeneration[];
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null ? (value as Record<string, unknown>) : null;
}

function str(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function num(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

/** `fix_batches` → "fix batches" — the generic humanization of a node id. */
export function humanizeLoopNodeId(nodeId: string): string {
  return nodeId.replace(/[_-]+/g, " ").trim();
}

function sentenceCase(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function confidenceClause(confidence: number | undefined): string {
  return confidence === undefined ? "" : ` · confidence ${confidence.toFixed(2)}`;
}

interface MutableStoryRow extends LoopStoryRow {
  /** Consecutive same-node success grouping accumulator (§5.3). */
  group?: { nodeId: string; generation: number; count: number; indexes: number[] };
}

interface RunningEntry {
  nodeId: string;
  itemIndex?: number;
  generation: number;
  startedAt: string;
  seq: number;
  taskId?: string;
  taskRunId?: string;
}

function runningKey(nodeId: string, itemIndex: number | undefined): string {
  return `${nodeId}:${itemIndex ?? "x"}`;
}

function branchKey(generation: number, nodeId: string): string {
  return `${generation}:${nodeId}`;
}

function taskLink(payload: Record<string, unknown>): LoopStoryTaskLink | undefined {
  const taskId = str(payload.task_id);
  const taskRunId = str(payload.task_run_id);
  return taskId && taskRunId ? { taskId, taskRunId } : undefined;
}

function generationStartedRow(
  frame: LoopRunEventFrame,
  payload: Record<string, unknown>,
  fallbackStrategy: string | undefined
): MutableStoryRow {
  const generation = num(payload.generation) ?? 0;
  const strategy = str(payload.reattempt_strategy, fallbackStrategy ?? "");
  const failedOnly = strategy === "failed_only" && generation > 1;
  return {
    key: `f-${frame.seq}`,
    kind: "generation_started",
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    tone: "neutral",
    icon: "round",
    title: `Round ${generation} started${failedOnly ? " — redoing the failed group(s)" : ""}`,
    sub: failedOnly
      ? "Failed-only re-attempt: clean groups carry over untouched; only the failed groups run again."
      : undefined,
    micro: `generation_started · gen ${generation}`,
    generation,
  };
}

function gateVerdictRow(
  frame: LoopRunEventFrame,
  payload: Record<string, unknown>
): MutableStoryRow {
  const passed = str(payload.verdict) === "pass";
  const confidence = num(payload.confidence);
  const rawIssues = Array.isArray(payload.blocking_issues) ? payload.blocking_issues : [];
  const issues: LoopStoryIssue[] = [];
  for (const item of rawIssues) {
    const record = asRecord(item);
    const id = str(record?.id);
    if (id !== "") issues.push({ id, note: str(record?.note) || undefined });
  }
  return {
    key: `f-${frame.seq}`,
    kind: "gate_verdict",
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    tone: passed ? "success" : "warning",
    icon: passed ? "check-pass" : "check-warn",
    title: passed ? "Check passed — everything handled" : "Check: not clean yet",
    sub: `Verdict ${passed ? "pass" : "revise"}${confidenceClause(confidence)}${passed ? "" : "."}`,
    issues: passed ? undefined : issues,
    micro: `gate_verdict · ${passed ? "pass" : "revise"}`,
    nodeId: str(payload.node_id) || undefined,
    generation: num(payload.generation),
  };
}

function nodeFailedRow(
  frame: LoopRunEventFrame,
  payload: Record<string, unknown>,
  failedOnlyLive: boolean
): MutableStoryRow {
  const nodeId = str(payload.node_id);
  const itemIndex = num(payload.item_index);
  const failure = parseLoopActionFailure(str(payload.output_ref) || undefined);
  const redoClause = failedOnlyLive
    ? " An incomplete group fails as a whole and is queued to be redone."
    : "";
  return {
    key: `f-${frame.seq}`,
    kind: "node_failed",
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    tone: "danger",
    icon: "node-failed",
    title: `${sentenceCase(humanizeLoopNodeId(nodeId))} came up short`,
    sub: failure ? `${failure.cause}${redoClause}` : redoClause.trim() || undefined,
    micro: `node_failed · ${nodeId}${itemIndex === undefined ? "" : `[${itemIndex}]`}`,
    nodeId,
    generation: num(payload.generation),
  };
}

function needsApprovalRow(
  frame: LoopRunEventFrame,
  payload: Record<string, unknown>
): MutableStoryRow {
  return {
    key: `f-${frame.seq}`,
    kind: "needs_approval",
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    tone: "warning",
    icon: "approval",
    title: "Asked for your approval",
    sub: str(payload.title) || undefined,
    micro: `needs_approval · ${str(payload.gate_id, "approve")}`,
    generation: num(payload.generation),
  };
}

interface Transition {
  tone: PillTone;
  icon: LoopStoryIcon;
  title: string;
  sub?: string;
}

function wakeTransition(cause: string): Transition {
  const title =
    cause === "watch_poll" ? "A scheduled check woke the run" : "A new event woke the run";
  return {
    tone: "info",
    icon: "watching",
    title,
    sub: "The run left watching and started a new round.",
  };
}

function terminalTransition(
  to: string,
  cause: string,
  failureCause: string | undefined
): Transition | null {
  switch (to) {
    case "failed":
      return { tone: "danger", icon: "node-failed", title: "Run failed", sub: failureCause };
    case "done":
      return { tone: "success", icon: "done", title: "Finished as Done" };
    case "no-op":
      return { tone: "neutral", icon: "done", title: "Ran — nothing to do" };
    case "blocked":
      return { tone: "warning", icon: "check-warn", title: "Run blocked", sub: failureCause };
    case "exhausted":
      return {
        tone: "warning",
        icon: "check-warn",
        title:
          cause === "iteration_cap"
            ? "Round cap reached — run exhausted"
            : "A limit was reached — run exhausted",
        sub: failureCause,
      };
    case "stalled":
      return {
        tone: "neutral",
        icon: "check-warn",
        title: "Run stalled — no progress",
        sub:
          cause === "no_progress"
            ? "The same open points kept coming back round after round."
            : failureCause,
      };
    default:
      return null;
  }
}

function statusTransition(
  from: string,
  to: string,
  cause: string,
  failureCause: string | undefined
): Transition | null {
  if (cause === "operator_stop") {
    return { tone: "neutral", icon: "stopped", title: "Stopped by you" };
  }
  const terminal = terminalTransition(to, cause, failureCause);
  if (terminal) return terminal;
  switch (to) {
    case "running":
      if (from === "watching") return wakeTransition(cause);
      if (from === "paused") {
        return {
          tone: "neutral",
          icon: "resumed",
          title: "Run resumed",
          sub: "Continuing exactly where it left off.",
        };
      }
      if (from === "needs-approval") {
        return { tone: "neutral", icon: "resumed", title: "Approved — the run resumed" };
      }
      return { tone: "neutral", icon: "started", title: "Run started" };
    case "watching":
      return from === "running"
        ? {
            tone: "info",
            icon: "watching",
            title: "Went back to watching",
            sub: "Nothing left to do until a new event arrives.",
          }
        : { tone: "info", icon: "watching", title: "Started watching" };
    case "paused":
      return {
        tone: "neutral",
        icon: "paused",
        title: "Run paused",
        sub: "Requested by you; the run parked at a safe point.",
      };
    case "queued":
      return { tone: "neutral", icon: "started", title: "Run queued", sub: "Waiting for a slot." };
    // `needs-approval` renders through its own `needs_approval` frame — a status
    // row here would duplicate the story beat.
    default:
      return null;
  }
}

function statusChangedRow(
  frame: LoopRunEventFrame,
  payload: Record<string, unknown>
): MutableStoryRow | null {
  const from = str(payload.from);
  const to = str(payload.to, str(payload.status));
  const cause = str(payload.cause);
  const failure = asRecord(payload.failure);
  const transition = statusTransition(from, to, cause, str(failure?.cause) || undefined);
  if (!transition) return null;
  return {
    key: `f-${frame.seq}`,
    kind: "status_changed",
    seq: num(frame.seq) ?? 0,
    at: str(frame.at),
    tone: transition.tone,
    icon: transition.icon,
    title: transition.title,
    sub: transition.sub,
    micro: `status_changed · ${from || "?"} → ${to || "?"}`,
  };
}

/** Finalizes grouped success rows: `{label} — {k} of {n} clean` + `[i–j]` micro. */
function finalizeSuccessRow(row: MutableStoryRow, branchTotal: number): LoopStoryRow {
  const { group, ...publicRow } = row;
  if (!group) return publicRow;
  const label = humanizeLoopNodeId(group.nodeId);
  const indexes = [...group.indexes].sort((a, b) => a - b);
  const hasBranches = branchTotal > 1;
  const micro =
    indexes.length === 0
      ? `node_succeeded · ${group.nodeId}`
      : indexes.length === 1
        ? `node_succeeded · ${group.nodeId}[${indexes[0]}]`
        : `node_succeeded · ${group.nodeId}[${indexes[0]}–${indexes[indexes.length - 1]}]`;
  return {
    ...publicRow,
    title: hasBranches
      ? `${sentenceCase(label)} — ${group.count} of ${branchTotal} clean`
      : `Finished ${label}`,
    micro,
  };
}

/**
 * Builds the story timeline + "Happening now" from the retained frames. Frames
 * arrive in seq order; rows return newest-first. Consecutive branch successes of
 * one node collapse into a single row; `node_running` never renders a row — it
 * feeds the now-card until a terminal node event or a status change clears it.
 */
export function buildRunStory(
  frames: readonly LoopRunEventFrame[],
  context: LoopRunStoryContext
): LoopRunStory {
  const live = !isTerminalLoopStatus(context.status);
  const failedOnlyLive = live && context.reattemptStrategy === "failed_only";
  const goalIds = context.graph ? goalNodeIds(context.graph) : new Set<string>();
  const rows: MutableStoryRow[] = [];
  const running = new Map<string, RunningEntry>();
  const branches = new Map<string, Set<number>>();

  const trackBranch = (generation: number, nodeId: string, itemIndex: number | undefined) => {
    if (itemIndex === undefined) return;
    const key = branchKey(generation, nodeId);
    const set = branches.get(key) ?? new Set<number>();
    set.add(itemIndex);
    branches.set(key, set);
  };

  for (const frame of frames) {
    const kind = str(frame.kind);
    const payload = asRecord(frame.payload);
    if (!payload) continue;
    switch (kind) {
      case "generation_started":
        rows.push(generationStartedRow(frame, payload, context.reattemptStrategy));
        break;
      case "gate_verdict":
        rows.push(gateVerdictRow(frame, payload));
        break;
      case "needs_approval":
        rows.push(needsApprovalRow(frame, payload));
        break;
      case "node_running": {
        const nodeId = str(payload.node_id);
        if (nodeId === "") break;
        const itemIndex = num(payload.item_index);
        const generation = num(payload.generation) ?? 0;
        trackBranch(generation, nodeId, itemIndex);
        const link = taskLink(payload);
        running.set(runningKey(nodeId, itemIndex), {
          nodeId,
          itemIndex,
          generation,
          startedAt: str(frame.at),
          seq: num(frame.seq) ?? 0,
          taskId: link?.taskId,
          taskRunId: link?.taskRunId,
        });
        break;
      }
      case "node_succeeded": {
        const nodeId = str(payload.node_id);
        if (nodeId === "") break;
        const itemIndex = num(payload.item_index);
        const generation = num(payload.generation) ?? 0;
        trackBranch(generation, nodeId, itemIndex);
        running.delete(runningKey(nodeId, itemIndex));
        const previous = rows[rows.length - 1];
        if (
          previous?.group &&
          previous.group.nodeId === nodeId &&
          previous.group.generation === generation
        ) {
          previous.group.count += 1;
          if (itemIndex !== undefined) previous.group.indexes.push(itemIndex);
          previous.at = str(frame.at);
          previous.seq = num(frame.seq) ?? previous.seq;
          break;
        }
        rows.push({
          key: `f-${frame.seq}`,
          kind: "node_succeeded",
          seq: num(frame.seq) ?? 0,
          at: str(frame.at),
          tone: "success",
          icon: "node-done",
          title: `Finished ${humanizeLoopNodeId(nodeId)}`,
          micro: `node_succeeded · ${nodeId}`,
          nodeId,
          generation,
          group: {
            nodeId,
            generation,
            count: 1,
            indexes: itemIndex === undefined ? [] : [itemIndex],
          },
        });
        break;
      }
      case "node_failed": {
        const nodeId = str(payload.node_id);
        if (nodeId === "") break;
        const itemIndex = num(payload.item_index);
        trackBranch(num(payload.generation) ?? 0, nodeId, itemIndex);
        running.delete(runningKey(nodeId, itemIndex));
        rows.push(nodeFailedRow(frame, payload, failedOnlyLive));
        break;
      }
      case "status_changed": {
        const to = str(payload.to, str(payload.status));
        if (to !== "running") running.clear();
        const row = statusChangedRow(frame, payload);
        if (row) rows.push(row);
        break;
      }
      default:
        break;
    }
  }

  // The latest round marker is the live one — accent while the run works,
  // neutral once superseded or parked (§5.3, prototype states file).
  if (context.status === "running") {
    const latestRound = [...rows].reverse().find(row => row.kind === "generation_started");
    if (latestRound) latestRound.tone = "accent";
  }

  const finalized = rows.map(row =>
    finalizeSuccessRow(
      row,
      row.group ? (branches.get(branchKey(row.group.generation, row.group.nodeId))?.size ?? 0) : 0
    )
  );

  return {
    rows: finalized.reverse(),
    now: live ? projectNow(running, context, goalIds) : null,
  };
}

function projectNow(
  running: Map<string, RunningEntry>,
  context: LoopRunStoryContext,
  goalIds: Set<string>
): LoopStoryNow | null {
  let newest: RunningEntry | null = null;
  for (const entry of running.values()) {
    if (!newest || entry.seq > newest.seq) newest = entry;
  }
  if (!newest) return null;
  const current = newest;
  const childRunId = context.generations
    ?.flatMap(generation => generation.outputs)
    .find(
      output =>
        output.node_id === current.nodeId &&
        (output.item_index ?? undefined) === current.itemIndex &&
        output.status === "awaiting_child"
    )?.child_loop_run_id;
  return {
    nodeId: current.nodeId,
    label: humanizeLoopNodeId(current.nodeId),
    itemIndex: current.itemIndex,
    generation: current.generation,
    startedAt: current.startedAt,
    taskLink:
      current.taskId && current.taskRunId
        ? { taskId: current.taskId, taskRunId: current.taskRunId }
        : undefined,
    childRunId: childRunId || undefined,
    isGoalNode: goalIds.has(current.nodeId),
  };
}

/** Joins humanized labels as an ordered sequence: `a`, `a, then b`, `a, b, then c`. */
function joinLabels(labels: string[]): string {
  if (labels.length <= 1) return labels[0] ?? "";
  return `${labels.slice(0, -1).join(", ")}, then ${labels[labels.length - 1]}`;
}

/** Latest-generation output statuses that still count as "ahead" work. */
const PENDING_OUTPUT_STATUSES = new Set(["pending", "ready", "enqueued"]);

/**
 * The "What happens next" quiet note (§5.3): remaining downstream nodes of the
 * current generation in topological order, humanized into one sentence, ending
 * with the watch/stop clause. Static per definition — no invention.
 */
export function buildNextNote(
  graph: LoopGraph | null,
  generations: readonly LoopRunGeneration[] | undefined
): string | null {
  if (!graph || graph.nodes.length === 0) return null;
  const watchNode = findWatchNode(graph);
  const hasGate = graph.nodes.some(node => node.isGate);
  const latest = (generations ?? []).reduce((max, gen) => Math.max(max, gen.generation), 0);
  const latestOutputs = new Map<string, string>();
  for (const generation of generations ?? []) {
    if (generation.generation !== latest) continue;
    for (const output of generation.outputs) {
      latestOutputs.set(output.node_id, output.status);
    }
  }
  const remaining = topoOrder(graph).filter(id => {
    if (watchNode && id === watchNode.id) return false;
    const status = latestOutputs.get(id);
    return status === undefined || PENDING_OUTPUT_STATUSES.has(status);
  });
  const ahead =
    remaining.length > 0
      ? `Still ahead this round: ${joinLabels(remaining.map(humanizeLoopNodeId))}.`
      : "";
  const clause = watchNode
    ? hasGate
      ? "After that, the run goes back to watching. If the next check comes back clean, it finishes as Done; if new work arrives, a new round starts automatically."
      : "After that, the run goes back to watching and finishes when there's nothing left to do."
    : hasGate
      ? "When every step is done, a final check decides: clean finishes the run as Done; anything left starts another round."
      : "When every step is done, the run finishes.";
  return [ahead, clause].filter(Boolean).join(" ");
}
