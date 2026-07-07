import type { LoopRunRecord } from "../types";
import { UNBOUNDED_CAP } from "./loop-catalog";
import { LOOP_CEILINGS } from "./loop-limits";
import { formatTokenCount } from "./loop-runs-view";

/** Meter tint: neutral until near the ceiling (warn), danger at/over it (§4.4). */
export type LoopMeterTone = "neutral" | "warn" | "danger";

export type LoopMeterKey = "attempts" | "tokens" | "wall" | "cost" | "breadth";

export interface LoopRunMeter {
  key: LoopMeterKey;
  label: string;
  /** Primary value label (`2`, `268K`, `22m`, `$1.12`, `3`). */
  value: string;
  /** Secondary `/ max` label when a ceiling exists (`/ 50`, `/ 1.5M`, `/ ∞`). */
  max?: string;
  /** Bar fill `0..100`; `null` renders no bar (unbounded or display-only). */
  percent: number | null;
  tone: LoopMeterTone;
  /** The cost meter: display-only, no bar, no cap — the tone is always neutral. */
  displayOnly: boolean;
  /** Foot tags (`Enforced`, `on_exceeded: escalate → needs-approval`, `Display-only`). */
  tags: string[];
}

/**
 * Coarse display-only USD estimate per 1M tokens (ADR-017 §9.5.2). AGH enforces no
 * USD cap and the run projection carries no per-model price, so this is a single
 * documented blended heuristic for operator visibility ONLY — never a billed figure
 * and never a budget dimension. It is superseded the day the daemon exposes a real
 * per-run rate; until then the Cost meter renders `tokens × this rate` and is
 * labelled an estimate, never a ceiling.
 */
export const LOOP_COST_PER_1M_TOKENS = 5;

/** Warn at 90% of a ceiling, danger at/over it (mirrors `loopBudgetBar`). */
function ratioTone(ratio: number): LoopMeterTone {
  if (ratio >= 1) return "danger";
  if (ratio >= 0.9) return "warn";
  return "neutral";
}

function clampPercent(ratio: number): number {
  return Math.max(0, Math.min(100, Math.round(ratio * 100)));
}

/** Compact budget label (`1.5M`, `500K`, `off`) for the meter's `/ max`. */
function tokenBudgetMax(budget: number): string | undefined {
  return budget > 0 ? formatTokenCount(budget) : undefined;
}

/** Elapsed wall-clock seconds derived from `created_at -> last_progress_at`. */
function elapsedSeconds(run: LoopRunRecord): number {
  const started = Date.parse(run.created_at);
  const last = Date.parse(run.last_progress_at);
  if (Number.isNaN(started) || Number.isNaN(last) || last < started) return 0;
  return Math.round((last - started) / 1000);
}

/** Compact wall-clock label (`22m`, `1.5h`, `2d`) for the meter value/ceiling. */
function wallLabel(seconds: number): string {
  if (seconds <= 0) return "0s";
  const day = 86_400;
  const hour = 3_600;
  const minute = 60;
  if (seconds >= day) return `${(seconds / day).toFixed(seconds % day === 0 ? 0 : 1)}d`;
  if (seconds >= hour) return `${Math.round(seconds / hour)}h`;
  if (seconds >= minute) return `${Math.round(seconds / minute)}m`;
  return `${seconds}s`;
}

/**
 * Derived display-only cost estimate label (`~$1.34`) from the run's token usage. The
 * leading `~` keeps the estimate qualifier visible next to the value (not only in the
 * meter's foot tag), since the figure comes from a client-side blended rate, not a
 * daemon-computed price (R-003, ADR-017 §9.5.2).
 */
export function deriveCostEstimate(tokensUsed: number): string {
  const usd = (Math.max(0, tokensUsed) / 1_000_000) * LOOP_COST_PER_1M_TOKENS;
  return `~$${usd.toFixed(2)}`;
}

function budgetPolicyTag(run: LoopRunRecord): string {
  return run.budget_on_exceeded === "escalate"
    ? "on_exceeded: escalate → needs-approval"
    : "on_exceeded: halt → exhausted";
}

function attemptsMeter(run: LoopRunRecord): LoopRunMeter {
  const unbounded = run.iteration_cap <= 0;
  const percent = unbounded ? null : clampPercent(run.generation / run.iteration_cap);
  return {
    key: "attempts",
    label: "Attempts",
    value: String(run.generation),
    max: unbounded ? `/ ${UNBOUNDED_CAP}` : `/ ${run.iteration_cap}`,
    percent,
    tone: unbounded ? "neutral" : ratioTone(run.generation / run.iteration_cap),
    displayOnly: false,
    tags: unbounded ? ["Unbounded", "until-clean"] : ["Enforced"],
  };
}

function tokensMeter(run: LoopRunRecord): LoopRunMeter {
  const capped = run.budget_tokens > 0;
  const ratio = capped ? run.tokens_used / run.budget_tokens : 0;
  return {
    key: "tokens",
    label: "Tokens",
    value: formatTokenCount(run.tokens_used),
    max: capped ? `/ ${tokenBudgetMax(run.budget_tokens)}` : undefined,
    percent: capped ? clampPercent(ratio) : null,
    tone: capped ? ratioTone(ratio) : "neutral",
    displayOnly: false,
    tags: capped ? ["Enforced", budgetPolicyTag(run)] : ["Off", "0 = unlimited"],
  };
}

function wallMeter(run: LoopRunRecord): LoopRunMeter {
  const elapsed = elapsedSeconds(run);
  const capped = run.budget_wall_sec > 0;
  const ratio = capped ? elapsed / run.budget_wall_sec : 0;
  return {
    key: "wall",
    label: "Wall clock",
    value: wallLabel(elapsed),
    max: capped ? `/ ${wallLabel(run.budget_wall_sec)}` : undefined,
    percent: capped ? clampPercent(ratio) : null,
    tone: capped ? ratioTone(ratio) : "neutral",
    displayOnly: false,
    tags: capped ? ["Enforced", budgetPolicyTag(run)] : ["Off", "0 = unlimited"],
  };
}

function costMeter(run: LoopRunRecord): LoopRunMeter {
  return {
    key: "cost",
    label: "Cost",
    value: deriveCostEstimate(run.tokens_used),
    percent: null,
    tone: "neutral",
    displayOnly: true,
    tags: ["Display-only", "estimate · tokens × rate"],
  };
}

function breadthMeter(breadth: number): LoopRunMeter {
  const ceiling = LOOP_CEILINGS.fanOutBreadth;
  const known = breadth > 0;
  return {
    key: "breadth",
    label: "Breadth",
    value: known ? String(breadth) : "—",
    max: `/ ${ceiling}`,
    percent: known ? clampPercent(breadth / ceiling) : null,
    tone: known ? ratioTone(breadth / ceiling) : "neutral",
    displayOnly: false,
    tags: ["batches · this tick"],
  };
}

/**
 * Builds the 5 run-page meters (Attempts / Tokens / Wall / Cost / Breadth) from the
 * run projection. Bars warn-tint only near the ceiling; Cost is display-only with no
 * bar and no cap (truthful UI). `breadth` is the materialized fan-out branch count of
 * the current tick, derived by the caller from the latest generation's outputs.
 */
export function buildRunMeters(run: LoopRunRecord, breadth = 0): LoopRunMeter[] {
  return [
    attemptsMeter(run),
    tokensMeter(run),
    wallMeter(run),
    costMeter(run),
    breadthMeter(breadth),
  ];
}
