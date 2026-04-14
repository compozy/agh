# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Wire `internal/task` to live session execution through a daemon-owned adapter, make executable subtasks create dedicated sessions by default, preserve explicit attach-session flows, and reconcile orphaned non-terminal runs during daemon boot.
- Deliver the required unit/integration coverage, keep scope tight to task_06, and keep tracking/commit closeout behind clean verification.

## Important Decisions
- Treat the PRD, TechSpec, and ADRs as the approved design baseline; no separate design artifact is needed for this run.
- Keep task/run status mutations manager-owned. Daemon will classify session liveness during boot recovery, but recovery actions should still flow through task-manager logic instead of direct ad hoc store writes whenever possible.
- Mirror existing automation session-creation behavior for dedicated task sessions: system session type, workspace-bound tasks by workspace ID, and global tasks by the daemon home/global workspace path.
- Keep cooperative task cancellation rooted in the session manager by adding a request-stop path in `internal/session` and letting the daemon bridge prefer cooperative cancel before the existing forced stop surface.
- Preserve healthy extensions after partial extension-manager boot errors; hook rebuild must still run when at least one registered extension survives startup.

## Learnings
- `internal/task` already exposes the accepted `SessionExecutor` seam and `TaskManager.StartRun` already depends on it, but no daemon adapter or boot-time task recovery wiring exists yet.
- `internal/daemon` currently has no `internal/task` integration at all; `rg` over `internal/daemon`, `internal/api/core`, `internal/automation`, and `internal/extension` returned no task-domain wiring.
- Session status queries already repair stale on-disk metadata through `session.Status`, which gives boot recovery a reliable way to treat non-active sessions as not live after restart.
- The real extension manager can return a joined startup error while still keeping healthy extensions registered, so daemon boot cannot treat every `manager.Start()` error as a total extension-runtime loss.
- Coverage for the touched packages is now above the required floor: `internal/task` 80.1%, `internal/daemon` 80.4%, and `internal/session` 81.3%.

## Files / Surfaces
- `internal/daemon/task_runtime.go`
- `internal/daemon/task_runtime_test.go`
- `internal/task/interfaces.go`
- `internal/task/manager.go`
- `internal/task/manager_test.go`
- `internal/task/manager_integration_test.go`
- `internal/session/manager.go`
- `internal/session/manager_lifecycle.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections
- Corrected daemon extension boot handling so partial extension startup failures still publish healthy registered extensions and rebuild hooks; without that fix, the daemon integration suite failed while verifying this task.

## Ready for Next Run
- Implementation and verification are complete; remaining closeout work is task tracking updates plus the single local commit with code/test changes only.
