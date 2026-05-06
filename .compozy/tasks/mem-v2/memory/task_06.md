# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the deterministic Slice 1 recall path with FTS5/trigram candidate retrieval, stable packaging, live recall-signal persistence, and scope-aware shadow-by-id.

## Important Decisions

- `internal/memory/recall` is the pure deterministic ranking/packaging package. It depends on `internal/memory/contract`, receives candidates through a source interface, and owns trivial-query skip, score fusion, scope precedence, stable headers, freshness banners, and failure-safe side effects.
- `Store.Recall` is the storage-backed entry point for runtime consumers. It resolves workspace identity, ensures the derived catalog is ready, queries `memory_chunks_fts` plus `memory_chunks_fts_trigram`, and maps rows into recall candidates.
- Recall packages group blocks least-specific first while top-K selection is still ranking-based. Shadow-by-id uses `(type, slug)` and deeper scope wins: global < workspace < agent-global < agent-workspace.
- Prompt augmentation now consumes `memcontract.Packaged` from `Store.Recall`; it no longer reranks via the legacy `Search` path.
- `memory_recall_signals` remains live in Slice 1. Recall writes update count, last recalled timestamp, EMA-like recall score, freshness barrier, and surfaced IDs without bubbling update failures to the recall caller.

## Learnings

- The existing daemon prompt augmentation fixture used a two-token query (`auth migration`), which is now intentionally trivial under ADR-011. Tests that expect recall must use at least three meaningful terms or a non-ASCII query with sufficient length.
- `internal/memory` coverage is sensitive to new storage source code; adding real no-catalog, nil-context, signal-failure, migration, and CJK recall paths kept coverage above the 80% package floor.
- FTS5 trigram successfully covers Japanese substring recall when the query is a long non-ASCII token and unicode tokenization alone would be insufficient.

## Files / Surfaces

- `internal/memory/recall/`
- `internal/memory/recall_source.go`
- `internal/memory/recall.go`
- `internal/memory/catalog.go`
- `internal/memory/store_test.go`
- `internal/memory/recall_test.go`
- `internal/daemon/prompt_input_composite_test.go`

## Errors / Corrections

- First focused recall test pass showed `NewRecallAugmenter` returned the original message when the test store lacked a catalog DB; the test now enables `WithCatalogDatabasePath`, matching the chunk-backed recall requirement.
- Initial shadow integration expected agent memory to shadow workspace/global entries but used the feedback-typed helper; the fixture now uses a project-typed agent memory so the `(type, slug)` identity matches.
- First `make verify` failed in `TestPromptInputCompositeIncludesDurableMemoryRecall` because the test used the now-trivial `auth migration` query and fixture text did not contain `sessions`; the query and fixture now satisfy the deterministic recall gate.
- A staticcheck suggestion warning around a literal nil context was avoided by using the existing nil-context helper while still covering the validation path.

## Ready for Next Run

- Task 06 passed focused tests, race tests, coverage, `git diff --check`, and full `make verify` before tracking updates.
- Next loop iteration should execute `task_07` (Local Provider and Registry Surface).
