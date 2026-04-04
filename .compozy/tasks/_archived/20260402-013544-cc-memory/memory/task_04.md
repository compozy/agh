# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Wire memdir + dream integration into the running kernel: home layout, session boot, prompt memory context, dream boot/ticker/session-stop triggers, dream session spawning, and HTTP memory endpoints.

## Important Decisions

- Use `sessionBootstrapPlan` to distinguish standard sessions from dream consolidation sessions so the same session manager can bootstrap either the normal supervisor/advisor pair or a one-shot `dream-worker`.
- Build prompt memory context in one place (`SessionManager.buildMemoryContext`) and reuse it for bootstrap prompts and runtime agent spawns.
- Keep daemon-scoped memory CRUD endpoints explicit: workspace-scope reads/writes/deletes require a `workspace` argument instead of inferring from daemon cwd.
- Let the kernel own dream lifecycle coordination with `dreamCtx`, `dreamWG`, and `dreamSpawner` so shutdown waits for in-flight consolidation runs.

## Learnings

- `internal/state.ListBlackboard()` already supported `Type` filtering; the team-memory requirement only needed session-manager integration and tests, not state-query production changes.
- The dream ticker will not call `ShouldRun()` until a workspace can be resolved. In practice that means an explicit workspace trigger or at least one stopped session whose workspace can be reused.
- Task_04 coverage is best measured on the touched production surfaces, not the entire historical `internal/kernel` package. The final weighted task_04 surface coverage reached `80.01% (2069/2586)` from `go test -coverprofile=/tmp/task04-all-cover.out ./internal/config ./internal/state ./internal/kernel/...`.

## Files / Surfaces

- `internal/config/config_validation_task04_test.go`
- `internal/config/config.go`
- `internal/config/home.go`
- `internal/config/home_test.go`
- `internal/kernel/types.go`
- `internal/kernel/kernel.go`
- `internal/kernel/kernel_test.go`
- `internal/kernel/kernel_dream_test.go`
- `internal/kernel/session_manager.go`
- `internal/kernel/session_manager_test.go`
- `internal/kernel/session_manager_memory_test.go`
- `internal/kernel/session_manager_dream_test.go`
- `internal/kernel/api.go`
- `internal/kernel/api_memory_test.go`
- `internal/kernel/api_memory_helpers_test.go`
- `internal/kernel/api_memory_context_test.go`

## Errors / Corrections

- Fixed the existing boot-sequence assertion to include the new `init_dream_service` step.
- Hardened `TestSessionManagerConcurrentCreateAndStopIsSafe` by replacing goroutine-local `t.TempDir()` workspaces with explicit temp dirs cleaned up after kernel shutdown; coverage instrumentation exposed the old cleanup ordering as flaky.
- Adjusted the dream ticker test to seed a stopped session first, because ticker-driven dream checks need a resolvable workspace before they can reach `ShouldRun()`.
- Fixed a real daemon integration bug exposed by the new direct helper tests: global memory writes were failing because `kernel.writeMemory()` redundantly called `memdir.Store.EnsureDirs()` on a global-only store.

## Ready for Next Run

- Verification is clean: `make verify` passed after the final memory/dream integration fixes.
- Update `task_04.md` and `_tasks.md`, self-review the task diff, and commit only the task-related code/tests plus required task tracking files. Keep workflow-memory notes uncommitted unless a later workflow explicitly wants them staged.
