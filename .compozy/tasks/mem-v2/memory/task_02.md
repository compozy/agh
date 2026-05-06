# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 02: numbered Slice 1 memory schema migrations, stable `.agh/workspace.toml` workspace_id resolution, idempotent backfill from legacy path-keyed `workspace_root`, removal/de-authorization of path-keyed authority, and focused fresh/migrated/reopen/idempotent tests.
- Completion requires clean task validation, self-review, `make verify`, tracking updates, and one local commit.

## Important Decisions

- Task 02 is a foundation task only; public CLI/API/web/docs/provider/runtime wiring from the full TechSpec remains in later tasks unless required by schema or resolver tests.
- `workspace_id` is the only durable workspace-scoped memory owner after migration. Legacy `workspace_root` can only be used as bounded migration input.
- Resolver identity follows ADR-004 post-review refinement: ULID in `<workspace>/.agh/workspace.toml`; permission-denied/invalid identity fails closed.

## Learnings

- Task 01 already extracted `internal/memory/contract`; this task should import contract enums/types instead of reintroducing local memory DTOs.
- TechSpec Data Models require Slice 1 DDL primitives for `memory_catalog_entries`, `memory_chunks` + both FTS tables/triggers, `memory_events`, `memory_decisions`, `memory_recall_signals`, and `memory_consolidations`.
- Backfill coverage caught two production migration defects: legacy catalog scans must include `workspace_root`, and rebuilt catalog rows must insert into `memory_catalog_entries_new` before the rename.

## Files / Surfaces

- Expected primary surfaces: `internal/store/schema.go`, `internal/store/globaldb/migrate_workspace.go`, `internal/workspace/resolver.go`, `internal/workspace/workspace.go`, `internal/memory/catalog.go`, and focused tests under those packages.
- Current implementation touches workspace identity/resolver helpers, memory catalog/store schema and history paths, and global DB memory event migration/observation tests.

## Errors / Corrections

- Corrected legacy global memory-operation migration to resolve path-keyed `workspace_root` values through `.agh/workspace.toml` before writing `memory_events.workspace_id`.
- Corrected tests that still asserted or mutated legacy `memory_operation_log.workspace_root` after the durable event table hard cut.
- First full `make verify` rerun failed only because `sdk/typescript/src/integration.test.ts` timed out at 30s under the full Vitest suite; isolated rerun of the exact file passed in 166ms with no code changes, so treat as a load-sensitive verification flake unless it reproduces consistently.
- Second full `make verify` reproduced the same SDK integration timeout. Correction: set `sdk/typescript` Vitest project `fileParallelism: false` so subprocess-based SDK integration tests do not compete with sibling SDK test files for workers/artifacts under the root suite.
- After the SDK scheduling correction, `bun run test` passed from the repository root with 329 test files and 2088 tests.
- Full `make verify` then failed in Go race tests on two identity-contract mismatches. Corrections: API memory history test now expects the resolved `.agh/workspace.toml` ID, and daemon native-tools memory test creates the workspace root before writing workspace-scoped memory.
- Affected Go package race validation now passes: `go test -race -parallel=4 ./internal/api/core ./internal/daemon ./internal/memory ./internal/store/globaldb`.
- Full `make verify` still reproduced the SDK subprocess integration timeout under verification load. Correction: raised only `sdk/typescript/src/integration.test.ts` real-stdio timeout from 30s to 120s, matching the existing SDK build timeout without changing assertions.
- SDK project validation passed after timeout adjustment: `bunx vitest run --project extension-sdk --reporter verbose`.
- Pre-commit full verification now passes: `make verify`.
- Code-only implementation commit created: `96bb1d3f` (`feat: add memory v2 workspace identity schema`).
- Post-commit full verification passed: `make verify` completed with frontend tests `329 passed (329)`, Go tests `DONE 8090 tests`, `0 issues`, and package boundaries `OK`.

## Ready for Next Run

- Task 02 implementation is complete and verified. Tracking/workflow memory updates are intentionally left uncommitted per task instruction to keep tracking-only files out of the automatic commit.
