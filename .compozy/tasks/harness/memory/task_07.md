# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build the daemon-owned bridge that turns detached harness task-run terminal completion into either synthetic session reentry or observable silent completion/drop, without bypassing task-runtime durability or the task-04 synthetic prompt path.

## Important Decisions
- Pre-change inspection confirmed there is no existing detached completion bridge or dedicated observability summary for wake/drop outcomes; task 07 needs one daemon-owned coordinator.
- The bridge should consume durable task completion signals and reuse `session.Manager.PromptSynthetic` for all wake-ups so busy-session FIFO semantics stay centralized in `internal/session`.
- The bridge should use task-level `EventObserver` callbacks fed by persisted `task.EventRecord` rows rather than polling task state, so detached completion stays push-driven and boot recovery can replay the same durable source.
- Recovery should dedupe from both persisted run metadata (`metadata.reentry`) and already-recorded synthetic session events, so crash windows do not resend the same wake-up when the run already injected a synthetic turn.

## Learnings
- `task.Service` already emits stable durable task event records and live stream events for task lifecycle transitions, including `task.run_completed` / `task.run_failed`, so the bridge can stay push-driven without polling hidden state.
- `session.Manager.Status` returns stopped-session metadata from disk, which allows explicit active/stopped/unavailable wake target decisions without guessing from only in-memory sessions.
- `event_summaries` inserts require the target session to exist in the global session index, so daemon integration tests with fake session managers need explicit workspace/session seeding even when the runtime surface is mocked.
- Running the full repo gates exposed unrelated transient integration flakes in `internal/extension` and `internal/daemon`; targeted reruns passed, and the final clean evidence is `make test-integration` plus `make verify`.

## Files / Surfaces
- `internal/daemon/task_runtime.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/harness_reentry_bridge.go`
- `internal/daemon/harness_detached_work.go`
- `internal/daemon/task_runtime_test.go`
- `internal/task/live.go`
- `internal/task/live_types.go`
- `internal/task/manager.go`
- `internal/session/synthetic_prompt.go`

## Errors / Corrections
- `make verify` initially failed on bridge lint issues (`funlen`, `lll`, `unparam`) and a staticcheck warning in the new helper test; refactoring `processTerminalRun` into `applyWakeDecision`, removing the dead `error` return from `evaluateDecision`, and routing the nil-context assertion through `nilTaskRuntimeContext()` resolved the gate cleanly.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/daemon ./internal/task -count=1`
  - `go test -tags integration ./internal/daemon -run 'TestDetachedHarnessCompletionWakeEmitsSyntheticReentryEndToEnd|TestDetachedHarnessCompletionWakePreservesFIFOAcrossRuns|TestBootRecoveryDetachedHarnessWakeUsesPersistedSyntheticEventForDedupe|TestBootWiresDetachedHarnessTaskRuntimeAcrossScopes|TestBootRecoversDetachedHarnessRunThroughTaskRuntimeRules' -count=1`
  - `go test -tags integration -coverprofile=.codex/tmp/task07-daemon.cover ./internal/daemon -count=1`
  - `make test-integration`
  - `make verify`
- Coverage evidence:
  - `internal/daemon/harness_reentry_bridge.go` = 83.0% (249/300)
  - `internal/daemon/task_runtime.go` = 80.0% (132/165)
- Unrelated workspace changes still exist in task tracking/docs and `web/` docs; keep commit scope limited to task 07 code files.
