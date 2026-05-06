---
status: completed
title: Frozen Snapshot and Prompt Assembly
type: backend
complexity: high
dependencies:
  - task_06
  - task_07
---

# Task 08: Frozen Snapshot and Prompt Assembly

## Overview

Build the frozen snapshot and prompt-assembly layer that turns recall and provider outputs into stable session context. This task isolates how memory becomes prompt material at session boot and refresh boundaries so later daemon wiring and UI inspection can rely on one packaging model.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `System Architecture`, `Recall pipeline`, `Session ledger`, and `Development Sequencing` step 15.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside memory-assembly and prompt-composition seams.
- TESTS REQUIRED: frozen snapshot creation, refresh invalidation rules, packaging limits, and scope-aware prompt composition must ship here.
- NO WORKAROUNDS: sub-agents inherit the parent snapshot and remain read-only rather than rebuilding private memory state.
</critical>

<requirements>
- MUST build the Slice 1 frozen snapshot model that captures memory state at session boot.
- MUST assemble prompt-facing memory sections from recall/provider outputs using one deterministic packaging path.
- MUST preserve the approved snapshot refresh semantics (`agh memory reload` affects the next boot, not ad-hoc prompt mutation).
- MUST enforce size/freshness/packaging rules from the TechSpec when creating prompt sections.
- MUST leave daemon boot wiring thin so later composition-root work can call into this layer directly.
</requirements>

## Subtasks
- [x] 8.1 Implement frozen-snapshot data structures and refresh invalidation rules.
- [x] 8.2 Build prompt-assembly helpers that package recall/provider outputs into prompt-ready sections.
- [x] 8.3 Add focused tests for snapshot boot, refresh semantics, size caps, and scope-aware composition.
- [x] 8.4 Confirm sub-agent inheritance and read-only semantics remain enforceable through this layer.

## Implementation Details

See TechSpec `System Architecture`, `Recall pipeline`, and `Development Sequencing` step 15. This task should stop at service-level assembly behavior; full composition-root wiring happens later in `task_19`.

### Relevant Files
- `internal/memory/assembler.go` — current assembly logic that must evolve into the Slice 1 packaging model.
- `internal/daemon/composed_assembler.go` — daemon-side assembly consumer that later wiring will call.
- `internal/daemon/prompt_sections.go` — prompt-section composition helpers.
- `internal/situation/service.go` — situation/context surfaces that may expose packaged memory context.
- `internal/daemon/authored_context_runtime.go` — existing authored-context composition patterns to mirror where useful.
- `.compozy/tasks/mem-v2/analysis/analysis_ai-harness.md` — prompt packaging and compaction references.

### Dependent Files
- `internal/daemon/boot.go` — later daemon wiring task depends on the finished snapshot/assembly layer.
- `web/src/systems/session/components/session-inspector.tsx` — later session inspector work depends on stable packaged memory semantics.
- `.compozy/tasks/mem-v2/task_19.md` — daemon wiring depends on the completed assembly service.
- `.compozy/tasks/mem-v2/task_22.md` — session inspector task depends on final snapshot composition semantics.

### Related ADRs
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — constrains packaged recall behavior.
- [ADR-002: Three Scopes with Agent Two-Tier](adrs/adr-002.md) — constrains packaging precedence.
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — influences what is prompt-facing versus forensic only.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: packaged memory sections must remain consumable by the bundled provider and future provider implementations without daemon-specific forks.
- Agent manageability: no public control surface lands here, but later inspect/status surfaces must reflect these snapshot semantics faithfully.
- Config lifecycle: none — checked surfaces are refresh/snapshot-related settings and docs; public config work is deferred.

### Web/Docs Impact

- `web/`: none yet — checked surfaces are session inspector and knowledge pages; they update after daemon wiring and generated contracts land.
- `packages/site`: none — checked surfaces are runtime memory/session docs and references; they update after public behavior is exposed.

## Deliverables

- Frozen snapshot model and prompt-assembly service for Slice 1 memory context.
- Snapshot refresh invalidation behavior aligned to the TechSpec.
- Focused tests for packaging, caps, scope precedence, and sub-agent inheritance rules.

## Tests

- Unit tests:
  - [x] Frozen snapshot captures the approved memory state at session boot.
  - [x] Snapshot refresh semantics affect the next boot/refresh boundary rather than mutating active prompt state unexpectedly.
  - [x] Prompt packaging enforces size/freshness and precedence rules deterministically.
- Integration tests:
  - [x] Parent and sub-agent flows observe the approved snapshot inheritance and read-only behavior.
  - [x] Daemon-side assembly consumers can use the service without duplicating packaging logic.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/agent/memory_manager.py`
- `.resources/hermes/agent/context_engine.py`
- `.resources/codex/codex-rs/core/src/compact.rs`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 has one deterministic frozen-snapshot and prompt-assembly layer.
- Later daemon/UI work can consume packaged memory context without rebuilding recall logic.

## Completion Notes

- Added `SnapshotService`, `FrozenSnapshot`, `SnapshotBlock`, and next-boot-only invalidation semantics in `internal/memory`.
- Added provider-backed and store-backed snapshot capture with deterministic block order: global, workspace, agent-global, agent-workspace.
- Centralized recall prompt formatting through `RenderRecallPromptSection` and routed `NewRecallAugmenter` through that renderer.
- Added `Assembler.PromptStartupSection` so daemon composition can pass session/workspace/agent startup metadata without duplicating snapshot or packaging logic.
- Enforced sub-agent inheritance by cloning the parent snapshot, preserving the rendered section, setting `ControllerMode=read_only`, and avoiding child-private re-resolution.
- Added focused tests for frozen boot capture, reload boundary behavior, prompt caps/freshness, scope precedence, provider snapshot blocks, sub-agent inheritance, daemon consumers, race safety, and coverage.
