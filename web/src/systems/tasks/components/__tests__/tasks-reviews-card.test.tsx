import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksReviewsCard } from "../tasks-reviews-card";
import { buildTaskRunReviewFixture } from "../../mocks/fixtures";
import type { TaskRunReview } from "../../types";

describe("TasksReviewsCard", () => {
  it("renders empty state when no reviews exist", () => {
    render(<TasksReviewsCard reviews={[]} />);
    expect(screen.getByTestId("tasks-reviews-card-empty")).toBeInTheDocument();
  });

  it("renders error state when fetch fails", () => {
    render(<TasksReviewsCard errorMessage="boom" reviews={[]} />);
    expect(screen.getByTestId("tasks-reviews-card-error")).toHaveTextContent("boom");
  });

  it("renders status pill but never surfaces approved/rejected as statuses", () => {
    const reviews: TaskRunReview[] = [
      buildTaskRunReviewFixture({ review_id: "review_a", status: "in_review" }),
      buildTaskRunReviewFixture({
        review_id: "review_b",
        status: "recorded",
        outcome: "rejected",
        next_round_guidance: "Add reconciliation evidence.",
      }),
    ];
    render(<TasksReviewsCard reviews={reviews} />);
    expect(screen.getByTestId("tasks-reviews-row-review_a-status")).toHaveTextContent("in_review");
    expect(screen.getByTestId("tasks-reviews-row-review_b-status")).toHaveTextContent("recorded");
    expect(screen.getByTestId("tasks-reviews-row-review_b-outcome")).toHaveTextContent("rejected");
    expect(screen.getByTestId("tasks-reviews-row-review_b-guidance")).toHaveTextContent(
      "Add reconciliation evidence."
    );
  });

  it("indicates pending outcomes for in_review reviews", () => {
    const reviews: TaskRunReview[] = [
      buildTaskRunReviewFixture({ review_id: "review_pending", status: "in_review" }),
    ];
    render(<TasksReviewsCard reviews={reviews} />);
    expect(
      screen.getByTestId("tasks-reviews-row-review_pending-outcome-pending")
    ).toHaveTextContent("pending");
  });

  it("surfaces missing-work counts when reviewer-supplied items are present", () => {
    const reviews: TaskRunReview[] = [
      buildTaskRunReviewFixture({
        review_id: "review_mw",
        status: "recorded",
        outcome: "rejected",
        missing_work: ["evidence-a", "evidence-b"],
      }),
    ];
    render(<TasksReviewsCard reviews={reviews} />);
    expect(screen.getByTestId("tasks-reviews-row-review_mw-missing-work")).toHaveTextContent(
      "Missing work · 2"
    );
  });
});
