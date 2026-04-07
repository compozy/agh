# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Re-root the shared API server boundary into `internal/api/core`, merge the old `internal/apisupport` helpers into that package, preserve handler/parser/SSE behavior, and remove the old package boundaries without bridges.

## Important Decisions
- Use `internal/api/core` as the only owner for shared transport-facing interfaces, handlers, parsers, SSE helpers, conversions, memory/workspace helpers, and status mapping.
- Merge `internal/apisupport/session_workspace.go` directly into `internal/api/core` as package-local helpers instead of keeping exported forwarders or aliases.
- Keep `internal/httpapi/prompt.go` transport-local and only repoint its shared payload helpers to `internal/api/core`.

## Learnings
- `internal/api/core` coverage landed at `81.3%` after adding direct package-level tests for the merged session/workspace helper behavior from the retired `apisupport` package.
- Transport integration stayed green after the cutover: `go test -tags integration ./internal/httpapi ./internal/udsapi -count=1` and the full `make test-integration` both passed.

## Files / Surfaces
- Added: `internal/api/core/*` with the moved shared API implementation and tests, including `session_workspace_internal_test.go`.
- Updated consumers: `internal/httpapi/{prompt.go,server.go,shared.go,shared_test.go}`, `internal/udsapi/{server.go,shared.go,shared_test.go}`, `internal/apitest/{apitest.go,apitest_test.go}`, and `internal/api/contract/contract_test.go`.
- Removed package boundaries: `internal/apicore/` and `internal/apisupport/`.

## Errors / Corrections
- Initial coverage for `internal/api/core` was `79.9%`, below the task threshold; fixed by adding direct tests for merged session/workspace helper behavior rather than weakening the requirement.

## Ready for Next Run
- Task 04 code and verification are complete. Update task tracking, keep workflow memory in sync, and preserve code-only staging for the local commit because `.compozy/tasks/refac-v2/` already has unrelated tracking changes in the worktree.
