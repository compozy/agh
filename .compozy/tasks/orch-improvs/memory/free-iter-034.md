# Free Iteration 034

## Slice

Add native review request, list, and show tools wired to task.Service review authority.

## Acceptance Mapping

- `_techspec.md` Surface Matrix: advances native tool coverage for review-gate management.
- `_techspec_review_gate.md` API/UDS/CLI/Native Tools: adds model-callable review request/read surfaces before HTTP/UDS/CLI parity slices.
- Safety invariant: daemon native tools delegate to `task.Service`; they do not read or write GlobalDB review rows directly.

## Changes

- Added builtin tool ids and descriptors for:
  - `agh__task_run_review_request` / `task_run_review_request`
  - `agh__task_run_review_list` / `task_run_review_list`
  - `agh__task_run_review_show` / `task_run_review_show`
- Added the three review read/request tools to the builtin task toolset.
- Added daemon native bindings:
  - `task_run_review_request` validates task/run/policy input, derives actor context from scope, and calls `task.Service.RequestRunReview`.
  - `task_run_review_list` validates a typed `RunReviewQuery` and calls `task.Service.ListRunReviews`.
  - `task_run_review_show` validates `review_id` and calls the new `task.Service.GetRunReview`.
- Added `task.Manager.GetRunReview` as the service-level read authority over existing review persistence.
- Updated API test stubs and daemon native-tool fakes for the expanded task manager interface.

## Decisions

- `task_run_review_request/list/show` are task-toolset native tools, not autonomy-only tools, because operators and agents need review-gate management access outside reviewer sessions.
- The daemon native layer never bypasses `task.Service`; `RequestRunReview`, `ListRunReviews`, and `GetRunReview` remain the only authority surfaces used by these tools.
- `task_run_review_request` requires both `task_id` and `run_id` in the native MVP. Later HTTP/UDS/CLI route forms can derive one or both ids from path parameters without changing task-service authority.
- Malformed review policy such as `none` is rejected before any `task.Service.RequestRunReview` call.

## Verification

- `go test ./internal/task -run TestTaskManagerRunReviews -count=1`
- `go test ./internal/tools/builtin -count=1`
- `go test ./internal/daemon -run 'TestDaemonNativeTools/Should_route_task_run_review|TestDaemonNativeTools/Should_reject_malformed_task_run_review' -count=1`
- `go test ./internal/tools/builtin ./internal/daemon ./internal/api/testutil -count=1`
- `go test -race ./internal/daemon -run 'TestDaemonNativeTools/Should_route_task_run_review|TestDaemonNativeTools/Should_reject_malformed_task_run_review' -count=1`
- `go test -race ./internal/task -run TestTaskManagerRunReviews -count=1`
- `make lint`
- `make verify`

Final gate evidence: `make verify` passed with Bun lint/typecheck/test, Vitest `329 files / 2088 tests`, web build, `golangci-lint` `0 issues`, Go race gate `DONE 8238 tests in 99.634s`, and package boundaries respected.

## Remaining

- API/UDS/CLI/codegen surfaces for profiles, reviews, notification cursors, bridge subscriptions, and scheduler health.
- Web package integration for task orchestration/review surfaces.
- `packages/site` runtime docs.
- `docs/_memory` lessons.
- QA report and QA execution pair.
- Three clean CodeRabbit rounds.
- Tracked `.agents/skills/cy-codex-loop/scripts/__pycache__/_state_io.cpython-314.pyc` remains dirty; cleanup requires explicit user permission.
