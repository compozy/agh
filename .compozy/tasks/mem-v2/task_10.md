---
status: completed
title: Extractor Hook, Inbox, and Runtime Queue
type: backend
complexity: critical
dependencies:
  - task_05
  - task_09
---

# Task 10: Extractor Hook, Inbox, and Runtime Queue

## Overview

Implement the Slice 1 extractor execution path: a typed persisted-message hook, daemon-owned inbox semantics, and the bounded runtime queue that feeds controller proposals. This task makes post-response extraction real without coupling it to streaming internals or allowing `_inbox/` ownership to drift across packages.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Extractor`, `Safety Invariants`, `_system/` invariants, and `Development Sequencing` steps 19-20.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside hook dispatch, extractor runtime, and inbox ownership seams.
- TESTS REQUIRED: persisted-message dispatch, queue backpressure/coalescing, DLQ behavior, controller handoff, and shutdown cleanup must ship here.
- NO WORKAROUNDS: do not tail session DB tables asynchronously to discover extractor triggers.
</critical>

<requirements>
- MUST add the typed persisted-message hook/event at the actual durable call site.
- MUST implement daemon-owned `_inbox/` consumption semantics and bounded queue/coalescing behavior.
- MUST route extractor outputs into controller proposals rather than direct writes.
- MUST add DLQ/failure handling under `_system/` without polluting prompt-facing memory surfaces.
- MUST keep runtime ownership with the manager/daemon lifecycle so queue workers shut down cleanly.
</requirements>

## Subtasks
- [x] 10.1 Add the persisted-message hook event and dispatch it at the durable session-message boundary.
- [x] 10.2 Implement the extractor runtime queue, coalescing, and controller handoff.
- [x] 10.3 Add daemon-owned `_inbox/` semantics and DLQ/failure outputs under `_system/`.
- [x] 10.4 Add cleanup/backpressure/shutdown tests for the extractor runtime.

## Implementation Details

See TechSpec `Extractor`, `Safety Invariants`, and `Development Sequencing` steps 19-20. The extractor must follow the durable transcript boundary, not provider-stream deltas, and must stay manager-owned for lifecycle cleanup.

### Relevant Files
- `internal/hooks/events.go` — hook taxonomy that must gain the persisted-message event.
- `internal/session/hook_dispatch_events.go` — session-side hook dispatch call sites.
- `internal/daemon/hook_dispatch_events.go` — daemon-side hook wiring patterns for lifecycle events.
- `internal/store/sessiondb/session_db.go` — durable session transcript boundary that should trigger extraction.
- `internal/daemon/boot.go` — later daemon wiring will attach the extractor runtime here.
- `.compozy/tasks/mem-v2/analysis/analysis_extraction-location.md` — extractor design evidence.

### Dependent Files
- `internal/memory/controller/*` — extractor outputs must become controller proposals.
- `internal/memory/dream.go` — later dreaming work benefits from controller-applied extracted memories, not a parallel write path.
- `.compozy/tasks/mem-v2/task_14.md` — public contract work may expose extractor-related state and failures.
- `.compozy/tasks/mem-v2/task_19.md` — daemon wiring task depends on extractor runtime readiness.

### Related ADRs
- [ADR-010: Fact Extraction Location — Hybrid Per-Turn Hook + Optional Compaction Flush](adrs/adr-010.md) — normative extractor behavior.
- [ADR-005: `_system/` Namespace Invariant](adrs/adr-005.md) — constrains DLQ/failure output placement.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — requires extractor outputs to route through the controller.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task establishes the call-site hook and runtime queue semantics that later providers/extensions must integrate with.
- Agent manageability: no public control surface lands here yet, but later surfaces must reflect daemon ownership of `_inbox/` and extractor failures.
- Config lifecycle: none — checked surfaces are extractor model/mode settings and docs; public config work is deferred to `task_13`.

### Web/Docs Impact

- `web/`: none yet — checked surfaces are knowledge/settings/session UI and generated types; extractor-related UI depends on later public contracts.
- `packages/site`: none yet — checked surfaces are runtime memory/hooks/session docs; docs update after public manageability surfaces land.

## Deliverables

- Persisted-message hook event wired at the durable transcript boundary.
- Extractor runtime queue with bounded/coalescing behavior and controller handoff.
- Daemon-owned `_inbox/` and DLQ/failure outputs under `_system/`.
- Focused runtime cleanup/backpressure coverage.

## Tests

- Unit tests:
  - [x] Persisted-message events emit exactly at the durable transcript boundary and not on transient stream deltas.
  - [x] Queue coalescing and backpressure behavior respect the approved capacity and drop/merge rules.
  - [x] Extractor outputs become controller proposals and never direct writes.
- Integration tests:
  - [x] Shutdown/join behavior cleans up extractor workers and queue resources without leaks.
  - [x] DLQ writes land under `_system/` and stay out of prompt-facing memory packaging.
  - [x] A realistic post-response flow produces extracted proposals through the controller seam.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/claude-code/services/extractMemories/extractMemories.ts`
- `.resources/hermes/tools/memory_tool.py`
- `.resources/codex/codex-rs/memories/write/src/start.rs`
- `.resources/codex/codex-rs/memories/write/src/phase2.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Slice 1 extractor behavior runs from the durable message boundary through a daemon-owned inbox/queue path.
- No parallel or hidden mutation path bypasses the controller.
