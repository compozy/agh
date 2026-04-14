# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed `task_07` by adding shared task/run contracts and `internal/api/core` handlers for the `internal/task.Manager` surface without duplicating lifecycle rules in transports.
- Kept scope to contract/core/test work; transport route wiring remains deferred to `task_08`.

## Important Decisions
- Use existing task-domain identity types (`created_by`, `origin`, `owner`) directly in API payloads to preserve semantics from ADR-005.
- Treat the PRD/TechSpec/ADRs as the approved design input for this scoped execution task rather than inventing a separate design artifact.
- Add `BaseHandlerConfig.Tasks` plus `TaskActorContextResolver` as the transport seam so HTTP/UDS can share the same handlers while still deriving actor context server-side.
- Keep task-run list reads thin by filtering `GetTask(...).Runs` in the core layer instead of inventing a separate transport-only rule surface.

## Learnings
- `internal/task.Manager` already exposes all needed lifecycle methods, including `CreateChildTask`, `GetTask/ListTasks` with actor context, `AttachRunSession`, and `CancelRun`.
- Task-domain errors already distinguish validation, permission, not-found, invalid-transition, and session-binding failures, which the API layer can map directly.
- There is currently no task/run API surface in `internal/api/core` or `internal/api/contract`; the baseline `rg` search returned no matches.
- Reusing `network.ValidateChannel` kept task channel validation consistent with the network ingress rules already enforced elsewhere in the daemon.
- The `internal/api/core` package coverage gate needed helper/error-path test additions outside the new task handlers because package coverage spans the whole shared API/core package, not just task files.

## Files / Surfaces
- Implemented: `internal/api/contract/tasks.go`, `internal/api/contract/responses.go`, `internal/api/contract/contract_test.go`, `internal/api/core/interfaces.go`, `internal/api/core/handlers.go`, `internal/api/core/errors.go`, `internal/api/core/tasks.go`, `internal/api/core/test_helpers_test.go`, `internal/api/core/tasks_test.go`, `internal/api/core/tasks_internal_test.go`, `internal/api/core/tasks_integration_test.go`, `internal/api/core/automation_additional_test.go`, `internal/api/core/errors_test.go`, `internal/api/core/handlers_internal_test.go`, and `internal/api/testutil/apitest.go`.

## Errors / Corrections
- The worktree already contains unrelated task tracking edits and an untracked memory directory; avoid touching unrelated files while implementing `task_07`.
- `internal/api/core` coverage initially stalled below the required gate (`79.3%`); corrected that with shared helper/error-path tests and reverified at `80.0%` without widening product scope.

## Ready for Next Run
- `task_08` should bind HTTP and UDS routes directly to the new task/run handler methods and install transport-specific actor-context resolution where needed.
