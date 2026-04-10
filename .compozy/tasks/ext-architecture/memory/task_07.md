# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented `internal/extension/host_api.go` with `HostAPIHandler`, all 12 Host API methods, capability enforcement, typed JSON-RPC errors, and per-extension rate limiting.
- Verified with `go test ./internal/extension -count=1`, `go test -tags integration ./internal/extension -count=1`, `go test ./internal/extension -cover` (`80.2%`), and `make verify`.

## Important Decisions
- Host API failures use `*subprocess.RPCError` directly so protocol errors stay transport-aligned for capability denial, rate limiting, invalid params, and unknown methods.
- `Manager.wrapHostHandler` now injects the extension name into handler context and converts manager-side capability denials to typed RPC errors instead of leaking raw Go errors.
- `memory/store`, `memory/recall`, and `memory/forget` adapt the existing markdown-backed `memory.Store`; tags persist via an `<!-- agh-tags: ... -->` comment and recall scores the rendered body plus tags without adding a new persistence surface.
- `skills/list` resolves workspace-scoped skills through an injected workspace resolver, and `sessions/events` accepts both `since` and `offset` inputs to match the task and protocol expectations.

## Learnings
- Observer-backed `observe/events` integration requires the test harness to seed the workspace row in the shared global DB before creating sessions, otherwise session and event-summary foreign keys block end-to-end reads.
- Fixed-clock integration tests must derive `since` filters from the harness clock (`env.now`) instead of `time.Now()` to avoid accidentally filtering out synthetic events.

## Files / Surfaces
- `internal/extension/host_api.go`
- `internal/extension/manager.go`
- `internal/extension/host_api_test.go`
- `internal/extension/host_api_integration_test.go`

## Errors / Corrections
- Corrected the initial test harness to pass `testing.TB` explicitly instead of relying on a nonexistent implicit helper and cleaned up the workspace ID plumbing.
- Corrected the observe integration path by sharing a seeded `globaldb` registry with the observer in tests.

## Ready for Next Run
- Task 07 is implementation-complete and verified. Task 08 needs to wire `HostAPIHandler` into daemon boot with the real session manager, memory store, observer, skills registry, capability checker, and workspace resolver.
