import { AlertCircle, History, Loader2, RotateCcw } from "lucide-react";

import { Button, Empty, Pill, Section, Spinner } from "@agh/ui";

import {
  decisionOpLabel,
  decisionSourceLabel,
  formatKnowledgeDateTime,
} from "@/systems/knowledge/lib/knowledge-formatters";
import type { MemoryDecision } from "@/systems/knowledge/types";

import { pillToneFromDecisionOp, pillToneFromDecisionSource } from "./knowledge-pill-tone";

interface KnowledgeDecisionsSectionProps {
  decisions: MemoryDecision[] | undefined;
  isLoading: boolean;
  error: Error | null;
  onRevertDecision?: (decision: MemoryDecision) => Promise<void>;
  revertingDecisionId?: string | null;
  revertError?: string | null;
}

function KnowledgeDecisionsSection({
  decisions,
  isLoading,
  error,
  onRevertDecision,
  revertingDecisionId = null,
  revertError = null,
}: KnowledgeDecisionsSectionProps) {
  const handleRevert = async (decision: MemoryDecision) => {
    if (!onRevertDecision) return;
    try {
      await onRevertDecision(decision);
    } catch {
      // Error state is surfaced through `revertError`.
    }
  };

  return (
    <Section data-testid="knowledge-decisions-section" label="Recent controller decisions">
      {isLoading ? (
        <div
          className="flex items-center gap-2 px-1 py-3 text-xs text-(--subtle)"
          data-testid="knowledge-decisions-loading"
        >
          <Spinner /> Loading decisions…
        </div>
      ) : error ? (
        <Empty
          className="max-w-md"
          data-testid="knowledge-decisions-error"
          description={error.message ?? "Failed to load decisions"}
          icon={AlertCircle}
          title="Unable to load decisions"
        />
      ) : !decisions || decisions.length === 0 ? (
        <Empty
          className="max-w-md"
          data-testid="knowledge-decisions-empty"
          description="No controller decisions recorded for this memory yet."
          icon={History}
          title="No decisions"
        />
      ) : (
        <ul
          className="flex flex-col divide-y divide-(--line) rounded-lg border border-(--line) bg-(--canvas-soft)"
          data-testid="knowledge-decisions-list"
        >
          {decisions.map(decision => (
            <li
              className="flex flex-col gap-1.5 px-4 py-3"
              data-testid={`knowledge-decision-${decision.id}`}
              key={decision.id}
            >
              <div className="flex flex-wrap items-center gap-2">
                <Pill
                  mono
                  data-testid={`knowledge-decision-op-${decision.id}`}
                  tone={pillToneFromDecisionOp(decision.op)}
                >
                  {decisionOpLabel(decision.op)}
                </Pill>
                <Pill
                  mono
                  data-testid={`knowledge-decision-source-${decision.id}`}
                  tone={pillToneFromDecisionSource(decision.source)}
                >
                  {decisionSourceLabel(decision.source)}
                </Pill>
                <span className="ml-auto font-mono text-eyebrow uppercase tracking-badge text-(--subtle)">
                  {formatKnowledgeDateTime(decision.decided_at)}
                </span>
                {onRevertDecision && decision.applied_at ? (
                  <Button
                    data-testid={`revert-memory-decision-${decision.id}`}
                    disabled={revertingDecisionId === decision.id}
                    onClick={() => void handleRevert(decision)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    {revertingDecisionId === decision.id ? (
                      <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />
                    ) : (
                      <RotateCcw className="size-3.5" />
                    )}
                    Revert
                  </Button>
                ) : null}
              </div>
              {decision.reason ? <p className="text-xs text-(--muted)">{decision.reason}</p> : null}
              <div className="flex flex-wrap items-center gap-3 font-mono text-badge uppercase tracking-badge text-(--subtle)">
                <span data-testid={`knowledge-decision-confidence-${decision.id}`}>
                  Confidence {decision.confidence.toFixed(2)}
                </span>
                {decision.applied_at ? (
                  <span data-testid={`knowledge-decision-applied-${decision.id}`}>
                    Applied {formatKnowledgeDateTime(decision.applied_at)}
                  </span>
                ) : (
                  <span data-testid={`knowledge-decision-pending-${decision.id}`}>Not applied</span>
                )}
                {decision.target_filename ? (
                  <span data-testid={`knowledge-decision-target-${decision.id}`}>
                    Target {decision.target_filename}
                  </span>
                ) : null}
              </div>
              {revertError && revertingDecisionId === decision.id ? (
                <p
                  className="text-small-body text-(--danger)"
                  data-testid={`knowledge-decision-revert-error-${decision.id}`}
                >
                  {revertError}
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}

export { KnowledgeDecisionsSection };
export type { KnowledgeDecisionsSectionProps };
