import { StatusCard, Time, type PillTone } from "@agh/ui";

import type { TaskRunReview } from "../types";

const OUTCOME_TONE: Record<string, PillTone> = {
  approved: "success",
  rejected: "warning",
  blocked: "danger",
  error: "danger",
  timeout: "danger",
};

function titleFor(review: TaskRunReview): string {
  const round = review.review_round ? `Review round ${review.review_round}` : "Review";
  const outcome = review.outcome ?? review.status;
  return `${round}: ${outcome.replaceAll("_", " ")}`;
}

/**
 * Run-detail review card (§4.9): one StatusCard per review round — outcome
 * tone, reason, and next-round guidance in plain language.
 */
export function TaskRunReviewCard({ review }: { review: TaskRunReview }) {
  const tone = review.outcome ? (OUTCOME_TONE[review.outcome] ?? "neutral") : "info";

  return (
    <StatusCard
      className="border border-line-soft"
      data-testid={`tasks-run-review-${review.review_id}`}
      tone={tone}
    >
      <StatusCard.Header label={titleFor(review)}>
        {review.reviewed_at ? (
          <span className="ml-auto shrink-0 text-eyebrow tabular-nums text-subtle">
            <Time iso={review.reviewed_at} mode="relative" />
          </span>
        ) : null}
      </StatusCard.Header>
      {review.reason || review.review_text ? (
        <StatusCard.Body>{review.reason ?? review.review_text}</StatusCard.Body>
      ) : null}
      {review.next_round_guidance ? (
        <StatusCard.Body className="text-subtle">
          Next round: {review.next_round_guidance}
        </StatusCard.Body>
      ) : null}
      {review.reviewer_agent_name ? (
        <StatusCard.Footer>
          <span className="text-form-hint text-subtle">
            Reviewed by {review.reviewer_agent_name}
          </span>
        </StatusCard.Footer>
      ) : null}
    </StatusCard>
  );
}
