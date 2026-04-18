# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extend the extension Host API so embedded consumers can read the richer task surfaces already exposed through task manager and observer-backed APIs.
- Keep Host API naming and request shapes aligned with existing `tasks/*` conventions and reuse shared task contract payloads from task_07.
- Also fix the existing `tasks` and `tasks/get` Host API payload shaping, which still omits several task_07 fields.

## Important Decisions
- Final method set: `tasks/runs/get`, `tasks/timeline`, `tasks/tree`, `tasks/dashboard`, and `tasks/inbox`.
- Treat live-read parity as request/response reads; do not introduce a separate Host API streaming protocol unless implementation evidence forces it.
- Keep Host API task payload shaping aligned with `internal/api/contract/tasks.go` by expanding existing `tasks` and `tasks/get` converters instead of creating host-only DTO forks.

## Learnings
- Current Host API already routes `tasks/get` to manager `GetTask`, but its converter drops `summary`, dependency references, and newer summary metadata added in task_07.
- Current Host API observer surface only exposes generic health/events, so dashboard and inbox require interface and handler expansion even though the underlying observer already supports them.
- `TestReferenceExtensionsEndToEnd` rebuilds the extension SDK and fails fast when shared contract codegen is stale, so Host API contract changes require `make codegen` before integration or full verify passes.

## Files / Surfaces
- `internal/extension/protocol/host_api.go`
- `internal/extension/protocol/host_api_test.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/capability.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_tasks.go`
- `internal/extension/host_api_test.go`
- `internal/extension/host_api_integration_test.go`
- `sdk/typescript/src/generated/contracts.ts`
- `openapi/agh.json`

## Errors / Corrections
- Integration failure in `TestReferenceExtensionsEndToEnd` was caused by stale generated contracts; corrected by running `make codegen`.
- `make verify` initially failed on `gocritic` `hugeParam` for `taskDashboardPayloadFromView`; corrected by switching that helper to a pointer parameter and updating the zero-value test.

## Ready for Next Run
- Task completed locally with targeted extension unit/integration passes, `internal/extension` coverage at `80.5%`, and a clean `make verify`.
