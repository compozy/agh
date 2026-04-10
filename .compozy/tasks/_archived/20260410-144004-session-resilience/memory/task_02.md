# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement task 02 stop classification and explicit stop-cause propagation so every stop path writes a classified `stop_reason` to session metadata and emits it in `session_stopped`.

## Important Decisions

- Keep `Stop()` as the user stop entrypoint and use a cause-aware manager method for daemon shutdown rather than inferring cause from context or process state.
- `handleProcessExit()` should only synthesize a cause when the session does not already have one; explicit shutdown/user-requested causes must survive watcher races.
- Task 02 stays out of global DB and API propagation because task 03 owns that data-layer work.

## Learnings

- `finalizeStopped()` now classifies from explicit `StopCause` before recording `session_stopped`, writes the classified stop fields to meta, and keeps the final in-memory `SessionInfo` aligned with that classification.
- `handleProcessExit()` must only synthesize `CauseCompleted` / `CauseProcessExited` when no explicit stop was already requested, otherwise user/shutdown reasons get overwritten during watcher races.
- The task 01 shared-memory note still applies: stop metadata still stops at `SessionMeta`, in-memory `session.SessionInfo`, and the stored stop event payload for this task.

## Files / Surfaces

- `internal/session/manager_lifecycle.go`
- `internal/session/session.go`
- `internal/session/stop_cause.go`
- `internal/daemon/daemon.go`
- `internal/session/manager_test.go`
- `internal/session/manager_stop_integration_test.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections

- `make verify` initially failed on a `staticcheck` QF1002 complaint in `handleProcessExit()`; fixed by switching to a tagged `switch waitErr`.

## Ready for Next Run

- Task 02 is complete. Next task should propagate the already-classified stop fields through the global DB/API/query layers and keep explicit field mapping boundaries intact.
