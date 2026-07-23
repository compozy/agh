import { Check, RotateCcw } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { Button, buttonVariants, StatusCard, type StatusCardTone } from "@agh/ui";

import type { LoopCoordinatorFailure } from "../../lib/loop-events";
import type { LoopRunRecord } from "../../types";
import { LoopRunSection } from "./loop-run-section";

interface LoopRunOutcomeCardProps {
  run: LoopRunRecord;
  /** Run-level failure from the terminal status frame; null when none streamed. */
  failure: LoopCoordinatorFailure | null;
  /** Frozen run duration (`26m 41s`). */
  durationLabel: string;
  /** The terminal transition's `from` status, read from the retained frame. */
  fromStatus?: string;
  /** `contract.no_progress.window` from the pinned definition (stalled card). */
  noProgressWindow?: number;
  /** Blocking-issue ids from the last check (stalled card evidence). */
  repeatedIssueIds: string[];
  onStartNewRun: () => void;
  isStartPending?: boolean;
}

interface OutcomeView {
  sectionLabel: string | null;
  tone: StatusCardTone;
  cardClass: string;
  titleClass: string;
  title: string;
  body?: string;
  recovery?: string;
  showStartNewRun: boolean;
}

/** The exhausted card names the limit that tripped, from the run's own caps. */
function exhaustedLimit(run: LoopRunRecord): string {
  if (run.iteration_cap > 0 && run.generation >= run.iteration_cap) return "Round cap";
  if (run.budget_tokens > 0 && run.tokens_used >= run.budget_tokens) return "Token budget";
  if (run.budget_wall_sec > 0) return "Time limit";
  return "A limit";
}

function outcomeView(props: LoopRunOutcomeCardProps): OutcomeView | null {
  const { run, failure, durationLabel, noProgressWindow } = props;
  switch (run.status) {
    case "failed":
      return {
        sectionLabel: "What went wrong",
        tone: "danger",
        cardClass: "border border-danger/25 bg-danger-tint",
        titleClass: "text-danger",
        title: `Failed after ${durationLabel}`,
        body: failure?.cause,
        recovery: failure?.recovery,
        showStartNewRun: true,
      };
    case "blocked":
      return {
        sectionLabel: "Why it stopped",
        tone: "warning",
        cardClass: "border border-warning/25 bg-warning-tint",
        titleClass: "text-warning",
        title: "Blocked on something outside the run",
        body:
          failure?.cause ?? "The run stopped on an external dependency it cannot resolve itself.",
        recovery: failure?.recovery,
        showStartNewRun: false,
      };
    case "exhausted":
      return {
        sectionLabel: "Why it stopped",
        tone: "warning",
        cardClass: "border border-warning/25 bg-warning-tint",
        titleClass: "text-warning",
        title: `${exhaustedLimit(run)} reached after ${durationLabel}`,
        body: "This loop's limit policy stops the run as exhausted instead of asking. Finished work is kept.",
        recovery: failure?.recovery,
        showStartNewRun: true,
      };
    case "stalled":
      return {
        sectionLabel: "Why it stopped",
        tone: "neutral",
        cardClass: "border border-line bg-canvas-tint",
        titleClass: "text-fg-strong",
        title: "Stalled — no progress",
        body: noProgressWindow
          ? `No new progress across ${noProgressWindow} rounds — the same open points kept coming back.`
          : "No new progress across recent rounds — the same open points kept coming back.",
        recovery: failure?.recovery,
        showStartNewRun: false,
      };
    default:
      return null;
  }
}

/**
 * The terminal outcome card (§7): "What went wrong" for failed runs plus the
 * blocked / exhausted / stalled explainers and the quiet no-op note. `cause` and
 * `recovery` come from the terminal status frame — never invented; "Start a new
 * run" re-posts the same inputs as a fresh run.
 */
export function LoopRunOutcomeCard(props: LoopRunOutcomeCardProps) {
  const { run, failure, fromStatus, repeatedIssueIds, onStartNewRun, isStartPending } = props;
  if (run.status === "no-op") {
    return (
      <div
        className="flex items-start gap-2.25 rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted"
        data-testid="loop-run-noop-note"
      >
        <Check aria-hidden="true" className="mt-0.5 size-3.5 shrink-0 text-subtle" />
        <span>Ran, nothing to do — the run finished without changes.</span>
      </div>
    );
  }
  const view = outcomeView(props);
  if (!view) return null;
  const vaultRecovery = view.recovery !== undefined && /vault/i.test(view.recovery);
  const micro = [
    `status_changed · ${fromStatus ?? "?"} → ${run.status}`,
    failure ? `cause ${failure.code}` : null,
  ]
    .filter(Boolean)
    .join(" · ");
  const card = (
    <StatusCard className={`gap-0 px-4 py-3.5 ${view.cardClass}`} tone={view.tone}>
      <div
        className={`text-ws-name font-medium ${view.titleClass}`}
        data-testid="loop-run-outcome-title"
      >
        {view.title}
      </div>
      {view.body ? (
        <StatusCard.Body className="mt-1 max-w-[62ch] leading-relaxed">{view.body}</StatusCard.Body>
      ) : null}
      {view.recovery ? (
        <StatusCard.Body className="mt-2 max-w-[62ch] leading-relaxed">
          {view.recovery}
        </StatusCard.Body>
      ) : null}
      {run.status === "stalled" && repeatedIssueIds.length > 0 ? (
        <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1" data-testid="loop-run-stalled-issues">
          {repeatedIssueIds.map(id => (
            <span key={id} className="font-mono text-mono-id text-subtle">
              {id}
            </span>
          ))}
        </div>
      ) : null}
      {view.showStartNewRun || vaultRecovery ? (
        <StatusCard.Action className="mt-3.5">
          {view.showStartNewRun ? (
            <Button
              data-testid="loop-run-start-new"
              disabled={isStartPending}
              onClick={onStartNewRun}
              size="sm"
              type="button"
              variant="primary"
            >
              <RotateCcw aria-hidden="true" className="size-3.5" />
              Start a new run
            </Button>
          ) : null}
          {vaultRecovery ? (
            <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/vault">
              Open Vault
            </Link>
          ) : null}
        </StatusCard.Action>
      ) : null}
      <div className="mt-3 font-mono text-pill-group-badge text-faint">{micro}</div>
    </StatusCard>
  );
  if (!view.sectionLabel) return card;
  return (
    <LoopRunSection label={view.sectionLabel} data-testid="loop-run-outcome">
      {card}
    </LoopRunSection>
  );
}
