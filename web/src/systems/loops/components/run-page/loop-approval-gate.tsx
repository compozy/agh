import { Check, PenLine, X } from "lucide-react";

import { Button, Eyebrow } from "@agh/ui";

import type { LoopApprovalRequest } from "../../lib/loop-events";
import type { LoopRunRecord } from "../../types";
import { formatRunInputs } from "../../lib/loop-runs-view";
import { MonoTag } from "../mono-tag";

/** The closed HITL verdict vocabulary (ADR-017 §9.8): resume / revise / halt. */
export type LoopGateDecision = "approve" | "request_changes" | "reject";

interface LoopApprovalGateProps {
  run: LoopRunRecord;
  /** The `needs_approval` payload (approver context + facts); null before it streams. */
  request?: LoopApprovalRequest | null;
  isPending?: boolean;
  onDecision: (decision: LoopGateDecision, gateId: string) => void;
}

/**
 * The human approval gate (§4.4 / ADR-005/017): the live `needs-approval` pause with
 * the approver's context, the decision facts, and the closed action set — Approve &
 * resume, Request changes, Reject & halt — routing to resume / revise-next-generation /
 * terminal halt. Facts come from the `needs_approval` payload; when it has not streamed
 * yet, the run's resolved inputs stand in (truthful — no fabricated diff/test facts).
 */
export function LoopApprovalGate({ run, request, isPending, onDecision }: LoopApprovalGateProps) {
  const gateId = request?.gateId ?? "approve";
  const title = request?.title ?? "Approve to resume this run?";
  const inputPreview = formatRunInputs(run.inputs, 4);
  const facts =
    request?.facts && request.facts.length > 0
      ? request.facts
      : inputPreview
        ? [{ label: "Inputs", value: inputPreview }]
        : [];
  return (
    <section
      className="rounded-lg border border-info/30 bg-info-tint px-4 py-4"
      data-testid="loop-approval-gate"
    >
      <div className="flex flex-wrap items-center gap-2.5">
        <MonoTag className="rounded-xs border border-info/30 bg-info-tint px-1.5 py-0.5 text-info">
          needs-approval
        </MonoTag>
        <h2 className="text-[14px] font-medium text-fg-strong">{title}</h2>
      </div>
      {request?.prompt ? (
        <p className="mt-2 max-w-[68ch] text-[12.5px] leading-relaxed text-muted">
          {request.prompt}
        </p>
      ) : (
        <p className="mt-2 max-w-[68ch] text-[12.5px] leading-relaxed text-muted">
          This run is paused at a human gate. It stays a live{" "}
          <b className="text-info">needs-approval</b> state — not a finish — and resumes the moment
          you decide.
        </p>
      )}
      {facts.length > 0 ? (
        <div className="mt-3.5 flex flex-wrap gap-x-6 gap-y-2">
          {facts.map(fact => (
            <div
              key={fact.label}
              className="flex flex-col gap-0.5"
              data-testid="loop-approval-fact"
            >
              <Eyebrow className="text-faint">{fact.label}</Eyebrow>
              <span className="font-mono text-[11.5px] text-fg">{fact.value}</span>
            </div>
          ))}
        </div>
      ) : null}
      <div className="mt-4 flex flex-wrap gap-2">
        <Button
          type="button"
          variant="primary"
          size="sm"
          data-testid="loop-approval-approve"
          disabled={isPending}
          onClick={() => onDecision("approve", gateId)}
        >
          <Check className="size-3.5" aria-hidden="true" />
          Approve &amp; resume
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid="loop-approval-request-changes"
          disabled={isPending}
          onClick={() => onDecision("request_changes", gateId)}
        >
          <PenLine className="size-3.5" aria-hidden="true" />
          Request changes
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="text-danger hover:border-danger/40 hover:text-danger"
          data-testid="loop-approval-reject"
          disabled={isPending}
          onClick={() => onDecision("reject", gateId)}
        >
          <X className="size-3.5" aria-hidden="true" />
          Reject &amp; halt
        </Button>
      </div>
    </section>
  );
}
