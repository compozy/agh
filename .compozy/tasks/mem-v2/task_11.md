---
status: pending
title: Dreaming Runtime and Promotion Gates
type: backend
complexity: critical
dependencies:
  - task_05
  - task_06
  - task_07
---

# Task 11: Dreaming Runtime and Promotion Gates

## Overview

Extend the existing dreaming runtime to the Slice 1 model: recall-signal-informed promotion, dedicated dreaming agent behavior, and `_system/` failure handling. This task upgrades consolidation from the current simpler runtime into the approved v2 gate stack without reopening the controller or extractor architecture.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Dreaming v2`, `Data Models`, `_system/` invariants, and `Development Sequencing` steps 21-22.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside dreaming/consolidation seams and their controller/provider integrations.
- TESTS REQUIRED: gate evaluation, signal-threshold promotion, dedicated-agent flow, DLQ behavior, and idempotent `promoted_at` handling must ship here.
- NO WORKAROUNDS: keep the approved Time → Sessions → Lock cascade and add the signal gate instead of replacing the existing runtime with heuristics.
</critical>

<requirements>
- MUST extend the current dreaming runtime with the Slice 1 promotion gates and scoring semantics.
- MUST consume recall signals and controller/provider outputs rather than inventing a second retrieval pipeline.
- MUST drive promotions back through the controller/local provider path.
- MUST write dreaming failures and retries under `_system/` without polluting prompt-facing memory state.
- MUST preserve idempotent promotion bookkeeping and restart safety.
</requirements>

## Subtasks
- [ ] 11.1 Extend dreaming gate/scoring behavior with recall-signal-informed promotion logic.
- [ ] 11.2 Wire dedicated dreaming-agent behavior and controller-backed promotion writes.
- [ ] 11.3 Add `_system/` failure/retry handling and idempotent promotion markers.
- [ ] 11.4 Add focused dreaming tests for gate evaluation, promotion, restart safety, and failure handling.

## Implementation Details

See TechSpec `Dreaming v2`, `Monitoring and Observability`, and `Development Sequencing` steps 21-22. This task should evolve the existing dreaming runtime rather than replacing it wholesale or introducing a second promotion path.

### Relevant Files
- `internal/memory/dream.go` — current dreaming logic to extend with Slice 1 gates and promotion behavior.
- `internal/memory/dream_test.go` — existing dreaming coverage to expand.
- `internal/memory/consolidation/runtime.go` — current runtime gating/wiring that must add the signal gate.
- `internal/memory/consolidation/runtime_test.go` — runtime coverage for gate sequencing and restart behavior.
- `internal/memory/lock.go` — existing lock semantics that must remain intact.
- `internal/daemon/boot.go` — later daemon wiring task depends on the finished runtime.

### Dependent Files
- `internal/config/config.go` — later config task depends on final dreaming keys and defaults.
- `internal/api/contract/settings.go` — later settings/public contract task depends on dreaming metadata.
- `.compozy/tasks/mem-v2/task_13.md` — config/settings backend depends on final dreaming knobs.
- `.compozy/tasks/mem-v2/task_19.md` — daemon wiring task depends on the completed dreaming runtime.

### Related ADRs
- [ADR-007: Daily-Log Retention Policy](adrs/adr-007.md) — retention and dreaming-window constraints.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — constrains signal semantics.
- [ADR-005: `_system/` Namespace Invariant](adrs/adr-005.md) — constrains dreaming failure outputs.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: bundled and future external providers must be able to participate in dreaming via the provider ABC rather than custom side paths.
- Agent manageability: no public trigger/status surface lands here yet, but later CLI/API/native-tool routes must reflect these exact gate semantics.
- Config lifecycle: dreaming-specific config work is deferred to `task_13`, but this task defines the runtime semantics those keys will control.

### Web/Docs Impact

- `web/`: none yet — checked surfaces are settings memory page and session/knowledge views; public dreaming metadata lands later.
- `packages/site`: none yet — checked surfaces are runtime memory/config docs and CLI/API references; docs update after public surfaces stabilize.

## Deliverables

- Extended dreaming runtime with signal-aware promotion gates.
- Dedicated dreaming-agent behavior and controller-backed promotion writes.
- `_system/` failure/retry handling with focused coverage.

## Tests

- Unit tests:
  - [ ] Time, sessions, lock, and signal-threshold gates evaluate in the approved order.
  - [ ] Promotion writes route through controller/provider seams and set `promoted_at` idempotently.
  - [ ] Failure and retry paths write the expected `_system/` outputs without leaking prompt-facing content.
- Integration tests:
  - [ ] Dreaming restarts safely after interrupted runs and does not double-promote.
  - [ ] Recall signals and candidate scoring drive the expected promotion decisions in realistic fixtures.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/goclaw/internal/consolidation/dreaming_worker.go`
- `.resources/goclaw/internal/consolidation/scoring.go`
- `.resources/hermes/agent/memory_manager.py`
- `.resources/codex/codex-rs/memories/write/src/metrics.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Slice 1 dreaming runs on the approved gate stack and promotes through the controller/provider seam.
- Failure/retry behavior stays inside `_system/` and remains restart-safe.

