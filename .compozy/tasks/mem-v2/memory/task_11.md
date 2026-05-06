# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Slice 1 Dreaming v2 promotion semantics without reopening extractor/controller architecture.
- Required behavior: Time -> Sessions -> Lock -> recall-signal gate; controller-backed curated promotion; `_system/` artifact/DLQ paths; idempotent `promoted_at`.

## Important Decisions

- Added `DreamGateConfig` and catalog-backed `dreamCandidates` scoring over `memory_recall_signals`; defaults match the TechSpec gate intent: 5 candidates, recall_count >= 2, score >= 0.75, 14-day half-life, weights 0.30/0.35/0.20/0.15.
- Kept `ShouldRun` side-effect free for Time/Sessions and enforced the new signal gate inside `Run` after lock acquisition so lock ordering remains Time -> Sessions -> Lock -> Signal.
- Wrote synthesized run artifacts directly under `_system/dreaming/YYYYMMDD-dreaming-curator.md` and failures under `_system/dream/failures/<run_id>.json`; these paths bypass prompt-facing filename scans and remain non-injectable.
- Drove curated promotion through `Store.ProposeCandidate` with `OriginDreaming`; controller now bypasses multi-source surface ambiguity for dreaming-origin candidates because dreaming syntheses intentionally consolidate multiple source memories.
- `memory_consolidations` and `memory_events` now record dream started/promoted/failed outcomes without schema changes.
- `consolidation.NewSessionSpawner` uses `dreaming-curator` when config is blank or still the generic default agent; explicit non-default dream agents remain honored until task_13 finalizes config lifecycle.

## Learnings

- Dreaming synthesis content naturally overlaps multiple source memories; treating that as regular surface ambiguity converts valid promotions into noop decisions.
- `internal/memory` package coverage remains near the floor; task-local helper coverage was needed to keep the package at 80.0%.

## Files / Surfaces

- Production: `internal/memory/dream.go`, `internal/memory/dream_v2.go`, `internal/memory/controller/controller.go`, `internal/memory/consolidation/runtime.go`.
- Tests: `internal/memory/dream_test.go`, `internal/memory/consolidation/runtime_test.go`.

## Errors / Corrections

- Initial focused test showed promotions were not stamping recall signals because controller surface ambiguity returned noop for multi-source dreaming content. Fixed by allowing `OriginDreaming` to bypass the multi-target surface ambiguity rule while preserving exact/filename/entity-slot rules.
- `make lint` caught `Service.Run` over `funlen`, two long signatures, and dead test assignment; refactored into gate/promotion/failure helpers and removed the dead assignment.
- Coverage initially reported `internal/memory` at 79.8%; added behavior tests for scoring and `_system` path validation.

## Ready for Next Run

- `task_11` is complete with full `make verify` PASS.
- Next task is `task_12` (Session Lineage and Ledger Materialization).
- Open dependency: task_19 still owns concrete daemon boot/lifecycle attachment for dreaming/extractor runtimes.
