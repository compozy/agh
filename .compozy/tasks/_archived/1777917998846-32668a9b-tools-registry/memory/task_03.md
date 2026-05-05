# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 03 registry indexing, toolset expansion, effective policy evaluation, operator/session projections, and concrete child-session tool subset validation.
- Required evidence: focused unit/integration tests for collisions, toolsets, deny/source/lineage/ACP policy, projection differences; affected package coverage >=80%; `make verify`; self-review; task tracking; one local commit after clean verification.
- Local implementation commit: `275c9855 feat: add tool registry policy projections`.

## Important Decisions
- Keep runtime semantics in `internal/tools`; do not import daemon/API/CLI/extension/MCP/session/task/skills/network packages into `internal/tools`.
- Actual backend invocation remains Task 04. This task should implement deterministic indexing, policy/projection decisions, and fail-closed gates without expanding into dispatch/hook execution.
- `internal/tools` must define config-neutral policy input types because `internal/config` already imports `internal/tools`.
- Operator projections include registered unavailable/unauthorized/conflicted tools with reason codes; session projections include only callable tools for the effective scope.
- Descriptor `VisibilityOperator` and `VisibilityInternal` are operator-diagnostic only and are denied from session-callable projections with `visibility_denied`.
- `internal/store` validates persisted lineage tool atoms with a local canonical ToolID grammar instead of importing `internal/tools`, avoiding an import cycle through resources/store while keeping persistence fail-closed.

## Learnings
- Existing `internal/tools` has core contracts only; no `registry.go`, `policy.go`, or `projection.go` implementation exists yet.
- Task 02 left agent `tools`/`deny_tools` as exact canonical IDs or namespace-prefix wildcards and `toolsets` as canonical ToolsetIDs; runtime expansion belongs here.
- `internal/store/session_lineage.go` currently normalizes generic string atoms; Task 03 must validate concrete tool atoms as canonical `ToolID`s and provide child subset enforcement.
- Focused tools package coverage is 86.2% via `go test ./internal/tools -coverprofile=/tmp/agh-tools-task03.cover -count=1`.

## Files / Surfaces
- Added `internal/tools/pattern.go`, `toolset.go`, `policy.go`, `registry.go`, and focused tests for collisions, toolset expansion, policy precedence/source/lineage/ACP, projection differences, search, and call revalidation.
- Updated `internal/tools/reason.go` with toolset/unknown reason codes and `visibility_denied`.
- Updated `internal/store/session_lineage.go`, `internal/store/session_lineage_test.go`, and `internal/store/globaldb/global_db_session_lineage_test.go` for concrete ToolID atom validation and child subset enforcement.

## Errors / Corrections
- Pre-change signal: `internal/tools/registry.go` is absent (`test -f internal/tools/registry.go` returned false).
- Avoided an `internal/tools` import from `internal/store` after detecting the resources/store import cycle risk; lineage persistence now uses a small local canonical ID validator.
- Fixed registry `Call` revalidation to use the operator projection internally so a registered-but-denied tool reports `ErrToolDenied` instead of `ErrToolNotFound`.
- Refactored touched store tests to satisfy AGH test-shape conventions after the convention checker flagged direct top-level assertions.
- Self-review caught missing `Descriptor.Visibility` enforcement in the policy evaluator; production code now hides operator/internal descriptors from session projections and focused tests cover it.

## Ready for Next Run
- Focused evidence passed:
  - `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/tools/{toolset,policy,registry}_test.go` run one file at a time.
  - `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/store/session_lineage_test.go`.
  - `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/store/globaldb/global_db_session_lineage_test.go`.
  - `go test ./internal/tools ./internal/store ./internal/store/globaldb -count=1`.
  - `go test -race ./internal/tools ./internal/store ./internal/store/globaldb -count=1`.
  - `go test ./internal/tools -coverprofile=/tmp/agh-tools-task03.cover -count=1` => 86.2%.
  - `git diff --check`.
  - `make verify` passed after the visibility correction: 6628 Go tests in 67.442s plus frontend format/lint/typecheck/test/build and package boundaries.
  - Local commit created: `275c9855 feat: add tool registry policy projections`.
  - Post-commit `make verify` passed: frontend format/lint/typecheck/test/build, Go lint with 0 issues, 6628 Go tests in 5.976s, and package boundaries.
- Next: Task 04 can wire dispatch, hooks, budgets, and observability onto the policy/projection foundation.
