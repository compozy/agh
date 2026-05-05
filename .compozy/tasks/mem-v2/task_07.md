---
status: pending
title: Local Provider and Registry Surface
type: backend
complexity: high
dependencies:
  - task_01
  - task_03
  - task_05
  - task_06
---

# Task 07: Local Provider and Registry Surface

## Overview

Implement the bundled local `MemoryProvider` and the registry surface that will own provider selection and collision handling. This task makes the new provider ABC concrete without yet introducing external providers, so later config, daemon wiring, and public manageability surfaces can depend on one real implementation.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `MemoryProvider ABC`, `Extensibility Integration Plan`, and `Development Sequencing` steps 13-14.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside provider/runtime boundaries and extension registry seams.
- TESTS REQUIRED: provider lifecycle, collision rejection, local-provider read/write/recall integration, and contract-only import safety must ship here.
- NO WORKAROUNDS: provider implementations import `internal/memory/contract` only, never controller/recall internals directly.
</critical>

<requirements>
- MUST implement the bundled local provider against the new contract types only.
- MUST add provider registry behavior that rejects collisions and emits the expected observability signal.
- MUST keep provider initialization/read/write/recall hooks compatible with the controller and recall subsystems already built.
- MUST avoid introducing external-provider compatibility code in Slice 1.
- MUST provide clean seams for later config, daemon, native-tool, and host API tasks.
</requirements>

## Subtasks
- [ ] 7.1 Implement the bundled local provider on top of the new controller/recall/store substrate.
- [ ] 7.2 Add provider registry behavior for registration, collision handling, and selection.
- [ ] 7.3 Add focused provider tests for hook lifecycle and import-boundary discipline.
- [ ] 7.4 Confirm future external providers can target this surface without runtime-private imports.

## Implementation Details

See TechSpec `MemoryProvider ABC`, `Extensibility Integration Plan`, and `Development Sequencing` steps 13-14. This task should deliver the local provider plus registry mechanics, but not yet the full daemon/public exposure of provider controls.

### Relevant Files
- `internal/extension/registry.go` — registry patterns and collision handling to extend for memory providers.
- `internal/extension/host_api.go` — host API surfaces that will later expose memory-provider behavior.
- `internal/extension/host_api_test.go` — existing host API coverage to extend with provider-safe wiring.
- `internal/daemon/native_tools.go` — later provider-aware memory surfaces will depend on registry state.
- `internal/memory/*` — source location for the new local provider implementation.
- `.compozy/tasks/mem-v2/analysis/analysis_hermes.md` — provider lifecycle precedent.

### Dependent Files
- `internal/config/config.go` — later config task will expose provider selection and provider-level memory settings.
- `internal/api/contract/settings.go` — later settings/public contract task depends on provider metadata.
- `internal/daemon/boot.go` — later daemon wiring task depends on the local provider and registry.
- `.compozy/tasks/mem-v2/task_13.md` — config/settings backend depends on provider metadata.
- `.compozy/tasks/mem-v2/task_18.md` — native-tool and extension-host surfaces depend on final registry behavior.

### Related ADRs
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — normative provider lifecycle.
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — constrains what the local provider owns.
- [ADR-012: Slice 1 Fat Scope — Single TechSpec with Four Eixos](adrs/adr-012.md) — explains why the local provider ships in Slice 1.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task defines the actual provider seam and registration semantics that later extension host and provider selection surfaces will expose.
- Agent manageability: no public CLI/HTTP/UDS/native-tool changes land here yet, but later provider-facing verbs must reflect this registry behavior.
- Config lifecycle: none yet — checked surfaces are provider selection keys, settings payloads, docs, and validation; they land in `task_13`.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types and settings UI; public provider settings contract lands later.
- `packages/site`: none — checked surfaces are provider/memory runtime docs and references; docs update after public config/verbs stabilize.

## Deliverables

- Bundled local MemoryProvider implementation using contract-only imports.
- Provider registry behavior with collision handling and focused coverage.
- Local provider lifecycle tests across initialize/read/write/recall-style hooks.

## Tests

- Unit tests:
  - [ ] Local provider uses contract-only types and does not import runtime-private packages.
  - [ ] Registry rejects provider collisions deterministically and surfaces the correct error path.
  - [ ] Local provider delegates to controller/recall/store seams correctly for Slice 1 behavior.
- Integration tests:
  - [ ] The bundled provider initializes and serves memory operations inside a daemon-like runtime harness.
  - [ ] Collision and fallback behaviors emit the expected observability signals and do not corrupt active provider selection.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/agent/memory_provider.py`
- `.resources/hermes/plugins/memory/__init__.py`
- `.resources/hermes/plugins/memory/honcho/__init__.py`
- `.resources/hermes/tests/agent/test_memory_provider.py`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 has a real bundled provider plus registry semantics ready for config and daemon wiring.
- Provider authors have one stable contract-only integration surface.

