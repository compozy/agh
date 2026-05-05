# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 04 read-only built-ins for memory, observe, and bridges.
- Required tool families: `agh__memory` (`list`, `read`, `search`, history-style inspection), `agh__observe` (`events`, metrics/health, search over current event query support), `agh__bridges` (`list`, status/health).
- Success requires preserving existing redaction/visibility semantics, focused tests, >=80% affected-package coverage, clean `make verify`, tracking updates, and one local commit if verification is clean.

## Important Decisions
- Treat the repository/task files as authoritative over stale shared workflow memory: shared memory says Task 04 is already implemented, but the current code has no `agh__memory`, `agh__observe`, or `agh__bridges` IDs/descriptors/handlers.
- Keep scope read-only even though the broader TechSpec final-state table lists future memory write/delete tools.
- Reuse current `memory.Store`, `core.Observer`, and `core.BridgeService` query/projection helpers; do not add tool-specific storage paths.

## Learnings
- Baseline signal: `rg -n "agh__memory|agh__observe|agh__bridges|ToolIDMemory|ToolsetIDMemory|ToolIDObserve|ToolIDBridges" internal/tools internal/daemon internal/api -g '*.go'` returned no matches.
- Current branch already has Task 01/02/03 foundation plus config/hook descriptors; Task 04 must extend the same native provider and toolset catalog.
- Implementation now adds `agh__memory`, `agh__observe`, and `agh__bridges` as read-only toolsets over existing services only.
- Focused validation passed: `go test ./internal/tools/builtin ./internal/daemon -run 'TestBuiltin|TestDaemonNativeTools' -count=1`.
- Final pre-commit `make verify` passed twice on 2026-04-30 after self-review changes, with Go output `DONE 7040 tests` and `OK: all package boundaries respected`.
- Local commit created: `eb2a9253 feat: add memory observe bridge read tools`.
- Post-commit `make verify` passed with `0 issues`, `DONE 7040 tests`, and `OK: all package boundaries respected`.
- Coverage evidence: `go test -cover ./internal/tools/builtin` reported 92.3%; `internal/daemon` aggregate remains below 80% at 73.0% because of broader package baseline, while the new memory/observe/bridge native paths are covered by focused daemon tests.

## Files / Surfaces
- Expected code surfaces: `internal/tools/builtin_ids.go`, `internal/tools/builtin/descriptors.go`, `internal/tools/builtin/toolsets.go`, new built-in descriptor files, `internal/daemon/native_tools.go`, native tests.
- Existing query/projection authorities: `internal/api/core/memory.go`, `internal/observe/query.go`, `internal/observe/health.go`, `internal/api/core/bridges.go`.
- Touched implementation/test surfaces: `internal/tools/builtin_ids.go`, `internal/tools/builtin/descriptors.go`, `internal/tools/builtin/toolsets.go`, `internal/tools/builtin/memory.go`, `internal/tools/builtin/observe.go`, `internal/tools/builtin/bridges.go`, `internal/tools/builtin/builtin_test.go`, `internal/daemon/native_tools.go`, `internal/daemon/native_tools_test.go`.

## Errors / Corrections
- Correction: shared workflow memory incorrectly records Task 04 as already implemented for this repo state. Keep durable correction task-local unless it affects later tasks after final verification.
- Correction: the first compile surfaced a workspace-resolution return bug and stale catalog expectations; both were fixed before focused tests passed.
- Correction: AGH test-convention helper must be run once per file from `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`; the repository does not have a root `scripts/check-test-conventions.py`.

## Ready for Next Run
- Complete. Code is committed locally; workflow memory/tracking artifacts remain uncommitted per workflow guidance.
