import { ArrowRight, Check, X } from "lucide-react";

import { Eyebrow } from "@agh/ui";

import type { LoopGateVerdict } from "../../lib/loop-events";
import { MonoTag } from "../mono-tag";

interface LoopGateCardProps {
  verdict: LoopGateVerdict;
}

/**
 * A gate verdict card (§4.4): flat tint (pass = success / revise = danger), the
 * verdict + confidence, the per-criterion results, the structured blocking-issue IDs
 * (the stall-detector signature, ADR-022), and the routing edge. Rendered from a
 * `gate_verdict` SSE frame; no side-stripe, no gradient.
 */
export function LoopGateCard({ verdict }: LoopGateCardProps) {
  const passed = verdict.verdict === "pass";
  const tintClass = passed
    ? "border-success/25 bg-success-tint"
    : "border-danger/25 bg-danger-tint";
  const verdictClass = passed ? "text-success" : "text-danger";
  return (
    <div
      className={`mt-2 rounded-md border px-3 py-2.5 ${tintClass}`}
      data-testid="loop-gate-card"
      data-verdict={verdict.verdict}
    >
      <div className={`flex items-center gap-2 text-[12.5px] font-medium ${verdictClass}`}>
        {passed ? (
          <Check className="size-3.5" aria-hidden="true" />
        ) : (
          <X className="size-3.5" aria-hidden="true" />
        )}
        Verdict: {verdict.verdict}
        {verdict.confidence !== undefined ? (
          <span className="font-normal text-muted">
            {" "}
            · confidence {verdict.confidence.toFixed(2)}
          </span>
        ) : null}
      </div>
      {verdict.criteria.length > 0 ? (
        <div className="mt-2.5">
          <Eyebrow className="text-faint">Criteria</Eyebrow>
          <div className="mt-1">
            {verdict.criteria.map(criterion => (
              <div
                key={criterion.id}
                className="flex items-center gap-2 border-t border-line-soft py-1.5 text-[11.5px] first:border-t-0"
              >
                <span className="font-mono text-fg">{criterion.id}</span>
                <MonoTag className="rounded-xs bg-badge-fill px-1.5 py-0.5">
                  {criterion.type}
                </MonoTag>
                {criterion.note ? (
                  <span className="min-w-0 truncate text-muted">{criterion.note}</span>
                ) : null}
                <span
                  className={`ml-auto shrink-0 font-medium ${criterion.status === "pass" ? "text-success" : "text-danger"}`}
                >
                  {criterion.status}
                </span>
              </div>
            ))}
          </div>
        </div>
      ) : null}
      {verdict.blockingIssues.length > 0 ? (
        <div className="mt-2.5">
          <Eyebrow className="text-faint">Blocking issues</Eyebrow>
          <div className="mt-1">
            {verdict.blockingIssues.map(issue => (
              <div
                key={issue.id}
                className="flex items-start gap-2 border-t border-line-soft py-1.5 text-[11.5px] first:border-t-0"
              >
                <span className="shrink-0 font-mono text-fg">{issue.id}</span>
                {issue.note ? <span className="min-w-0 text-muted">{issue.note}</span> : null}
              </div>
            ))}
          </div>
        </div>
      ) : null}
      {verdict.reason ? (
        <p className="mt-2 text-[12px] leading-relaxed text-muted">{verdict.reason}</p>
      ) : null}
      {verdict.route ? (
        <div className="mt-2.5 inline-flex items-center gap-1.5 rounded-sm border border-line-soft bg-input-fill px-2 py-1 text-[11.5px] text-subtle">
          <ArrowRight className="size-3" aria-hidden="true" />
          Routed to <b className="font-medium text-accent-strong">{verdict.route}</b>
        </div>
      ) : null}
    </div>
  );
}
