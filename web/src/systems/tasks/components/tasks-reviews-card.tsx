import { AlertCircle, Gavel } from "lucide-react";

import {
  BlockLoading,
  Empty,
  MonoId,
  Pill,
  Section,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  type PillTone,
} from "@agh/ui";

import { formatRelativeTime } from "../lib/task-formatters";
import type { TaskRunReview, TaskRunReviewOutcome, TaskRunReviewStatus } from "../types";

export interface TasksReviewsCardProps {
  reviews: TaskRunReview[];
  isLoading?: boolean;
  errorMessage?: string | null;
  /** Title for the card; defaults to "Reviews". */
  label?: string;
  /** Test id prefix used for review rows. */
  testIdPrefix?: string;
  /**
   * Optional list-level test id, defaults to `"tasks-reviews-card"`. Tests for
   * the run-level reviews variant override this to `"tasks-run-reviews-card"`.
   */
  testId?: string;
}

const REVIEW_STATUS_TONE: Record<TaskRunReviewStatus, PillTone> = {
  requested: "info",
  routed: "info",
  in_review: "accent",
  recorded: "neutral",
  circuit_opened: "warning",
  canceled: "neutral",
};

const REVIEW_OUTCOME_TONE: Record<TaskRunReviewOutcome, PillTone> = {
  approved: "success",
  rejected: "danger",
  blocked: "warning",
  error: "danger",
  timeout: "warning",
  invalid_output: "danger",
};

function reviewStatusTone(status: TaskRunReviewStatus | undefined): PillTone {
  if (!status) {
    return "neutral";
  }
  return REVIEW_STATUS_TONE[status] ?? "neutral";
}

function reviewOutcomeTone(outcome: TaskRunReviewOutcome | undefined): PillTone {
  if (!outcome) {
    return "neutral";
  }
  return REVIEW_OUTCOME_TONE[outcome] ?? "neutral";
}

function formatMissingWorkCount(missingWork: TaskRunReview["missing_work"]): number {
  if (Array.isArray(missingWork)) {
    return missingWork.length;
  }
  return 0;
}

export function TasksReviewsCard({
  reviews,
  isLoading = false,
  errorMessage = null,
  label = "Reviews",
  testIdPrefix = "tasks-reviews-row",
  testId = "tasks-reviews-card",
}: TasksReviewsCardProps) {
  if (isLoading && reviews.length === 0) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={`${testId}-loading-section`}
        label={label}
      >
        <BlockLoading
          label={`Loading ${label}`}
          size="sm"
          surface="bare"
          data-testid={`${testId}-loading`}
        />
      </Section>
    );
  }

  if (errorMessage && reviews.length === 0) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={testId}
        label={label}
      >
        <Empty
          data-testid={`${testId}-error`}
          description={errorMessage}
          icon={AlertCircle}
          title="Unable to load reviews"
        />
      </Section>
    );
  }

  if (reviews.length === 0) {
    return (
      <Section
        aria-label={label}
        bodyClassName="gap-4"
        className="w-full gap-4"
        data-testid={testId}
        label={label}
      >
        <Empty
          data-testid={`${testId}-empty`}
          description="Review-on-stop is off or no reviews have been requested yet. Verdicts are recorded only after a reviewer-bound session submits through submit_run_review."
          icon={Gavel}
          title="No reviews"
        />
      </Section>
    );
  }

  return (
    <Section
      aria-label={label}
      bodyClassName="gap-4"
      className="w-full gap-4"
      count={reviews.length}
      data-testid={testId}
      icon={Gavel}
      label={label}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Review</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Outcome</TableHead>
            <TableHead>Reviewer</TableHead>
            <TableHead>Round</TableHead>
            <TableHead>Requested</TableHead>
            <TableHead>Reviewed</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {reviews.map(review => {
            const status = review.status as TaskRunReviewStatus | undefined;
            const outcome = review.outcome as TaskRunReviewOutcome | undefined;
            const missingWorkCount = formatMissingWorkCount(review.missing_work);
            return (
              <TableRow data-testid={`${testIdPrefix}-${review.review_id}`} key={review.review_id}>
                <TableCell className="max-w-80">
                  <div className="flex min-w-0 flex-col gap-1.5">
                    <MonoId value={review.review_id} />
                    <span className="font-mono text-mono-id text-faint">run {review.run_id}</span>
                    {review.reason ? (
                      <p
                        className="line-clamp-2 text-form-label leading-relaxed text-muted"
                        data-testid={`${testIdPrefix}-${review.review_id}-reason`}
                      >
                        {review.reason}
                      </p>
                    ) : null}
                    {review.next_round_guidance ? (
                      <p
                        className="line-clamp-3 rounded-md bg-input-fill px-2 py-1.5 text-form-hint leading-relaxed text-muted"
                        data-testid={`${testIdPrefix}-${review.review_id}-guidance`}
                      >
                        {review.next_round_guidance}
                      </p>
                    ) : null}
                    {missingWorkCount > 0 ? (
                      <span
                        className="text-eyebrow text-warning"
                        data-testid={`${testIdPrefix}-${review.review_id}-missing-work`}
                      >
                        Missing work · {missingWorkCount}
                      </span>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell>
                  {status ? (
                    <Pill
                      data-testid={`${testIdPrefix}-${review.review_id}-status`}
                      tone={reviewStatusTone(status)}
                    >
                      {status}
                    </Pill>
                  ) : (
                    <span className="text-form-label text-faint">--</span>
                  )}
                </TableCell>
                <TableCell>
                  {outcome ? (
                    <Pill
                      data-testid={`${testIdPrefix}-${review.review_id}-outcome`}
                      tone={reviewOutcomeTone(outcome)}
                    >
                      {outcome}
                    </Pill>
                  ) : (
                    <span
                      className="text-form-label text-faint"
                      data-testid={`${testIdPrefix}-${review.review_id}-outcome-pending`}
                    >
                      pending
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <span className="text-form-label text-fg">
                      {review.reviewer_agent_name ?? "--"}
                    </span>
                    {review.reviewer_session_id ? (
                      <span className="font-mono text-mono-id text-faint">
                        session {review.reviewer_session_id}
                      </span>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell>
                  <span className="font-mono text-eyebrow tabular-nums text-muted">
                    round {review.review_round} · attempt {review.attempt}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-form-label text-faint">
                    {formatRelativeTime(review.requested_at)}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-form-label text-faint">
                    {review.outcome ? formatRelativeTime(review.reviewed_at) : "--"}
                  </span>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Section>
  );
}
