# Free Iteration 030 - Reviewer-Bound `submit_run_review` Native Tool

## Objective

- Add reviewer-bound native `submit_run_review` support wired to `task.Service.RecordRunReview`.
- Hide the tool when the caller session does not have an active review request binding.

## Decisions

- The internal tool id is `agh__task_run_review_submit`; the model-facing native tool name is `submit_run_review`.
- Availability first verifies that the bundled `agh-task-reviewer` skill declares `metadata.agh.requires_review_request = true`, then requires an active reviewer-session binding via `LookupRunReviewForSession`.
- The native tool is session-bound only. It does not use claim leases, does not expose claim tokens, and delegates all verdict authority to `task.Service.RecordRunReview`.
- Input must match the bound `review_id` and `run_id`; mismatches are denied before any verdict write.
- `missing_work` is normalized to bounded JSON for the task-service verdict path; rejected reviews still rely on the existing idempotent continuation-run creation in `RecordRunReview`.

## Files Touched

- `internal/tools/builtin_ids.go`
- `internal/tools/builtin/autonomy.go`
- `internal/tools/builtin/toolsets.go`
- `internal/tools/builtin/builtin_test.go`
- `internal/daemon/native_tools.go`
- `internal/daemon/native_review_tools.go`
- `internal/daemon/native_tools_test.go`

## Verification

- `go test ./internal/tools/builtin -count=1`
- `go test ./internal/daemon -run 'TestDaemonNativeTools/Should_route_reviewer-bound_submit_run_review|TestDaemonNativeTools/Should_hide_submit_run_review|TestDaemonNativeTools/Should_reject_schema-invalid_submit_run_review' -count=1`
- `go test ./internal/tools/builtin ./internal/daemon -count=1`
- `go test -race ./internal/daemon -run 'TestDaemonNativeTools/Should_route_reviewer-bound_submit_run_review|TestDaemonNativeTools/Should_hide_submit_run_review|TestDaemonNativeTools/Should_reject_schema-invalid_submit_run_review' -count=1`
- `go test -race ./internal/daemon -run TestDaemonNativeTools -count=1`
- `make lint`
- `make verify`

Final gate evidence:

- Bun lint/typecheck/test passed.
- Web build passed.
- `golangci-lint` reported `0 issues`.
- Go race gate finished with `DONE 8227 tests in 73.950s`.
- Package boundaries passed.

## Findings

- The first focused hidden-tool test expected `ErrToolDenied`, but the native-tool availability path intentionally maps hidden unavailable tools to `ErrToolUnavailable`; the test was corrected to the actual policy contract.
- `make lint` caught `gocritic hugeParam` on a heavy review binding passed by value; the implementation now passes that binding by pointer instead of suppressing lint.

## Remaining

- Profile/review read and update native tools remain pending.
- HTTP/UDS/CLI transport surfaces and OpenAPI/codegen remain pending.
- Web package updates, `packages/site` docs, `docs/_memory` lessons, QA pair, CodeRabbit clean rounds, and tracked `.pyc` cleanup remain pending.
