# Task Memory: task_19.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Wire Memory v2 Slice 1 subsystems at the daemon composition root: local provider/provider registry, snapshot prompt assembly, extractor runtime/service, dreaming runtime dependencies, session ledger materializer, and API/UDS/HTTP service adapters.
- Register package-boundary rules for the new memory/session/store packages and verify they pass under the repo boundary gate.

## Important Decisions

- Keep all concrete runtime assembly in `internal/daemon`; subordinate memory packages expose narrow services/adapters and do not import daemon/API transport packages.
- The daemon-owned local provider uses the `memory.Store` adapter from `internal/memory/provider/local/memstore` and is passed into prompt snapshots/provider registry instead of duplicating recall or file traversal logic.
- The extractor is daemon-owned and starts only from `Run`; `Close` is safe for boot-only tests because it drains runtime state without waiting for a never-started consumer goroutine.
- Provider snapshot requests now carry `workspace_root` in addition to `workspace_id` so the bundled local provider can render workspace/agent workspace prompt blocks through the same provider path without bypassing daemon wiring.

## Learnings

- Race-enabled daemon verification exposed real lifecycle bugs that focused hook tests did not catch: typed-nil local provider assignment during publish, extractor close waiting on an unstarted goroutine, and test session managers missing the spawn surface required by the extractor.
- `internal/daemon -race -parallel=4` is the cheapest reliable preflight for task 19 because the full gate runs that same profile.
- `make verify` passed after the lifecycle fixes: Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8359 tests in 127.259s`, and package boundaries `OK`.
- Post-state `make verify` also passed after task tracking/state updates: Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8359 tests in 15.520s`, and package boundaries `OK`.

## Files / Surfaces

- `internal/daemon/boot.go`, `internal/daemon/daemon.go`, `internal/daemon/memory_runtime.go`, `internal/daemon/hooks_bridge.go`, `internal/daemon/boundary.go`, `internal/daemon/daemon_test.go`, `internal/daemon/notifier_test.go`
- `internal/api/core/{interfaces,handlers,memory}.go`, `internal/api/core/memory_services_test.go`, `internal/api/httpapi/{handlers,server}.go`, `internal/api/udsapi/server.go`
- `internal/memory/extractor/{runtime,inbox}.go`, `internal/memory/contract/types.go`, `internal/memory/snapshot.go`, `internal/memory/provider/local/provider.go`, `internal/memory/provider/local/memstore/memstore.go`
- `magefile.go`

## Errors / Corrections

- Fixed `funlen` lint in `NewBaseHandlers`, `bootPromptProviders`, and `bootRuntimeServices` by extracting local helpers while preserving composition-root ownership.
- Fixed `goconst` in the fallback API provider payload by centralizing the local provider name constant.
- Fixed daemon shutdown failures by avoiding typed-nil local provider assignment and by waiting for extractor `done` only when the consumer was actually started.
- Fixed workspace prompt assembly through the local provider by threading `workspace_root` into provider snapshot requests and adding a `ForWorkspace` backend seam.

## Ready for Next Run

- Task 19 is complete and verified.
- Next task should be `task_20` (Web Knowledge and Memory UX) after the loop state advances.
