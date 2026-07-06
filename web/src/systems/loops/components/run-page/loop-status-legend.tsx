import { Eyebrow, type PillTone } from "@agh/ui";

import { LOOP_STATUS_LABELS, loopStatusTone } from "../../lib/loop-formatters";
import type { LoopRunStatus } from "../../types";

/** The 5 live and 6 terminal statuses in their design display order (§3.3). */
const LIVE_STATUSES: LoopRunStatus[] = [
  "queued",
  "running",
  "watching",
  "needs-approval",
  "paused",
];
const TERMINAL_STATUSES: LoopRunStatus[] = [
  "done",
  "no_op",
  "blocked",
  "failed",
  "exhausted",
  "stalled",
];

const DOT_CLASS: Record<PillTone, string> = {
  neutral: "bg-neutral",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/** One legend hint per status (why the state exists), truthful to §5.1. */
const STATUS_HINT: Record<LoopRunStatus, string> = {
  queued: "deferred start",
  running: "generation active",
  watching: "awaiting next tick",
  "needs-approval": "live pause",
  paused: "operator suspended",
  done: "goal verified",
  no_op: "ran, nothing to do",
  blocked: "external dependency",
  failed: "unrecoverable error",
  exhausted: "limit reached",
  stalled: "no progress",
};

function LegendRow({ status }: { status: LoopRunStatus }) {
  return (
    <div
      className="flex items-center gap-2.5 text-[11.5px] text-muted"
      data-testid={`loop-legend-${status}`}
    >
      <span
        aria-hidden="true"
        className={`size-1.5 rounded-full ${DOT_CLASS[loopStatusTone(status)]}`}
      />
      <b className="text-[12px] font-medium text-fg">{LOOP_STATUS_LABELS[status]}</b>
      <span className="ml-auto text-[10.5px] text-faint">{STATUS_HINT[status]}</span>
    </div>
  );
}

/**
 * The run-status legend rail (§4.4): all 11 first-class statuses grouped live vs
 * terminal, each with its truthful tone dot. `ready`/`awaiting_child` are absent — they
 * are node-level states, never run statuses (§3.3).
 */
export function LoopStatusLegend() {
  return (
    <div className="px-4 py-4" data-testid="loop-status-legend">
      <Eyebrow className="mb-3 text-faint">Run statuses · 11</Eyebrow>
      <div className="flex flex-col gap-2">
        <Eyebrow className="text-faint">Live</Eyebrow>
        {LIVE_STATUSES.map(status => (
          <LegendRow key={status} status={status} />
        ))}
        <Eyebrow className="mt-1 text-faint">Terminal</Eyebrow>
        {TERMINAL_STATUSES.map(status => (
          <LegendRow key={status} status={status} />
        ))}
      </div>
    </div>
  );
}
