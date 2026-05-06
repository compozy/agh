---
status: completed
title: Memory Observability and SSE Hygiene
type: backend
complexity: critical
dependencies:
  - task_02
  - task_05
  - task_06
---

# Task 09: Memory Observability and SSE Hygiene

## Overview

Add the observability layer that Memory v2 needs across global and per-workspace databases, and harden memory-related streaming surfaces against `<memory-context>` leakage. This task closes the gap between new durable memory authorities and the operator-facing event/health surfaces that must remain truthful and safe.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Monitoring and Observability`, `Safety Invariants`, and `Development Sequencing` steps 16 and 33.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside observe/SSE/health seams; public route wiring changes happen later.
- TESTS REQUIRED: event aggregation, redaction/scrubbing, health-state derivation, and reconnect-safe streaming behavior must ship here.
- NO WORKAROUNDS: observability must aggregate the new DB topology instead of pretending memory still lives in one global log.
</critical>

<requirements>
- MUST emit and aggregate the canonical `memory_events` set across global and per-workspace databases.
- MUST update memory-health derivation to reflect the new authorities and workspace topology.
- MUST scrub `<memory-context>` and equivalent prompt-only payloads from SSE/log-facing surfaces.
- MUST preserve durable-append-before-broadcast behavior for memory-related observe streams.
- MUST leave later HTTP/UDS/public tasks with thin adapters over the final observability services.
</requirements>

## Subtasks
- [x] 9.1 Implement memory event aggregation over the new DB topology.
- [x] 9.2 Update health derivation and operator-facing observability helpers for Slice 1 memory behavior.
- [x] 9.3 Add `<memory-context>` scrubber behavior for streaming/log surfaces.
- [x] 9.4 Add focused tests for aggregation, redaction, broadcast ordering, and reconnect safety.

## Implementation Details

See TechSpec `Monitoring and Observability`, `Safety Invariants`, and `Development Sequencing` step 16. This task should deliver the domain service/aggregation behavior; the public transport exposure is finished later in `task_16`.

### Relevant Files
- `internal/store/globaldb/global_db_observe.go` — current observe aggregation rooted in the old single-log assumption.
- `internal/api/core/memory.go` — current memory health/history shaping that must adopt the new observability model.
- `internal/sse/decode.go` — current shared SSE helpers and a natural home for scrubber support.
- `internal/daemon/harness_observability.go` — daemon-level observability wiring and assertions.
- `internal/observe/hooks_test.go` — existing observability test patterns to mirror for memory events.
- `internal/api/contract/responses.go` — health/observe response envelopes affected by the new aggregation.

### Dependent Files
- `internal/api/httpapi/routes.go` and `internal/api/udsapi/routes.go` — later transport tasks will expose the final observe behavior.
- `internal/cli/memory.go` — later CLI hard cut will rely on final event/health semantics.
- `.compozy/tasks/mem-v2/task_10.md` — extractor task depends on stable memory event taxonomy.
- `.compozy/tasks/mem-v2/task_19.md` — daemon wiring task depends on the completed observability layer.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — defines audit/event authority.
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — clarifies live truth vs forensic projections.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — constrains recall-event semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: memory events and scrubbed observability outputs become the stable signals that extensions/providers must rely on rather than private logs.
- Agent manageability: public observe/history/health verbs are not cut over yet, but later tasks must expose exactly this aggregated behavior.
- Config lifecycle: none — checked surfaces are observability settings, memory health payloads, and docs; public config work is deferred.

### Web/Docs Impact

- `web/`: none yet — checked surfaces are observability and session views; generated/public contract updates happen later.
- `packages/site`: none yet — checked surfaces are memory/observe/session docs and references; docs update once public outputs stabilize.

## Deliverables

- Cross-DB memory event aggregation and health derivation.
- `<memory-context>` SSE/log scrubber behavior for memory-related streams.
- Focused observability tests covering aggregation, ordering, and redaction.

## Tests

- Unit tests:
  - [x] Memory event aggregation merges global/workspace sources without double-counting or authority drift.
  - [x] Health derivation reflects disabled, degraded, unavailable, and healthy Slice 1 memory states correctly.
  - [x] `<memory-context>` scrubber removes prompt-only content without damaging unrelated streaming payloads.
- Integration tests:
  - [x] Observe/health services broadcast only after durable append and remain reconnect-safe.
  - [x] Workspace-scoped and global event queries show consistent state across the new DB topology.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/agent/memory_manager.py`
- `.resources/hermes/run_agent.py`
- `.resources/codex/codex-rs/memories/write/src/metrics.rs`
- `.resources/claude-code/memdir/findRelevantMemories.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 observability reflects the new authorities and DB topology truthfully.
- Prompt-only memory context does not leak through SSE/log-facing surfaces.
