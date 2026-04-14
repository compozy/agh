# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the daemon-backed `agh task` CLI command group for task create/list/get/update, child creation, dependency management, and run lifecycle actions without bypassing the shared UDS/API contract.
- Deliver unit tests for flag validation and payload/query mapping plus integration tests that exercise `agh task` end-to-end against a live UDS daemon.

## Important Decisions
- Treat `task_09.md`, `_techspec.md`, and ADR-004/ADR-005 as the approved design baseline; no conflicting requirements were found.
- Reuse `internal/api/contract/tasks.go`, `internal/api/udsapi/routes.go`, and the CLI organization style from `internal/cli/automation.go` instead of introducing task-specific transport shapes or direct store access.
- Extend the existing CLI daemon-backed integration harness with task-manager wiring because it currently boots UDS without the task service.
- Keep task/run output formatting inside `internal/cli/task.go` alongside command construction so future CLI additions reuse the same scope/owner/channel rendering rules instead of scattering task formatting across helpers.

## Learnings
- `internal/cli` currently has no task CLI implementation or task-related client methods, so this task needs both command-tree and client transport work.
- The existing `internal/cli/cli_integration_test.go` harness wires automation/network/memory/extension services into the UDS server but not `WithTasks`, so end-to-end task CLI coverage requires fixture expansion.
- The CLI integration harness also needs an observe-compatible bridge stub that implements `DeliveryMetrics()` because `observe.WithBridgeSource(...)` now depends on the widened `observe.BridgeSource` interface.
- `internal/cli` package coverage for this task clears the required bar at `80.6%` after exercising task mutation success paths and TOON/detail renderers in `internal/cli/task_test.go`.

## Files / Surfaces
- `internal/cli/root.go`
- `internal/cli/client.go`
- `internal/cli/helpers_test.go`
- `internal/cli/cli_integration_test.go`
- `internal/cli/automation.go` and existing CLI test files as reference patterns
- `internal/api/contract/tasks.go`
- `internal/api/udsapi/routes.go`

## Errors / Corrections
- Fixed a stale integration-harness compile break by adding `DeliveryMetrics()` to `integrationBridgeService` after `observe.BridgeSource` expanded.
- Relaxed the task-run lifecycle integration assertion to validate result JSON semantically instead of requiring an exact compact byte string, because persisted JSON may be normalized with whitespace.

## Ready for Next Run
- Implementation and verification are complete. Fresh evidence:
- `go test ./internal/cli -cover -count=1` -> `coverage: 80.6% of statements`
- `go test -tags integration ./internal/cli -count=1` -> pass
- `make verify` -> pass
- Local commit created: `f81534f` (`feat: add task cli commands`)
- Post-commit `make verify` also passed on the committed state.
- Tracking and workflow memory files remain intentionally unstaged; existing unrelated task-tracking/generated-file changes in the worktree were left untouched.
