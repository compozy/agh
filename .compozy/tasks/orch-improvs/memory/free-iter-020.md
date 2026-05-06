# Free Iteration 020 Memory

## Slice

- Add task run review verdict recording with approved/rejected/blocked events and idempotent continuation-run creation for rejected reviews.

## Acceptance Cursor

- TechSpec anchor: review-gate runtime authority in `task.Service`, especially `RecordRunReview`, verdict validation, review rollup persistence, and continuation-run creation on rejected reviews.
- Scope for this iteration: backend/domain/store only.
- Out of scope for this iteration: native `submit_run_review` tool, HTTP/UDS/CLI surfaces, web/site docs, bundled skills, QA pair, and CodeRabbit rounds.

## Implementation Notes

- Added verdict domain types and validation in `internal/task/review.go`, including required `delivery_id`, bounded reason/guidance text, supported verdict outcomes, approved-review empty `missing_work`, and rejected-review continuation guidance requirements.
- Added task-service `RecordRunReview` authority in `internal/task/manager_review.go`; store writes happen first, then review recorded/outcome/retry events are emitted through the existing task event path.
- Added GlobalDB verdict persistence in `internal/store/globaldb/global_db_task_review.go`: recorded verdict fields, task review rollup fields, reviewer-session binding enforcement, same-verdict replay idempotency, and rejected-review continuation run creation.
- Continuation runs are idempotent through `task_runs.review_id`; replaying the same verdict with the same delivery id returns the existing continuation instead of enqueueing a duplicate.
- `task.Run` now carries review-gate run lineage under `Run.Review *RunReviewLineage` instead of large direct fields. This keeps review data typed while avoiding `gocritic` value-copy churn across hot task-run paths.
- Added focused task manager and GlobalDB tests for approved verdicts, rejected verdict continuation runs, event emission, and replay behavior.

## Verification

- `go test ./internal/task ./internal/store/globaldb -run 'TestRunReviewValidation|TestTaskManagerRunReviews|TestGlobalDBTaskRunReviewStore' -count=1` passed.
- `go test ./internal/task ./internal/store/globaldb ./internal/api/testutil ./internal/daemon -count=1` passed.
- `go test -race ./internal/task -run 'TestRunReviewValidation|TestTaskManagerRunReviews' -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -count=1` passed in `196.785s`.
- `make lint` passed with `0 issues`.
- Final `make verify` passed: Bun lint/typecheck/test passed, Vitest passed `329` files / `2088` tests, web build passed, `golangci-lint` reported `0 issues`, Go race gate completed `8198` tests in `182.159s`, and package boundaries passed.

## Open Risks

- Native `submit_run_review` tool, HTTP/UDS/CLI surfaces, web/site docs, bundled skills, QA pair, and CodeRabbit rounds remain outside this slice.
- Review verdict authority now exists, but reviewer routing and session-start profile enforcement still need later slices before the full review-gate lifecycle is operator-visible end to end.
