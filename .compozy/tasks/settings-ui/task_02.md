---
status: pending
title: Settings service orchestration in internal/settings
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Settings service orchestration in internal/settings

## Overview

Introduce `internal/settings` as the daemon-facing orchestration layer that assembles section read models and applies validated mutations on top of the new persistence primitives. This package becomes the single owner of section semantics, scope rules, precedence metadata, and the runtime-apply matrix defined in the TechSpec.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", and "Runtime apply matrix"
- FOCUS ON "WHAT" — implement one settings service boundary, not transport-specific handlers
- MINIMIZE CODE — keep file-level organization inside `internal/settings`; do not turn it into a grab bag for low-level file edits
- TESTS REQUIRED — section assembly, mutation classification, and source-precedence logic must be covered
- GREENFIELD: Não aceitar ambiguidade silenciosa em scope, target ou precedence; preferir erro explícito
</critical>

<requirements>
- MUST add `internal/settings` with the service interface and request/response orchestration described in the TechSpec
- MUST assemble section envelopes for `general`, `memory`, `skills`, `automation`, `network`, `observability`, and `hooks-extensions`
- MUST implement collection orchestration for `providers`, `mcp-servers`, `environments`, and `hooks`
- MUST enforce v1 scope rules, including workspace-scoped editing only for `mcp-servers`
- MUST return semantic source metadata such as `effective_source`, `shadowed_sources`, `available_targets`, and `write_target`
- MUST classify every mutation according to the v1 runtime-apply matrix and return `MutationResult`
</requirements>

## Subtasks

- [ ] 2.1 Create `internal/settings` package structure with service, models, and request/response types
- [ ] 2.2 Implement section read-model assembly from config, runtime services, and workspace resolution
- [ ] 2.3 Implement collection list/put/delete orchestration with explicit scope and target semantics
- [ ] 2.4 Enforce v1 scope rules and source-precedence reporting, especially for `mcp-servers`
- [ ] 2.5 Encode the runtime-apply matrix and mutation result classification
- [ ] 2.6 Cover section assembly, mutation classification, and precedence behavior with tests

## Implementation Details

See TechSpec sections "Core Interfaces", "Data Models", "Runtime apply matrix", and "Development Sequencing". This task should consume `internal/config` write primitives from task_01 and runtime dependencies from the daemon composition root later, but it should not know about Gin, Cobra, or HTTP status codes.

### Relevant Files

- `internal/api/core/interfaces.go` — future consumer of the settings service contract
- `internal/config/config.go` — source of the canonical config model the read models are built from
- `internal/config/merge.go` — precedence behavior that `internal/settings` must surface to clients
- `internal/daemon/extensions.go` — likely runtime source for extensions-related read models
- `internal/daemon/tool_mcp_resources.go` — existing MCP-related runtime data that may inform section envelopes

### Dependent Files

- `internal/api/contract/` — will mirror these service shapes in task_04
- `internal/api/core/handlers.go` — will route transport calls into this service in task_05
- `internal/daemon/daemon.go` — will wire service dependencies from the composition root
- `internal/settings/*_test.go` — should own package-level unit coverage for read models and mutations

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the section-oriented settings model
- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — Defines canonical write targets and persistence behavior
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines mutation classification and action semantics

## Deliverables

- New `internal/settings` package with service interface, envelopes, request models, and mutation orchestration
- Scope, target, and source-precedence handling for section and collection resources
- Runtime-apply matrix enforcement and `MutationResult` generation **(REQUIRED)**
- Unit tests with >=80% coverage for `internal/settings` **(REQUIRED)**
- Integration-style tests with fake runtime dependencies and temp config state **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `GetSection` returns the correct config and runtime envelope for each supported section
  - [ ] Invalid scope combinations return a descriptive validation error
  - [ ] `mcp-servers` collection responses include `effective_source`, `shadowed_sources`, and `available_targets`
  - [ ] `target=auto` chooses the highest-precedence existing source and defaults new MCP servers to sidecar writes
  - [ ] Runtime-apply classification returns `applied_now`, `restart_required`, or `action_trigger` per the matrix
- Integration tests:
  - [ ] Provider overlay delete reveals builtin fallback metadata correctly
  - [ ] Workspace-scoped MCP mutation resolves the workspace root and persists to the intended target
  - [ ] Settings mutation results expose semantic `write_target` instead of filesystem paths
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for `internal/settings`
- A single daemon-facing service owns settings semantics without leaking transport or UI concerns
- Multi-source resources expose deterministic precedence metadata that the UI can render without guessing
