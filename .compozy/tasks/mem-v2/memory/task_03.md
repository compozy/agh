# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Execute Task 03: add the Memory v2 storage substrate without public controller/API wiring.
- Success criteria: per-workspace DB opener, atomic file writes, jittered SQLite `BEGIN IMMEDIATE` writes, workspace/global/agent store roots, replay/reindex helpers, crash/restart/replay coverage, >=80% focused coverage, and full `make verify` pass.

## Important Decisions

- `internal/store/workspacedb` owns workspace DB open/migrate lifecycle and opens `<workspace>/.agh/agh.db` after resolving the durable `.agh/workspace.toml` `workspace_id`.
- `store.ExecuteWrite` uses a single SQL connection, `BEGIN IMMEDIATE`, bounded `SQLITE_BUSY`/`SQLITE_LOCKED` retry jitter, rollback on callback failure, commit on success, and periodic checkpointing.
- Memory catalog mutation paths now run through `store.ExecuteWrite`; later controller tasks should not add parallel ad hoc write-transaction paths.
- Replay stays storage-local: `Store.ReplayPendingDecisions(ctx)` reads unapplied `memory_decisions`, applies idempotent file mutations/reindexing, and stamps `applied_at`; daemon/controller wiring remains for later tasks.
- Replay preserves `post_content` bytes when hashing/writing add/update decisions. Trimming the body would corrupt the stored-content hash contract.

## Learnings

- The `agh-test-conventions` skill references `scripts/check-test-conventions.py`, but this repository currently has no matching file. Validation used focused Go tests, race tests, coverage checks, and `make verify`.
- Staticcheck emits autofix warnings for direct `nil` context arguments even inside validation tests. The workspacedb test keeps invalid-root/nil-receiver coverage and avoids direct nil-context calls.
- `make verify` still emits existing non-fatal Vite chunk-size and macOS linker warnings while exiting 0 with lint/test/build/boundary success.

## Files / Surfaces

- Added storage runtime: `internal/store/write.go`, `internal/store/write_test.go`, `internal/store/workspacedb/workspace_db.go`, `internal/store/workspacedb/workspace_db_test.go`, and `internal/store/memv2_coverage_test.go`.
- Added atomic file helper coverage: `internal/fileutil/atomic.go` and `internal/fileutil/atomic_memv2_test.go`.
- Extended memory storage/catalog surfaces: `internal/memory/store.go`, `internal/memory/catalog.go`, `internal/memory/replay.go`, and `internal/memory/store_memv2_test.go`.
- Touched shared store helper: `internal/store/schema.go`.
- Touched `internal/skills/registry.go` only to remove a global lint blocker for the repeated `"workspace"` source string.

## Errors / Corrections

- First full `make verify` run reached Go lint and failed on funlen, goconst, `ifElseChain`, SQL rows leak, G115, line length, unused closure context, indent-error-flow, context-argument ordering, and nil-context staticcheck warnings.
- Corrections were structural: extracted `scanCandidates`, introduced `skillSourceWorkspaceName`, closed rows on unexpected query success, switched random jitter to `crypto/rand.Int`, used closure context, removed `else` after error return, reordered test helper args, and removed direct nil-context tests.
- Focused validation after corrections passed: `go test ./internal/store ./internal/store/workspacedb ./internal/fileutil ./internal/memory -count=1`.
- Race validation passed: `go test -race ./internal/store ./internal/store/workspacedb ./internal/fileutil ./internal/memory -count=1`.
- Focused coverage passed after final test adjustment: `internal/store` 82.3%, `internal/store/workspacedb` 91.2%, `internal/fileutil` 87.3%, `internal/memory` 80.1%.
- Full gate passed: `make verify` completed with Bun tests `329 passed (329)`, Go tests `DONE 8137 tests in 120.593s`, Go lint `0 issues`, and `OK: all package boundaries respected`.

## Ready for Next Run

- Task 03 implementation and verification are complete.
- Task 04 should start from the existing Memory v2 storage primitives: `store.ExecuteWrite`, `workspacedb.Open/OpenWorkspace`, agent-aware memory roots, and `Store.ReplayPendingDecisions`.
- Remaining Memory v2 tasks `task_04` through `task_26` are still pending.
