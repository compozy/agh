---
status: completed
title: Daemon Wiring and Boundary Registration
type: backend
complexity: critical
dependencies:
  - task_08
  - task_09
  - task_10
  - task_11
  - task_12
  - task_13
  - task_17
  - task_18
---

# Task 19: Daemon Wiring and Boundary Registration

## Overview

Wire the completed Memory v2 subsystems together at the composition root and register any new package boundaries required by the implementation. This task is where the individual slices become the actual running daemon behavior: controller, recall, provider, extractor, dreaming, ledger materialization, settings/config, and agent-manageability surfaces all come together.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `System Architecture`, `Architectural Boundaries`, and `Development Sequencing` steps 29-30.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, and `agh-cleanup-failure-paths` before editing production Go.
- MINIMIZE CODE churn outside composition-root and boundary-registration seams.
- TESTS REQUIRED: daemon boot wiring, runtime smoke flows, new package-boundary enforcement, and shutdown cleanup must ship here.
- NO WORKAROUNDS: all cross-package wiring belongs in `internal/daemon`; do not sneak composition into subordinate packages.
</critical>

<requirements>
- MUST wire controller, recall, provider registry, prompt assembly, observability, extractor, dreaming, and ledger materializer from the daemon composition root.
- MUST register any new internal package boundaries in the repo boundary checks.
- MUST keep subordinate packages free of composition-root back-pointers and hidden wiring.
- MUST ensure daemon startup/shutdown order is safe for the new memory runtime, including queue workers and background tasks.
- MUST preserve the no-feature-flag Slice 1 rollout semantics from the TechSpec.
</requirements>

## Subtasks
- [x] 19.1 Wire the new Memory v2 services at `internal/daemon` boot and runtime entry points.
- [x] 19.2 Register package-boundary rules for new memory/session/store packages.
- [x] 19.3 Add daemon-level tests for startup, shutdown, and cross-slice runtime behavior.
- [x] 19.4 Confirm no subordinate package now imports or wires daemon-level collaborators directly.

## Implementation Details

See TechSpec `System Architecture`, `Architectural Boundaries`, and `Development Sequencing` steps 29-30. This task must keep `internal/daemon` as the sole composition root and update the boundary graph in the same change when new packages appear.

### Relevant Files
- `internal/daemon/boot.go` — primary composition-root wiring for memory services.
- `internal/daemon/composed_assembler.go` — prompt assembly integration point.
- `internal/daemon/prompt_sections.go` — prompt section registration/ordering.
- `internal/daemon/native_tools.go` — daemon-owned native-tool/provider wiring that must match the new runtime.
- `internal/daemon/boundary.go` — runtime boundary verification behavior.
- `magefile.go` — CI-enforced import-boundary registration.

### Dependent Files
- `web/src/routes/_app/knowledge.tsx` — later web task depends on the daemon serving final Memory v2 behavior.
- `web/src/routes/_app/settings/memory.tsx` — later settings UI task depends on the daemon wiring.
- `packages/site/content/runtime/**` — later docs tasks depend on a truthful running daemon surface.
- `.compozy/tasks/mem-v2/task_20.md` — web knowledge task depends on the final runtime being wired.
- `.compozy/tasks/mem-v2/task_24.md` — discoverability/reference task depends on final runtime truth.

### Related ADRs
- [ADR-012: Slice 1 Fat Scope — Single TechSpec with Four Eixos](adrs/adr-012.md) — explains the atomic rollout.
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — provider wiring implications.
- [ADR-010: Fact Extraction Location — Hybrid Per-Turn Hook + Optional Compaction Flush](adrs/adr-010.md) — extractor wiring implications.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task is where the provider registry, host API, and daemon runtime finally become one live extensibility surface.
- Agent manageability: all CLI/HTTP/UDS/native-tool surfaces depend on this runtime wiring being truthful and complete.
- Config lifecycle: this task consumes the backend config/settings truth but does not add new keys; it must honor the final defaults/validation rules from `task_13`.

### Web/Docs Impact

- `web/`: downstream knowledge, settings, and session-inspector pages depend on the runtime served by this task, but no web code changes land here.
- `packages/site`: runtime/docs/reference truth depends on this wired daemon behavior, but docs changes are deferred.

## Deliverables

- Memory v2 services wired at the daemon composition root.
- Updated import-boundary registration for any new packages.
- Daemon-level tests for startup, runtime behavior, and clean shutdown.

## Tests

- Unit tests:
  - [x] Composition-root setup builds the final Memory v2 graph without hidden back-pointers.
  - [x] Boundary registration includes any new packages introduced by the Memory v2 implementation.
- Integration tests:
  - [x] Daemon startup serves the final memory runtime, including provider, extractor, dreaming, and ledger components.
  - [x] Daemon shutdown joins memory-owned background workers cleanly without leaks.
  - [x] `mage Boundaries` and relevant daemon integration tests pass with the new package graph.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/run_agent.py`
- `.resources/hermes/agent/memory_manager.py`
- `.resources/codex/codex-rs/memories/write/src/runtime.rs`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 runs end-to-end in the daemon with composition-root discipline preserved.
- Boundary checks and shutdown behavior remain clean after the new wiring lands.
