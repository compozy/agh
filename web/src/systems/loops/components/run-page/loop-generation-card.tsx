import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { Pill, PillDot, type PillTone } from "@agh/ui";

import type { LoopChannelMessage, LoopGateVerdict } from "../../lib/loop-events";
import type { LoopTimelineGeneration } from "../../lib/loop-timeline";
import type { GoalTurnTimelineItem } from "../../hooks/use-goal-turns";
import { LoopNodeRow } from "./loop-node-row";
import { LoopRunChannel } from "./loop-run-channel";
import { GoalTurnTimeline } from "./goal-turn-timeline";

interface LoopGenerationCardProps {
  generation: LoopTimelineGeneration;
  gateVerdicts: Record<string, LoopGateVerdict>;
  channelMessages: LoopChannelMessage[];
  isLive: boolean;
  goalTurns?: readonly GoalTurnTimelineItem[];
}

/** Channel-posting action nodes embed the converse-and-decide transcript (ADR-021). */
const CHANNEL_NODE_KIND = "agh__network_send";

/** Aggregate generation status for the header pill from its node states. */
function generationSummary(generation: LoopTimelineGeneration): {
  label: string;
  tone: PillTone;
  pulse: boolean;
} {
  const statuses = generation.nodes.map(node => node.status);
  if (statuses.some(status => status === "running" || status === "awaiting_child")) {
    return { label: "running", tone: "accent", pulse: true };
  }
  // `failed` and `blocked` are distinct families (danger vs warning), so the aggregate
  // pill names each separately and never labels a blocked-only generation "failed" (N-001).
  const failed = statuses.filter(status => status === "failed").length;
  const blocked = statuses.filter(status => status === "blocked").length;
  if (failed > 0 || blocked > 0) {
    const parts: string[] = [];
    if (failed > 0) parts.push(`${failed} failed`);
    if (blocked > 0) parts.push(`${blocked} blocked`);
    return { label: parts.join(" · "), tone: failed > 0 ? "danger" : "warning", pulse: false };
  }
  if (statuses.some(status => status === "pending" || status === "ready")) {
    return { label: "pending", tone: "neutral", pulse: false };
  }
  return { label: "completed", tone: "success", pulse: false };
}

/**
 * One collapsible generation on the timeline (§4.4): a header with the generation
 * number and an aggregate status pill, and a body rendering the node spine. The
 * latest generation opens expanded and accents its border while live. A channel
 * action node in the latest generation embeds its conversation.
 */
export function LoopGenerationCard({
  generation,
  gateVerdicts,
  channelMessages,
  isLive,
  goalTurns = [],
}: LoopGenerationCardProps) {
  const [open, setOpen] = useState(generation.isLatest);
  const summary = generationSummary(generation);
  const liveBorder = generation.isLatest && isLive ? "border-accent/25" : "border-line";
  return (
    <section
      className={`overflow-hidden rounded-lg border bg-canvas-soft ${liveBorder}`}
      data-testid={`loop-generation-${generation.generation}`}
    >
      <button
        type="button"
        className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-fg-strong/[0.02]"
        aria-expanded={open}
        onClick={() => setOpen(value => !value)}
        data-testid={`loop-generation-toggle-${generation.generation}`}
      >
        <span className="w-6 font-mono text-[11px] font-semibold text-faint">
          G{generation.generation}
        </span>
        <span className="text-[13.5px] font-medium text-fg-strong">
          Generation {generation.generation}
        </span>
        <span className="flex-1" />
        <Pill tone={summary.tone} pulse={summary.pulse} size="sm">
          <PillDot />
          {summary.label}
        </Pill>
        <ChevronDown
          className={`size-3.5 text-muted transition-transform ${open ? "" : "-rotate-90"}`}
          aria-hidden="true"
        />
      </button>
      {open ? (
        <div className="border-t border-line-soft px-5 pt-2 pb-4">
          <div className="relative pl-7 before:absolute before:top-3.5 before:bottom-3.5 before:left-2.5 before:w-px before:bg-line-strong">
            {generation.nodes.map(node => (
              <LoopNodeRow
                key={`${node.nodeId}-${node.itemIndex ?? "x"}`}
                node={node}
                gateVerdict={node.isGate ? gateVerdicts[node.nodeId] : undefined}
              >
                {node.kind === "goal" ? (
                  <GoalTurnTimeline
                    className="mt-3"
                    live={isLive && generation.isLatest && node.status === "running"}
                    turns={goalTurns.filter(
                      turn =>
                        turn.generation === generation.generation &&
                        turn.nodeId === node.nodeId &&
                        (node.itemIndex === undefined || turn.itemIndex === node.itemIndex)
                    )}
                  />
                ) : null}
                {generation.isLatest && node.kind === CHANNEL_NODE_KIND ? (
                  <LoopRunChannel
                    messages={channelMessages}
                    channelLabel={`#${node.nodeId}`}
                    isLive={isLive}
                  />
                ) : null}
              </LoopNodeRow>
            ))}
          </div>
        </div>
      ) : null}
    </section>
  );
}
