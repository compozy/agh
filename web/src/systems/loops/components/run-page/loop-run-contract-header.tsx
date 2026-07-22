import type { ReactNode } from "react";

import { Eyebrow, type PillTone } from "@agh/ui";

import { loopStatusLabel, loopStatusTone } from "../../lib/loop-formatters";
import type { LoopContract, LoopContractVerification, LoopRunRecord } from "../../types";
import { MonoTag } from "../mono-tag";

interface LoopRunContractHeaderProps {
  run: LoopRunRecord;
  /** The loop's declared contract (from `useLoop`); undefined while it loads. */
  contract?: LoopContract;
  /** The 5-meter strip, wired by the route. */
  meters?: ReactNode;
}

const TONE_DOT_CLASS: Record<PillTone, string> = {
  neutral: "bg-neutral",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

/** Compact method line for one verification criterion (mirrors the contract panel). */
function verificationMethod(criterion: LoopContractVerification): string {
  const parts: string[] = [];
  if (criterion.check) parts.push(criterion.check);
  if (criterion.expect) parts.push(`expect ${criterion.expect}`);
  if (criterion.agent) parts.push(`agent ${criterion.agent}`);
  if (criterion.tool) parts.push(`tool ${criterion.tool}`);
  if (criterion.rubric) parts.push(criterion.rubric);
  if (criterion.prompt) parts.push(criterion.prompt);
  return parts.join(" · ");
}

function reattemptLabel(strategy: string): string {
  return strategy === "full_body" ? "full-body re-attempt" : "failed-only re-attempt";
}

/**
 * The run contract summary: generation metadata, goal, definition of done,
 * verification rows, terminal outcomes, and resource meters. Window identity,
 * live status, and operator actions belong to the OS topbar publisher.
 */
export function LoopRunContractHeader({ run, contract, meters }: LoopRunContractHeaderProps) {
  const verification = contract?.verification ?? [];
  const terminalStates = contract?.terminal_states ?? [];
  const startOrigin = run.started_origin_kind ? `start ${run.started_origin_kind}` : null;
  return (
    <section
      className="flex flex-col gap-4 border-b border-line bg-canvas-soft px-6 py-4"
      data-testid="loop-run-contract-header"
    >
      <div className="flex items-start gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2.5 text-small-body text-subtle">
            <span>
              Generation <b className="font-medium text-fg-strong">{run.generation}</b>
              {run.iteration_cap > 0 ? ` of ${run.iteration_cap}` : " · unbounded"} ·{" "}
              {reattemptLabel(run.reattempt_strategy)}
            </span>
            {startOrigin ? (
              <span className="font-mono text-mono-id text-muted">{startOrigin}</span>
            ) : null}
          </div>
          <h2
            className="mt-2.5 max-w-[78ch] text-item-title leading-snug font-medium tracking-[-0.016em] text-fg-strong"
            data-testid="loop-run-goal"
          >
            {contract?.goal ?? `Run ${run.loop_name}`}
          </h2>
        </div>
      </div>

      {contract ? (
        <div className="flex flex-col gap-3" data-testid="loop-run-contract-detail">
          <div>
            <Eyebrow className="text-faint">Definition of done</Eyebrow>
            <p
              className="mt-1 max-w-[78ch] text-small-body leading-relaxed text-muted"
              data-testid="loop-run-dod"
            >
              {contract.definition_of_done}
            </p>
          </div>
          {verification.length > 0 ? (
            <div>
              <Eyebrow className="text-faint">Verification</Eyebrow>
              <div className="mt-1.5 flex flex-col gap-1.5">
                {verification.map(criterion => (
                  <div
                    key={criterion.id}
                    className="flex items-start gap-2.5"
                    data-testid="loop-run-verification-row"
                  >
                    <MonoTag className="min-w-[84px] shrink-0 justify-center rounded-xs bg-badge-fill px-1.5 py-0.5">
                      {criterion.type}
                    </MonoTag>
                    <div className="min-w-0">
                      <b className="text-form-label font-medium text-fg-strong">{criterion.id}</b>
                      <span className="ml-2 font-mono text-mono-id text-subtle">
                        {verificationMethod(criterion) || "—"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
          {terminalStates.length > 0 ? (
            <div className="flex flex-wrap gap-1.5" data-testid="loop-run-terminal-chips">
              {terminalStates.map(state => (
                <span
                  key={state}
                  className="inline-flex items-center gap-1.5 rounded-md border border-line-soft bg-canvas-tint px-2.5 py-1 text-form-hint text-muted"
                  data-testid="loop-run-terminal-chip"
                >
                  <span
                    aria-hidden="true"
                    className={`size-1.5 rounded-full ${TONE_DOT_CLASS[loopStatusTone(state)]}`}
                  />
                  {loopStatusLabel(state)}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}

      {meters}
    </section>
  );
}
