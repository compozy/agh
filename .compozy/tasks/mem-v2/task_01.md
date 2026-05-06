---
status: completed
title: Memory Contract Extraction and Hard Cut
type: backend
complexity: critical
dependencies: []
---

# Task 01: Memory Contract Extraction and Hard Cut

## Overview

Create the new `internal/memory/contract` package as the bottom dependency layer for Memory v2 and remove the ambiguous shared-type surface from `internal/memory/types.go`. This task establishes the import boundary that every later slice depends on: controller, recall, extractor, provider, API adapters, CLI, and extension host code must all speak the same contract types without compatibility bridges.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Public Interfaces / Types`, `Architectural Boundaries`, and `Development Sequencing` instead of duplicating schemas here.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside the files and transports listed below; no re-export shim, alias, or temporary compatibility layer is allowed.
- TESTS REQUIRED: unit and compile-boundary coverage must ship in the same task with >=80% coverage where the package threshold applies.
- NO WORKAROUNDS: `internal/memory/contract` must remain the lowest dependency in the memory subtree.
</critical>

<requirements>
- MUST create `internal/memory/contract` for memory enums, scope/agent-tier metadata, provenance, query/write DTOs, replay records, recall records, and provider-facing interfaces.
- MUST move callers off `internal/memory/types.go` and delete the file as a hard cut in the same task.
- MUST keep the new contract package free from imports on controller, recall, extractor, provider implementations, or daemon wiring.
- MUST preserve the Slice 1 lexical-only scope by excluding embedding/vector fields from the contract.
- MUST add compile-time assertions or focused tests proving the new package does not reintroduce cycles or speculative runtime seams.
</requirements>

## Subtasks
- [x] 1.1 Create `internal/memory/contract` and move all shared Memory v2 DTOs and enums into it.
- [x] 1.2 Update existing memory, API, CLI, tool, and extension callers to import the new contract surface directly.
- [x] 1.3 Delete `internal/memory/types.go` and any tests that exist only to preserve speculative shapes.
- [x] 1.4 Add focused tests for serialization, enum normalization, and import-boundary safety.
- [x] 1.5 Confirm no compatibility alias or re-export file remains in the old package path.

## Implementation Details

See TechSpec `Architectural Boundaries`, `Public Interfaces / Types`, and `Development Sequencing` steps 2-3. This task is intentionally structural: it should finish with a stable contract package that later tasks can depend on without reopening package topology.

### Relevant Files
- `internal/memory/types.go` — current mixed contract/runtime/speculative surface to remove.
- `internal/memory/interfaces_test.go` — existing seam tests that guard future memory interfaces.
- `internal/api/core/memory.go` — shared handler code that currently imports memory-domain types directly.
- `internal/cli/memory.go` — CLI output models currently tied to the old memory package types.
- `internal/extension/host_api.go` — extension-facing memory surface that must consume contract types, not runtime internals.
- `internal/tools/builtin/memory.go` — builtin memory tool descriptors that should align with contract-level nouns.

### Dependent Files
- `internal/memory/store.go` — later tasks depend on the new contract for replay and storage boundaries.
- `internal/memory/recall.go` — later recall/ranking work must import the contract package.
- `internal/memory/dream.go` — dreaming work will consume contract candidates and promotion DTOs.
- `.compozy/tasks/mem-v2/task_03.md` — store/replay task depends on this package split.
- `.compozy/tasks/mem-v2/task_14.md` — public API contract work depends on stable domain DTOs.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — defines the core memory taxonomy and authority split.
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — requires a stable import surface for provider authors.
- [ADR-012: Slice 1 Fat Scope — Single TechSpec with Four Eixos](adrs/adr-012.md) — explains why this hard cut must happen early.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task creates the stable import surface that provider implementations, extension host adapters, and future bridge SDK helpers must use.
- Agent manageability: no public verb changes land here, but all transport adapters must stop depending on runtime-private memory types.
- Config lifecycle: none — checked surfaces are config structs, defaults, tool-surface paths, settings routes, and site docs; they remain unchanged in this task.

### Web/Docs Impact

- `web/`: none — checked surfaces are `web/src/generated/**`, `web/src/systems/knowledge/**`, and `web/src/systems/settings/**`; generated/web changes are deferred until `task_15`, `task_20`, and `task_21`.
- `packages/site`: none — checked surfaces are runtime docs and generated references; docs changes are deferred until `task_23` and `task_24`.

## Deliverables

- `internal/memory/contract` created as the canonical shared Memory v2 import surface.
- All direct callers updated to consume the new contract package.
- `internal/memory/types.go` deleted with no shim or alias left behind.
- Unit/compile-boundary tests covering enum normalization, JSON/DB-facing DTO shapes, and import safety.

## Tests

- Unit tests:
  - [x] Contract enums normalize to the Slice 1 canonical values for scope, agent tier, origin, memory type, and write operations.
  - [x] Contract DTOs round-trip through JSON or helper serialization without introducing deprecated fields.
  - [x] Package-level tests prove no cycle exists between `contract` and controller/recall/provider packages.
- Integration tests:
  - [x] `go test ./internal/memory/... ./internal/api/... ./internal/cli/... ./internal/extension/...` passes after the hard cut.
  - [x] Existing memory-related transport packages compile and run against the new contract surface without re-export shims.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/agent/memory_provider.py`
- `.resources/hermes/tools/memory_tool.py`
- `.resources/codex/codex-rs/memories/write/src/lib.rs`
- `.resources/codex/codex-rs/memories/write/src/control.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- `internal/memory/contract` is the only shared Memory v2 DTO surface.
- `internal/memory/types.go` is gone and no compatibility layer remains.
