# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 05 end to end: add memory list/read/write/delete/consolidate endpoints to `internal/httpapi` and `internal/udsapi`, wire required runtime deps from `daemon`, add `agh memory` CLI commands plus daemon client methods, extend observe health responses with memory stats, and cover the new behavior with unit/integration tests before tracking updates and commit.

## Important Decisions
- Treat the task spec + techspec API/CLI tables as the approved design baseline; no separate design fork is needed unless implementation conflicts surface.
- Reuse the shared daemon `RuntimeDeps.MemoryStore` and add a narrow dream-trigger dependency for consolidate handlers instead of creating new memory store instances in API layers.
- Keep memory health stats as extra handler response payload derived from the shared memory store and dream service state.
- Keep `GET /api/observe/health` in the top-level `{health, memory}` response shape used by the techspec and the new handler tests.

## Learnings
- `internal/memory.Store` is immutable per workspace via `Store.ForWorkspace(workspaceRoot)`; workspace API requests must bind a clone instead of mutating the shared store.
- `memory.ErrValidation` is the correct signal for HTTP 400 / CLI usage-style errors; not-found should still map through `os.ErrNotExist`.
- Existing HTTP/UDS/CLI packages already have parallel patterns for request binding, response envelopes, route registration, and test helpers that can be extended directly.
- The CLI integration daemon needs the same `udsapi.WithMemoryStore(...)` and `udsapi.WithDreamTrigger(...)` wiring as the dedicated HTTP/UDS integration harnesses; otherwise the new memory routes exist but fail at runtime with `memory store is not configured`.
- File-level coverage after the final test pass is above the task target for the new task-owned memory files: `internal/httpapi/memory.go` 84.5%, `internal/udsapi/memory.go` 84.5%, `internal/cli/memory.go` 80.3%.

## Files / Surfaces
- `internal/httpapi/{server.go,observe.go,helpers_test.go,handlers_test.go,handlers_error_test.go}`
- `internal/udsapi/{routes.go,handlers.go,helpers_test.go,handlers_test.go,handlers_error_test.go,udsapi_integration_test.go}`
- `internal/cli/{client.go,helpers_test.go,command_paths_test.go,cli_integration_test.go,root.go,format.go,memory.go,memory_test.go}`
- `internal/daemon/daemon.go`
- `internal/memory/{store.go,types.go,dream.go,document.go}`

## Errors / Corrections
- Fixed a real test-harness gap after the first full CLI integration run: `TestMemoryWriteListIntegration` exposed that the CLI daemon fixture had picked up the new UDS memory routes without injecting `MemoryStore` or `DreamTrigger`, so the command failed before verification could complete.
- The first `make verify` pass failed on an unused field in the HTTP memory test stub; removing that dead field cleared the lint gate without changing runtime behavior.

## Ready for Next Run
- Verification is complete: `go test -race -cover ./internal/httpapi ./internal/udsapi ./internal/cli ./internal/daemon ./internal/memory -count=1`, `go test -race -tags integration ./internal/httpapi ./internal/udsapi ./internal/cli -count=1`, and `make verify` all pass. Remaining work is task tracking cleanup and the code-only local commit.
