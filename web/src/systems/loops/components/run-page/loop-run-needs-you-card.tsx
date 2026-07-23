import { Check, PenLine, X } from "lucide-react";

import { Button, Eyebrow, StatusCard } from "@agh/ui";

import type { LoopApprovalFact, LoopApprovalRequest } from "../../lib/loop-events";
import type { LoopRunRecord } from "../../types";
import { LoopRunSection } from "./loop-run-section";

/** The closed HITL verdict vocabulary (ADR-017 §9.8): resume / revise / halt. */
export type LoopGateDecision = "approve" | "request_changes" | "reject";

interface LoopRunNeedsYouCardProps {
  run: LoopRunRecord;
  /** The `needs_approval` payload; null before the frame replays. */
  request: LoopApprovalRequest | null;
  /** Usage-snapshot facts standing in when the payload carries none (§3). */
  fallbackFacts: LoopApprovalFact[];
  isPending?: boolean;
  onDecision: (decision: LoopGateDecision, gateId: string) => void;
}

/**
 * The "Needs you" card (§2/§7): a warning scard at the top of the main column
 * with the approver's prompt, the decision facts, and the closed decision set —
 * Approve & resume / Request changes / Reject & halt — wired with the gate id.
 */
export function LoopRunNeedsYouCard({
  run,
  request,
  fallbackFacts,
  isPending,
  onDecision,
}: LoopRunNeedsYouCardProps) {
  const gateId = request?.gateId ?? run.active_gate_id ?? "approve";
  const facts = request?.facts && request.facts.length > 0 ? request.facts : fallbackFacts;
  const micro =
    gateId === "budget"
      ? `needs_approval · ${gateId} · on_exceeded: ${run.budget_on_exceeded}`
      : `needs_approval · ${gateId}`;
  return (
    <LoopRunSection label="Needs you" data-testid="loop-run-needs-you">
      <StatusCard
        className="gap-0 border border-warning/25 bg-warning-tint px-4 py-3.5"
        tone="warning"
      >
        <div className="text-ws-name font-medium text-warning">
          {request?.title ?? "Approve to continue this run?"}
        </div>
        {request?.prompt ? (
          <StatusCard.Body className="mt-1 max-w-[62ch] leading-relaxed">
            {request.prompt}
          </StatusCard.Body>
        ) : (
          <StatusCard.Body className="mt-1 max-w-[62ch] leading-relaxed">
            The run parked and asks you first. Approving continues the round with the same limits;
            rejecting ends the run.
          </StatusCard.Body>
        )}
        {facts.length > 0 ? (
          <div className="mt-3 flex flex-wrap gap-x-5.5 gap-y-2">
            {facts.map(fact => (
              <div key={fact.label} className="flex flex-col gap-0.5" data-testid="loop-run-fact">
                <Eyebrow className="text-faint">{fact.label}</Eyebrow>
                <span className="font-mono text-mono-id tabular-nums text-fg">{fact.value}</span>
              </div>
            ))}
          </div>
        ) : null}
        <StatusCard.Action className="mt-3.5">
          <Button
            data-testid="loop-approval-approve"
            disabled={isPending}
            onClick={() => onDecision("approve", gateId)}
            size="sm"
            type="button"
            variant="primary"
          >
            <Check aria-hidden="true" className="size-3.5" />
            Approve &amp; resume
          </Button>
          <Button
            data-testid="loop-approval-request-changes"
            disabled={isPending}
            onClick={() => onDecision("request_changes", gateId)}
            size="sm"
            type="button"
            variant="outline"
          >
            <PenLine aria-hidden="true" className="size-3.5" />
            Request changes
          </Button>
          <Button
            className="text-danger hover:border-danger/40 hover:text-danger"
            data-testid="loop-approval-reject"
            disabled={isPending}
            onClick={() => onDecision("reject", gateId)}
            size="sm"
            type="button"
            variant="outline"
          >
            <X aria-hidden="true" className="size-3.5" />
            Reject &amp; halt
          </Button>
        </StatusCard.Action>
        <div className="mt-3 font-mono text-pill-group-badge text-faint">{micro}</div>
      </StatusCard>
    </LoopRunSection>
  );
}
