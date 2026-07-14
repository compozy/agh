import type { LoopConfig, LoopContract, LoopEffectiveConfig, RunLoopRequest } from "../types";

export type LoopEffectiveConfigView = Pick<
  LoopEffectiveConfig,
  | "human_gate_enabled"
  | "reattempt_strategy"
  | "iteration_cap"
  | "budget_tokens"
  | "budget_wall_sec"
  | "budget_on_exceeded"
  | "no_progress_window"
  | "fan_out_width"
  | "gate_max_revisions"
>;

export type LoopRunConfigOverrides = NonNullable<RunLoopRequest["config_overrides"]>;

const DEFAULT_GATE_MAX_REVISIONS = 3;

/**
 * Projects the saved per-Loop configuration over the definition defaults, then
 * overlays the current per-run request. The daemon remains authoritative after Dry
 * run via the returned `effective_config`; this projection keeps pre-run surfaces
 * aligned with the exact configuration that run creation will receive.
 */
export function resolveLoopEffectiveConfig(
  contract: LoopContract,
  stored: LoopConfig | null,
  runOverrides: LoopRunConfigOverrides | null = null
): LoopEffectiveConfigView {
  const baseline: LoopEffectiveConfigView = {
    human_gate_enabled: stored?.human_gate_enabled ?? false,
    reattempt_strategy: stored?.reattempt_strategy ?? "failed_only",
    iteration_cap: stored?.iteration_cap ?? contract.iteration_cap,
    budget_tokens: stored?.budget_tokens ?? contract.budget.tokens,
    budget_wall_sec: stored?.budget_wall_sec ?? contract.budget.wall_clock_sec,
    budget_on_exceeded:
      (stored?.budget_on_exceeded ?? contract.budget.on_exceeded) === "escalate"
        ? "escalate"
        : "halt",
    no_progress_window: stored?.no_progress_window ?? contract.no_progress.window,
    fan_out_width: stored?.fan_out_width ?? 0,
    gate_max_revisions: stored?.gate_max_revisions ?? DEFAULT_GATE_MAX_REVISIONS,
  };

  return {
    human_gate_enabled: runOverrides?.human_gate_enabled ?? baseline.human_gate_enabled,
    reattempt_strategy: runOverrides?.reattempt_strategy ?? baseline.reattempt_strategy,
    iteration_cap: runOverrides?.iteration_cap ?? baseline.iteration_cap,
    budget_tokens: runOverrides?.budget_tokens ?? baseline.budget_tokens,
    budget_wall_sec: runOverrides?.budget_wall_sec ?? baseline.budget_wall_sec,
    budget_on_exceeded:
      (runOverrides?.budget_on_exceeded ?? baseline.budget_on_exceeded) === "escalate"
        ? "escalate"
        : "halt",
    no_progress_window: runOverrides?.no_progress_window ?? baseline.no_progress_window,
    fan_out_width: runOverrides?.fan_out_width ?? baseline.fan_out_width,
    gate_max_revisions: runOverrides?.gate_max_revisions ?? baseline.gate_max_revisions,
  };
}
