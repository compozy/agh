# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add a daemon-owned `TaskManager` in `internal/task` that becomes the canonical surface for task create/get/list/update/child/dependency flows and enforces server-derived identity plus manager-owned task status.
- Meet task_04 deliverables with unit and integration tests, >=80% package coverage, and clean verification.

## Important Decisions
- Keep authorization in-domain using `task.ActorContext` (`Read`, `Write`, `CreateGlobal`, `CreateWorkspace`) rather than adding transport-specific state to the manager API.
- Add an explicit child-task manager operation so parent/child semantics stay centralized in `internal/task`.
- Treat `created_by` and `origin` as fully server-derived from `ActorContext`; create/update payloads remain unable to override them because those fields are absent from request structs and rewritten from trusted context in manager flows.
- Require `ActorContext` on manager read methods (`GetTask`, `ListTasks`) so the v1 principal-based read contract is enforceable inside `internal/task`, not deferred to transports.
- Enforce actor-kind/origin-kind pairing in `ActorContext.Validate()` so mismatched trusted contexts are rejected before any task mutation or read.
- Allow `global -> global|workspace` child relationships, but constrain `workspace -> workspace` children to the same workspace ID to keep cross-scope hierarchy rules bounded and legible.

## Learnings
- Storage dependencies from tasks 02/03 already provide everything needed for this task: task CRUD/list, dependency graph helpers, audit events, task runs, and idempotency records all exist behind `internal/store/globaldb`.
- `ActorContext` already models the v1 authorization contract sufficiently for local human, agent session, automation, extension, and network writer surfaces.
- `UpdateTask` must reconcile canonical status against current dependencies and runs before persisting mutable field changes; otherwise blocked or in-progress tasks can be incorrectly forced back to `ready`.
- Package-level verification for this task is clean (`go test ./internal/task`, `go test ./internal/task -cover`, `go test -tags integration ./internal/task`), but repo-wide `make verify` is blocked by unrelated web tests.

## Files / Surfaces
- `internal/task/`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `.codex/ledger/2026-04-14-MEMORY-task-manager-rules.md`
- `internal/task/actors.go`
- `internal/task/manager.go`
- `internal/task/manager_test.go`
- `internal/task/manager_integration_test.go`

## Errors / Corrections
- Pre-change baseline: `internal/task` had interfaces/types/validation only; there was no concrete manager implementation or child-task operation yet.
- Self-review caught and fixed a real regression in the first manager draft: `UpdateTask` was recomputing status without loading current dependencies/runs, which would have incorrectly changed blocked tasks back to `ready`.
- Repo-wide verification failed outside this task’s surface because of unrelated web test failures in `message-bubble.test.tsx` and `-automation.integration.test.tsx`; do not mark task tracking complete or create the automatic commit until those external failures are resolved or explicitly waived.

## Ready for Next Run
- If repo-wide verification becomes clean, update task tracking files for task 04 and create the single local commit required by the workflow.
