# Task Memory: task_22

## Objective Snapshot

- Implemented daemon-side ReviewRouter wake and routing after task-service review request creation.
- Added reviewer route selection and binding against persisted `TaskExecutionProfile.Review` selectors.
- Removed the audited review-store `strings.Contains(err.Error(), "uq_...")` constraint classification and replaced it with typed SQLite error inspection.

## Important Decisions

- Review request creation remains authoritative in `task.Service`; the new observer is a best-effort runtime notification and does not move review state by itself.
- The daemon owns runtime routing because it has the composition-root access to task service, task store, session manager, workspace resolver, and agent capability catalog.
- The router first tries active eligible reviewer sessions, then creates a reviewer system session only when peer selectors do not require an existing peer.
- No-route outcomes are persisted through `task.Service.RecordRunReview` with a deterministic delivery id prefix, preserving review authority and giving operators/API/UI a queryable diagnostic.
- Original-worker exclusion checks session id, peer id, and original agent name when available before any reviewer binding.

## Files / Surfaces

- `internal/task/review.go`
- `internal/task/manager.go`
- `internal/task/manager_review.go`
- `internal/task/manager_review_test.go`
- `internal/daemon/review_router.go`
- `internal/daemon/review_router_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/task_runtime.go`
- `internal/daemon/coordinator_runtime.go`
- `internal/store/globaldb/global_db_task_review.go`
- `internal/store/globaldb/global_db_task_review_test.go`

## Errors / Corrections

- `make lint` first reported `gocritic hugeParam` on the review-request notification and `lll` in the router test. The notification observer now passes a pointer and the test assertion was split.
- The first full `make verify` failed in `sdk/typescript/src/integration.test.ts` with a 30s timeout unrelated to the task_22 Go changes. The focused SDK integration test passed immediately after isolation, and the final full gate passed.

## Ready for Next Run

- `task_23` should build on the persisted review route/binding state and no-route diagnostics when constructing the task context bundle.
- Downstream web/docs tasks should present no-route diagnostics as task-service review outcomes, not as channel authority.
- The `.pyc` artifact remains unresolved and still requires an explicit user decision before cleanup.

## Verification Evidence

- `go test ./internal/task -run TestTaskManagerRunReviews -count=1` passed.
- `go test ./internal/store/globaldb -run TestGlobalDBTaskRunReviewStore -count=1` passed.
- `go test ./internal/daemon -run TestReviewRouterRoutesRunReviewRequests -count=1` passed.
- `go test ./internal/task ./internal/daemon ./internal/store/globaldb -count=1` passed.
- `go test -race ./internal/task -run TestTaskManagerRunReviews -count=1` passed.
- `go test -race ./internal/daemon -run TestReviewRouterRoutesRunReviewRequests -count=1` passed.
- `go test -race ./internal/store/globaldb -run TestGlobalDBTaskRunReviewStore -count=1` passed.
- `bunx vitest run sdk/typescript/src/integration.test.ts` passed after isolating the first full-gate timeout.
- `make lint` passed with `0 issues`.
- `make verify` passed: Bun lint/typecheck/test, Vitest 329 files / 2092 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8267 tests in 142.023s`, and package boundaries OK.
