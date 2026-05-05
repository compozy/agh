# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement a daemon-owned mechanical scheduler for Task 11. Success means scheduler lifecycle is context-bound, pending queued runs wake only idle eligible sessions, expired leases recover through task service APIs, boot rebuild happens before external servers expose stale work, scheduler emits logs/metrics without hooks, required tests pass, `make verify` passes, and tracking/commit are completed.

## Important Decisions
- Keep scheduler state ephemeral and rebuildable in `internal/scheduler`; daemon will adapt durable task/session data into narrow scheduler interfaces.
- Scheduler must not call `ClaimNextRun`. Wake prompts are advisory and tell agents to use existing task claim verbs.
- Expired lease recovery must call `task.Service.RecoverExpiredRunLeases` so audit events/hooks remain at the task transition site.

## Learnings
- Existing task service already has `RecoverExpiredRunLeases`, and daemon boot already runs task recovery before server boot; Task 11 should extend that ordering with scheduler rebuild/start rather than add another ownership path.
- Existing session manager exposes active session snapshots and `PromptSynthetic`; scheduler can use those for wakeups while skipping prompting/busy sessions.
- Existing hook tests already assert scheduler observability names are not hook events; Task 11 should not add `scheduler.*` hook descriptors.
- New `internal/scheduler` package unit tests pass with deterministic fake clocks and fakes for task/session/wake sources. The package delegates expired lease recovery through its task source, tracks ephemeral wake cooldown state, and never exposes a claim path.
- Daemon boot now rebuilds scheduler state and starts the loop after task boot recovery but before external transports; shutdown stops the loop before sessions and waits wake-drain workers after sessions close.
- Scheduler integration tests use real `globaldb` plus `task.Service`: wake notifications leave runs queued/unowned until `ClaimNextRun`, expired leases recover after reopening the DB, wrong-workspace sessions do not claim, and task events contain no `scheduler.*` entries.
- Fresh final verification passed at `2026-04-26 07:42:40 -03`: `make verify` completed frontend checks, Go lint with `0 issues`, race-enabled Go tests (`DONE 6202 tests in 6.427s`), build, and package-boundary checks. Focused scheduler coverage is `86.5%` with `go test ./internal/scheduler -cover`.
- Local code/tracking commit created: `8f29191f feat: add mechanical scheduler`. Post-commit `make verify` also passed with Go lint `0 issues`, `DONE 6202 tests in 6.483s`, and package-boundary checks OK.

## Files / Surfaces
- Added: `internal/scheduler/*` for scheduler runtime and deterministic tests.
- Added: `internal/scheduler/scheduler_integration_test.go` for real task service/store integration coverage.
- Added: `internal/daemon/scheduler_runtime.go`, `internal/daemon/boot.go`, and `internal/daemon/daemon.go` for composition, boot rebuild/start, and shutdown ownership.

## Errors / Corrections
- `make verify` initially exposed real lint issues: unchecked cleanup errors, long/large-value scheduler helpers, copied locks from value task/run structs, `gosec` int conversions, and oversized test helpers. The fix kept scheduler behavior unchanged while switching large values to pointers, logging cleanup errors, using `int` stats counters, and splitting focused tests.

## Ready for Next Run
- Task 11 is implemented and verified. Next tasks should treat `internal/scheduler` as a mechanical liveness component only: it wakes idle eligible sessions and sweeps expired leases through `task.Service`, but never claims work or emits `scheduler.*` hook events.
