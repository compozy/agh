# Free Iteration 018 - Review Request Authority Slice

## Slice

Add task run review domain types, GlobalDB review CRUD, and task manager review request/binding authority over `task_run_reviews`.

## Scope

- Add typed review-gate domain values in `internal/task`.
- Add a narrowed `RunReviewStore` surface and wire it into `task.Store`.
- Add GlobalDB transactional persistence for review request idempotency, session binding, lookup, and listing.
- Add `task.Service` methods for request, bind, session lookup, and list.
- Add focused domain/store tests.

## Out of Scope

- `RecordRunReview`, continuation-run creation, native `submit_run_review`, HTTP/UDS/CLI surfaces, codegen, web, and docs remain follow-up slices.

## Verification Plan

- `go test ./internal/task -run 'TestRunReview|TestTaskManagerRunReviews' -count=1`
- `go test ./internal/store/globaldb -run 'TestGlobalDBTaskRunReviewStore' -count=1`
- `go test -race` for the same focused packages if practical.
- `make verify` before completing the slice.

## Completed

- Added `task.ReviewPolicy`, `task.RunReviewStatus`, `task.RunReviewOutcome`, `task.RunReview`, request/binding/query types, and validation for bounded review text, missing-work JSON, reviewer identity, and terminal run status.
- Added task-service review authority for request, bind, lookup-by-session, and list paths, with event emission for `task.run_review_requested` and `task.run_review_bound`.
- Added GlobalDB `task_run_reviews` CRUD/list/bind logic with idempotent `(run_id, review_round, attempt)` request insertion and transactional linkage from `task_runs.review_request_id`.
- Updated manager/store fakes and test stubs across task, API testutil, and daemon tests.
- Added focused domain, manager, and GlobalDB review-store tests.
- Fixed a Host API prompt cleanup race exposed by `make verify`: `session.Manager` now tracks prompt pump goroutines through `WaitForPromptDrains`, and the Host API test harness waits for prompt drains before tempdir cleanup.

## Verification Evidence

- `go test ./internal/task -run 'TestRunReviewValidation|TestTaskManagerRunReviews' -count=1` passed.
- `go test ./internal/store/globaldb -run 'TestGlobalDBTaskRunReviewStore' -count=1` passed.
- `go test ./internal/task ./internal/store/globaldb ./internal/api/testutil ./internal/daemon -count=1` passed.
- `go test -race ./internal/task -run 'TestRunReviewValidation|TestTaskManagerRunReviews' -count=1` passed.
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBTaskRunReviewStore' -count=1` passed.
- `go test -race -parallel=4 ./internal/session -run TestWaitForPromptDrains -count=1` passed.
- `go test -race -parallel=4 ./internal/extension -run 'TestHostAPIHandlerSessionsPromptReturnsTurnIDAndPersistsEvents|TestHostAPIHandlerObserveEventsReturnsFilteredEventsWithSince' -count=20` passed.
- `go test -race -parallel=4 ./internal/extension -count=1` passed.
- `go test ./internal/store/globaldb -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -count=1` passed.
- `make lint` passed with `0 issues`.
- `make verify` passed: Bun lint/typecheck/test, Vitest `329 files / 2088 tests`, web build, `golangci-lint` `0 issues`, Go race gate `DONE 8193 tests in 96.339s`, and package boundaries respected.

## Follow-Up

- `RecordRunReview`, continuation-run creation, reviewer routing, native `submit_run_review`, HTTP/UDS/CLI surfaces, codegen, web, and docs remain follow-up slices.
