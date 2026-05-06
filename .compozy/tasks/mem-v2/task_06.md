---
status: completed
title: Deterministic Recall, Signals, and Shadow Rules
type: backend
complexity: critical
dependencies:
  - task_01
  - task_03
---

# Task 06: Deterministic Recall, Signals, and Shadow Rules

## Overview

Implement the deterministic Slice 1 recall path, including ranked retrieval, live recall signals, and shadow-by-id packaging rules. This task replaces the current mixed startup/turn-time recall behavior with a coherent recall subsystem that later prompt assembly, dreaming, public surfaces, and UI can consume consistently.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Recall pipeline`, `Data Models`, `Safety Invariants`, and `Monitoring and Observability`.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside recall/ranking/packaging seams; do not wire final public transports in this task.
- TESTS REQUIRED: ranking, trivial-query skip, shadow precedence, recall-signal writes, and packaging determinism must ship here.
- NO WORKAROUNDS: Slice 1 recall stays deterministic-first; no embeddings or cosine thresholds re-enter here.
</critical>

<requirements>
- MUST implement FTS5/trigram recall with deterministic ranking and trivial-query skip behavior.
- MUST persist live `memory_recall_signals` updates for later dreaming/promotion work.
- MUST enforce scope-aware shadow-by-id precedence when packaging recalled memory for consumers.
- MUST expose stable recall outputs for prompt assembly, search/history transports, and future trace surfaces.
- MUST keep vector/ranker behavior out of Slice 1 recall code and tests.
</requirements>

## Subtasks
- [x] 6.1 Build the deterministic recall/ranking package on top of the new catalog/chunks substrate.
- [x] 6.2 Add recall-signal writes and failure-safe signal update handling.
- [x] 6.3 Implement scope-aware shadow-by-id packaging rules and freshness handling.
- [x] 6.4 Add focused tests for ranking, trivial skips, shadow precedence, and signal recording.

## Implementation Details

See TechSpec `Recall pipeline`, `Data Models`, and `Development Sequencing` step 12. The output of this task should be reusable by prompt assembly, search endpoints, agent-manageability surfaces, and dreaming without each consumer reimplementing ranking or precedence rules.

### Relevant Files
- `internal/memory/recall.go` — current recall/ranking entry point to replace or refactor.
- `internal/memory/assembler.go` — current packaging helpers that later prompt assembly will consume.
- `internal/memory/staleness.go` — freshness/banner logic to align with Slice 1 defaults.
- `internal/memory/recall_test.go` — existing recall coverage to extend with deterministic ranking proofs.
- `internal/testutil/acpmock/testdata/memory_recall_fixture.json` — recall fixture data useful for integration-style tests.
- `.compozy/tasks/mem-v2/analysis/analysis_hermes.md` — recall competitor evidence.

### Dependent Files
- `internal/daemon/composed_assembler.go` — later prompt assembly depends on stable recall outputs.
- `internal/memory/dream.go` — dreaming promotion gates depend on live recall signals.
- `internal/api/core/memory.go` — search/history/trace public surfaces will later depend on recall DTOs.
- `.compozy/tasks/mem-v2/task_08.md` — frozen snapshot/prompt assembly depends on final packaging outputs.
- `.compozy/tasks/mem-v2/task_14.md` — public contract work depends on stable recall payloads.

### Related ADRs
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — normative recall behavior.
- [ADR-002: Three Scopes with Agent Two-Tier](adrs/adr-002.md) — defines precedence and shadow behavior.
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — defines the derived/indexed role of recall storage.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task defines the recall output/provider seam that bundled and future external providers must respect.
- Agent manageability: no public route/CLI change lands here yet, but later search/trace verbs must reuse this deterministic recall path.
- Config lifecycle: none — checked surfaces are recall config keys, freshness defaults, settings payloads, and docs; public config work is deferred.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types and knowledge/session UI; public recall payloads arrive later via codegen.
- `packages/site`: none — checked surfaces are runtime memory docs and API/CLI references; docs update after public contracts land.

## Deliverables

- Deterministic Slice 1 recall subsystem with ranked retrieval and packaging outputs.
- Live `memory_recall_signals` recording with failure-safe behavior.
- Shadow-by-id and freshness-aware packaging rules with focused coverage.

## Tests

- Unit tests:
  - [ ] Deterministic ranking returns stable top-K results for known fixtures.
  - [ ] Trivial or empty recall requests short-circuit with the approved skip behavior.
  - [ ] Shadow-by-id precedence picks the deeper scope winner without silent merge.
  - [ ] Recall-signal writes record the expected fields and handle update failures deterministically.
- Integration tests:
  - [ ] Recall output is consumable by prompt-assembly helpers without re-querying or re-ranking.
  - [ ] Search-like consumers can exercise the recall subsystem against fixture data and restart-safe storage.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/tools/session_search_tool.py`
- `.resources/hermes/agent/context_engine.py`
- `.resources/goclaw/internal/memory/recall_query.go`
- `.resources/claude-code/memdir/findRelevantMemories.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Slice 1 has one deterministic recall implementation with live signals and shadow-aware packaging.
- Later prompt, dreaming, transport, and UI work can consume recall outputs without duplicating ranking logic.

## Completion Notes

- Implemented `internal/memory/recall` as the deterministic Slice 1 recall package with trivial-query skip, weighted lexical/recency/signal ranking, scope-aware shadow-by-id, stable cache headers, freshness banners, and failure-safe signal/event side effects.
- Wired `Store.Recall` to chunk-backed FTS5 unicode + trigram retrieval, live `memory_recall_signals` updates, recall/skipped/signal-failure/shadow events, and prompt augmentation consumption via `Packaged`.
- Added catalog chunk maintenance on write/reindex plus migration 008 to backfill chunks and promote `memory_recall_signals` to the live scoring/promotion schema.
- Added focused coverage for deterministic ranking, trivial skips, shadow precedence, already-surfaced/system filters, CJK trigram recall, signal writes/failures, packaging determinism, schema upgrade, and daemon prompt augmentation.
