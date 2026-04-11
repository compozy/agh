# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Integrate the existing composed automation manager into daemon boot/shutdown and publish the runtime surface needed by later transport work.
- Keep scope on task 06: daemon wiring, TOML sync/overlay verification, lifecycle status, and required unit/integration coverage.

## Important Decisions
- Preserve the untracked `internal/automation/manager.go` already present in the workspace and treat task 06 as integration plus verification work.
- Insert automation boot after `bootExtensions` and before `bootServers` so hooks are available, extensions still boot on the current path, and transports only start after automation is ready.
- Use daemon-owned fan-out adapters for native hook lifecycle observers and hook telemetry sinks so automation consumes the existing observer/hooks boundary instead of direct `session.Manager` notifications.
- Expose a narrow transport-facing automation interface through runtime dependencies now; defer actual HTTP/UDS handler work to later tasks.
- Keep `memory.consolidated` as an exposed automation seam for now; do not widen task 06 into a dream-runtime refactor without a concrete callback surface in `internal/memory/consolidation`.

## Learnings
- `session.Manager` already emits direct notifier callbacks, but the daemon intentionally routes observer lifecycle handling through post-create/post-stop native hooks. Automation should plug into that same hook boundary.
- The shared workflow memory already settles the global automation workspace fallback: use the AGH home directory as `session.CreateOpts.WorkspacePath` for global jobs.
- Hooks currently accept only one telemetry sink at construction time, so daemon-side fan-out is the lowest-risk way to attach automation telemetry after hooks boot.
- The existing manager implementation already handled TOML sync, overlay application, scheduler/trigger composition, and status reporting; task 06 mainly needed daemon composition, runtime publication, and proof through tests.
- The full repo gate will reject tests that pass literal `nil` contexts into exported context-taking APIs, even when the production code guards against nil; valid coverage has to come from non-lint-violating paths.

## Files / Surfaces
- `internal/api/core/interfaces.go`
- `internal/automation/manager.go`
- `internal/automation/manager_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/hooks_bridge.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections
- `make verify` initially failed on `staticcheck` because a new manager test passed literal `nil` contexts; removed that test and replaced the lost coverage with valid observer/sort-helper coverage so `internal/automation` stayed above 80%.

## Ready for Next Run
- Task 06 is complete. Later transport and extension tasks should consume `RuntimeDeps.Automation` instead of constructing their own automation runtime, and should rely on the manager status surface plus existing daemon fan-outs rather than adding parallel lifecycle wiring.
