# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement metadata-only normal session health/presence for Task 07.
- Success means `internal/session` can persist/query health rows with state, health, last activity/presence, active prompt, attachable, wake eligibility, and deterministic ineligibility reason without prompts/model/tool/task/network side effects.
- Required validation includes targeted session/store tests, >=80% affected package coverage evidence, `make verify`, self-review, task tracking updates, and one local commit.

## Important Decisions
- Build on Task 06's existing v13 `session_health` global DB table/store; no schema migration is needed for Task 07.
- Keep health runtime-owned and metadata-only. `HEARTBEAT.md` remains wake policy only and will consume health later.
- Baseline before implementation: `rg -n "SessionHealth|GetSessionHealth|RecoverSessionHealth|TouchSessionPresence|session health" internal/session` returned no matches; `go test ./internal/session -run TestManagerSessionHealth -count=1` passed with `[no tests to run]`.
- Runtime-facing session health uses `session.HealthStore` and `session.HealthRecoveryResult` to avoid package stutter while keeping the manager API route-ready.
- Active prompt activity writes `last_activity_at` and prompt-active ineligibility; idle presence writes `last_presence_at` only when the prompt is not active or when prompt teardown explicitly records idle completion.

## Learnings
- Existing active prompt supervision persists `store.SessionLivenessMeta.Activity` in `internal/session/prompt_activity.go` but does not write `session_health`.
- `session_health` has FK to `sessions(id)`, so daemon wiring must ensure the global session index exists before durable health writes matter in production.
- Existing `_tasks.md` has pre-existing edits marking tasks 01/02/03/05/06 complete and renaming QA tasks; do not overwrite those changes when later marking task 07.
- `promptActivitySupervisor.finish` runs before current turn fields are cleared; task 07 added an explicit idle-presence persist path so completed prompts do not leave health stuck in `prompting`.
- Boot recovery now refreshes active health rows, recomputes persisted rows from repaired session metadata, and marks stale rows before scheduler/wake work can consume them.
- Coverage validation exposed an existing async hook shutdown bug: `asyncPool.Close` timed out but then waited forever when an active hook ignored context cancellation. Task 07 fixed the production close path so shutdown is bounded by the drain deadline and added a regression test.
- Stale detection must apply to idle non-prompt presence rows only; active prompt rows remain wake-ineligible due `session_prompt_active` even when their previous idle `last_presence_at` is past `session_health_stale_after`.

## Files / Surfaces
- Production surfaces: `internal/heartbeat/persistence.go`, `internal/store/globaldb/global_db_heartbeat.go`, `internal/session/health.go`, `internal/session/manager.go`, `internal/session/manager_helpers.go`, `internal/session/manager_lifecycle.go`, `internal/session/manager_prompt.go`, `internal/session/prompt_activity.go`, `internal/daemon/daemon.go`, `internal/daemon/boot.go`, `internal/daemon/soul_runtime.go`.
- Test surfaces: `internal/session/health_test.go`, `internal/heartbeat/persistence_test.go`.

## Errors / Corrections
- Corrected prompt completion health so idle persistence is explicit during supervisor finish.
- Renamed exported session health interfaces/results from `SessionHealthStore`/`SessionHealthRecoveryResult` to `HealthStore`/`HealthRecoveryResult` after focused lint caught package stutter.
- Corrected `internal/hooks.asyncPool.Close` to return after the drain deadline even when an active async hook ignores `ctx`, preventing validation and shutdown hangs.
- Corrected stale marking to exclude `active_prompt` and non-`idle` rows, preventing long prompts from being overwritten as stale.
- Test convention helper accepts one file per invocation; reran it separately for each touched test file.

## Ready for Next Run
- Task implementation, self-review corrections, tracking updates, commit, and post-commit verification are complete.
- Validation evidence: `go test ./internal/session -run 'TestManagerSessionHealthTransitions|TestManagerSessionHealthRecovery' -count=1`; `go test ./internal/store/globaldb -run TestGlobalDBSessionHealthStaleDetection -count=1`; `go test ./internal/hooks -run TestAsyncPoolCloseDeadline -count=1`; `go test ./internal/session ./internal/heartbeat ./internal/store/globaldb ./internal/daemon ./internal/hooks -count=1`; `go test -race ./internal/session ./internal/heartbeat ./internal/store/globaldb ./internal/daemon ./internal/hooks -count=1`; focused `golangci-lint`; AGH test convention helper for `internal/session/health_test.go`, `internal/heartbeat/persistence_test.go`, `internal/hooks/pool_close_test.go`, and `internal/store/globaldb/session_health_stale_test.go`; coverage `internal/session` 80.0568% and `internal/heartbeat` 80.1567%; `make verify` passed with `DONE 7600 tests` and boundaries OK.
- Commit: `ea7d7bf8 feat: add metadata-only session health` with code/test files only.
- Post-commit validation: `make verify` passed with `DONE 7600 tests in 12.530s` and boundaries OK.
