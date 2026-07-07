import type { ReactNode } from "react";
import { ExternalLink } from "lucide-react";
import { Link } from "@tanstack/react-router";

import type { PillTone } from "@agh/ui";

import type { LoopGateVerdict } from "../../lib/loop-events";
import type { LoopTimelineNode } from "../../lib/loop-timeline";
import { MonoTag } from "../mono-tag";
import { LoopGateCard } from "./loop-gate-card";

interface LoopNodeRowProps {
  node: LoopTimelineNode;
  /** Gate verdict for this node, from a `gate_verdict` SSE frame. */
  gateVerdict?: LoopGateVerdict;
  /** Embedded channel (converse-and-decide) rendered under a channel action node. */
  children?: ReactNode;
}

const DOT_CLASS: Record<PillTone, string> = {
  neutral: "bg-neutral",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/**
 * Title-case labels for every node-level status (`loop_generation_outputs.status`),
 * so the spine never mixes casing between node-only states (`succeeded`/`reused`) and
 * states that happen to share a run-status name (N-001). An unmapped status falls back
 * to its raw string rather than being coerced.
 */
const NODE_STATUS_LABEL: Record<string, string> = {
  ready: "Ready",
  pending: "Pending",
  running: "Running",
  awaiting_child: "Awaiting child",
  succeeded: "Succeeded",
  reused: "Reused",
  skipped: "Skipped",
  "no-op": "No-op",
  blocked: "Blocked",
  failed: "Failed",
};

function nodeStatusLabel(status: string): string {
  return NODE_STATUS_LABEL[status] ?? status;
}

/** A stable per-instance testid suffix so fan-out branches of one node are distinct (N-003). */
function nodeKey(node: LoopTimelineNode): string {
  return node.itemIndex !== undefined ? `${node.nodeId}-${node.itemIndex}` : node.nodeId;
}

/**
 * One node on the generation spine (§4.4): a status dot, the neutral class label, the
 * snake_case node id, its kind, and the runtime status. A carried-forward node wears a
 * read-only tag (failed-only re-attempt); a `run-loop await` node in `awaiting_child`
 * links to the awaited child run (round-10 N-004). A gate node renders its verdict
 * card; a channel node embeds its conversation via `children`.
 */
export function LoopNodeRow({ node, gateVerdict, children }: LoopNodeRowProps) {
  const key = nodeKey(node);
  return (
    <div className="relative py-2.5" data-testid={`loop-node-${key}`} data-status={node.status}>
      <span
        aria-hidden="true"
        className={`absolute top-3 -left-[23px] z-10 size-2.5 rounded-full ring-2 ring-canvas ${DOT_CLASS[node.tone]} ${node.pulse ? "animate-pulse motion-reduce:animate-none" : ""}`}
      />
      <div className="flex flex-wrap items-center gap-2.5">
        <MonoTag className="rounded-xs bg-badge-fill px-1.5 py-0.5">{node.classLabel}</MonoTag>
        <span className="text-[13px] font-medium text-fg-strong">{node.nodeId}</span>
        {node.kind ? <span className="font-mono text-[11px] text-subtle">{node.kind}</span> : null}
        {node.isCarriedForward ? (
          <MonoTag
            className="rounded-xs border border-line-soft px-1.5 py-0.5 text-subtle"
            data-testid={`loop-node-carry-forward-${key}`}
          >
            carried forward · read-only
          </MonoTag>
        ) : null}
        <span className="ml-auto flex items-center gap-1.5 text-[11.5px] font-medium text-muted">
          <span
            aria-hidden="true"
            className={`size-1.5 rounded-full ${DOT_CLASS[node.tone]} ${node.pulse ? "animate-pulse motion-reduce:animate-none" : ""}`}
          />
          {nodeStatusLabel(node.status)}
        </span>
      </div>
      {node.status === "awaiting_child" && node.childLoopRunId ? (
        <Link
          to="/loop-runs/$runId"
          params={{ runId: node.childLoopRunId }}
          className="mt-1.5 inline-flex items-center gap-1.5 font-mono text-[11px] text-info hover:text-accent-strong"
          data-testid={`loop-node-child-link-${key}`}
        >
          child run {node.childLoopRunId}
          <ExternalLink className="size-3" aria-hidden="true" />
        </Link>
      ) : null}
      {gateVerdict ? <LoopGateCard verdict={gateVerdict} /> : null}
      {children}
    </div>
  );
}
