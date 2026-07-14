import { Eyebrow } from "@agh/ui";

import {
  resolveLoopEffectiveConfig,
  type LoopRunConfigOverrides,
} from "../../lib/loop-effective-config";
import type { LoopConfig, LoopContract, LoopDryRunPreview, LoopInputSchema } from "../../types";
import { LoopContractPanel } from "../detail/loop-contract-panel";
import { LoopRunPlan } from "./loop-run-plan";

interface LoopRunPreviewProps {
  loopName: string;
  contract: LoopContract;
  config: LoopConfig | null;
  configOverrides: LoopRunConfigOverrides | null;
  inputs?: LoopInputSchema;
  /** Present only after a successful Dry run. */
  plan?: LoopDryRunPreview | null;
}

/**
 * The run-form right rail (§4.3): a live "what will run" summary, the read-only
 * contract (goal, DoD, verification, terminal chips — reused from the loop detail
 * surface), the lifecycle line, and — after Dry run — the resolved gen-1 plan.
 */
export function LoopRunPreview({
  loopName,
  contract,
  config,
  configOverrides,
  inputs,
  plan,
}: LoopRunPreviewProps) {
  const inputCount = inputs ? Object.keys(inputs).length : 0;
  const effective =
    plan?.effective_config ?? resolveLoopEffectiveConfig(contract, config, configOverrides);
  const reattemptLabel = effective.reattempt_strategy === "full_body" ? "full-body" : "failed-only";
  return (
    <aside className="flex flex-col gap-3.5" data-testid="loop-run-preview">
      <Eyebrow className="text-subtle">Run preview</Eyebrow>
      <div className="rounded-lg border border-line-soft bg-canvas-soft px-4 py-3.5">
        <Eyebrow className="text-faint">What will run</Eyebrow>
        <p className="mt-1.5 text-[12.5px] leading-relaxed text-muted">
          Run <b className="font-medium text-fg-strong">{loopName}</b> against its contract:{" "}
          {contract.goal} Iterates up to{" "}
          <b className="font-medium text-fg-strong">
            {effective.iteration_cap === 0 ? "∞" : effective.iteration_cap}
          </b>{" "}
          generations with {reattemptLabel} re-attempts from {inputCount} declared input
          {inputCount === 1 ? "" : "s"}, ending honestly at one of its terminal outcomes.
        </p>
      </div>
      <LoopContractPanel contract={contract} />
      <div className="rounded-lg border border-line-soft bg-canvas-soft px-4 py-3.5">
        <Eyebrow className="text-faint">Lifecycle</Eyebrow>
        <div className="mt-2 flex flex-wrap items-center gap-2 font-mono text-[11.5px] text-subtle">
          <span className="text-muted">running</span>
          <span className="text-faint">→</span>
          <span className="text-muted">verify</span>
          {/* needs-approval only appears when the loop actually declares a human gate —
              a loop with no human criterion can never enter it (truthful lifecycle, §4.3). */}
          {effective.human_gate_enabled ? (
            <>
              <span className="text-faint">→</span>
              <span className="text-warning">needs-approval</span>
            </>
          ) : null}
          <span className="text-faint">→</span>
          <span className="text-success">done</span>
        </div>
      </div>
      {plan ? <LoopRunPlan plan={plan} /> : null}
    </aside>
  );
}
