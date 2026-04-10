# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace daemon `notifierFanout` / `skillsHookDispatcher` with a single `internal/hooks.Hooks` composition in `boot.go`, wire skill reload + native callbacks, update shutdown ordering, and replace notifier-era tests with task_09 coverage.

## Important Decisions
- Hard cut-over only: delete notifier fanout/dispatcher code instead of keeping compatibility layers.
- Preserve observer and dream-session side effects through native hook callbacks owned by the daemon/hooks composition, not through separate post-session callback lists.
- Keep the `session.Notifier` seam in `internal/daemon` as a thin `hooksNotifier` adapter over `Hooks`; moving the interface implementation into `internal/hooks` would reintroduce the current `session -> config/skills -> hooks` package cycle.

## Learnings
- Current `internal/hooks.Hooks` already owns registry rebuild and async worker-pool lifecycle, but only exposes a no-op `OnAgentEvent`; task_09 still needs the session lifecycle notifier bridge in addition to daemon composition.
- The existing skills watcher only refreshes the skills registry; task_09 needs a daemon-level callback so the same watcher cycle also triggers `Hooks.Rebuild()`.
- Skill hook metadata now rejects the legacy per-hook `name` field; task fixtures must use the current declaration shape (`event`, executor fields, matcher fields, etc.) during daemon integration tests.

## Files / Surfaces
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/hooks_bridge.go`
- `internal/daemon/notifier.go`
- `internal/daemon/notifier_integration_test.go`
- `internal/daemon/notifier_test.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/hooks/payloads.go`
- `internal/skills/watcher.go`

## Errors / Corrections
- `go test -tags integration ./internal/daemon` initially failed because the new skill-hook integration fixtures still used the deprecated per-hook `name` field; corrected the fixtures to match the current parser contract.

## Ready for Next Run
- Verification complete after the daemon hooks cut-over: focused daemon/hooks tests, daemon integration tests, `internal/daemon` coverage (`80.4%`), and `make verify` all passed.
- Local code commit created: `4b3d39e` (`refactor: wire daemon hooks runtime`). Workflow memory and PRD tracking updates were kept out of the commit.
