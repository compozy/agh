# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 03 by adding a durable restart-operation store, daemon-owned restart orchestration, a reusable detached launcher, and the internal `agh daemon relaunch` helper flow.
- Keep scope on runtime orchestration and persistence only; API contract/handlers land in later tasks.

## Important Decisions
- Persist extra helper-only context in the restart record, including the pre-restart daemon socket path and old daemon start time, so the helper can wait on the old singleton resources even if the new config changes socket settings.
- Trigger daemon shutdown for restart by signaling the running daemon process after helper spawn instead of calling in-process `exec` or synchronously tearing down the runtime from the request path.
- Hand off the restart operation id to the replacement daemon through an internal environment variable so boot success can mark the persisted operation `ready` after fresh daemon discovery state exists.
- Factor detached process creation into shared process-launch code below `internal/cli` so both CLI start and daemon restart orchestration reuse the same spawn semantics.
- Launch detached helper/replacement processes via `os.StartProcess`-backed procutil helpers so detached children are not coupled to caller-context cancellation and the shared launcher stays acceptable to the strict lint profile.

## Learnings
- Existing daemon boot already writes `daemon.json` before `bootFinalize`, and shutdown removes `daemon.json` before releasing the singleton lock; the helper only needs to coordinate around those existing lifecycle points.
- Helper stage timeouts must cap longer parent-context deadlines; otherwise restart release polling can overrun its configured timeout and leave persisted state stuck in an in-flight status.
- Failure persistence in helper timeout/error paths must use the configured operation id rather than any zero-value operation returned alongside an error.

## Files / Surfaces
- `internal/cli/daemon.go`
- `internal/cli/root.go`
- `internal/config/home.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/info.go`
- `internal/daemon/lock.go`
- `internal/daemon/restart.go`
- `internal/daemon/restart_test.go`
- `internal/daemon/restart_integration_test.go`
- `internal/procutil/*`

## Errors / Corrections
- Corrected helper timeout handling so release/ready waits honor the configured stage deadline even when the caller context already has a later deadline.
- Corrected helper failure persistence to transition the configured restart operation to `failed` even when wait helpers return a zero-value operation on timeout/error.
- Reworked detached-process spawning away from `exec.CommandContext` so lint stays green without suppressions and detached daemons are not killed by later caller-context cancellation.

## Ready for Next Run
- Task is verified complete. Follow-on API work should call `internal/daemon` restart methods and read persisted operation state instead of rebuilding restart-file paths in transports.
