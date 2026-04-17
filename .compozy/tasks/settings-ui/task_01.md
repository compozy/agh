---
status: pending
title: Comment-preserving config editors and write targets
type: backend
complexity: high
dependencies: []
---

# Task 01: Comment-preserving config editors and write targets

## Overview

Replace the bootstrap-only persistence assumptions in `internal/config` with write primitives that are safe for operator-facing settings mutations. This task creates the lowest-level canonical write path for TOML overlays and MCP sidecars, and every later settings mutation depends on it being precise and non-destructive.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "API Endpoints", and "Technical Dependencies" for behavior
- FOCUS ON "WHAT" — describe the intended outcome and constraints, not local implementation minutiae
- MINIMIZE CODE — prefer targeted persistence helpers over broad rewrites or new stores
- TESTS REQUIRED — every mutation path in this task must be protected by unit and integration tests
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e rejeitar mutation ambígua em vez de canonizar arquivo do usuário
</critical>

<requirements>
- MUST replace blind whole-file TOML re-encoding for settings mutations with comment-preserving document edits
- MUST support global config writes, workspace overlay writes, and semantic write-target selection without leaking absolute paths to higher layers
- MUST add MCP sidecar write support that preserves unknown top-level keys and untouched server entries
- MUST validate the merged effective config before committing any overlay or sidecar change
- MUST reject unsupported mutations when the requested TOML edit would rewrite unrelated document structure
- SHOULD keep persistence primitives transport-agnostic so `internal/settings` can orchestrate them later without duplicating file logic
</requirements>

## Design References

This task is foundational — it does not render any individual Paper screen, but every later settings mutation flows through these persistence primitives. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [ ] 1.1 Add comment-preserving TOML edit primitives for global and workspace overlay files
- [ ] 1.2 Add explicit write-target resolution for config and sidecar destinations
- [ ] 1.3 Implement MCP JSON writer behavior that preserves unknown top-level keys and untouched definitions
- [ ] 1.4 Validate merged effective config before committing any write
- [ ] 1.5 Reject unsupported mutations when safe structured edits are not possible
- [ ] 1.6 Cover preservation, validation, and target-selection behavior with tests

## Implementation Details

See TechSpec sections "Data Models", "Collection mutation semantics", "Technical Dependencies", and ADR-002. This task should stay inside `internal/config` and expose reusable persistence helpers instead of embedding settings-specific orchestration.

### Relevant Files

- `internal/config/bootstrap.go` — current overlay writing path that cannot be reused as-is for settings mutations
- `internal/config/mcpjson.go` — existing MCP sidecar reading logic and the natural place to add safe writes
- `internal/config/merge.go` — merged effective config validation and overlay precedence
- `internal/config/config.go` — canonical config model that the editor must preserve semantically
- `internal/config/home.go` — source of global config and home-relative sidecar destinations

### Dependent Files

- `internal/settings/service.go` — will call these persistence primitives in task_02
- `internal/settings/targets.go` — will map settings resources onto these write targets in task_02
- `internal/config/bootstrap_test.go` — should expand to verify comment preservation and unsupported-edit rejection
- `internal/config/mcpjson_test.go` — should cover new sidecar writer semantics

### Related ADRs

- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — Canonical persistence model, comment preservation, and MCP writer requirements

## Deliverables

- Comment-preserving overlay edit primitives in `internal/config` for global and workspace writes
- MCP sidecar writer support with explicit target handling and preservation semantics
- Merged-config validation hooks that guard all writes **(REQUIRED)**
- Unit tests with >=80% coverage for new persistence helpers **(REQUIRED)**
- Integration tests covering real overlay/sidecar writes via temp directories **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Editing one TOML field preserves unrelated comments and untouched sections
  - [ ] Unsupported TOML mutation returns a descriptive error instead of reserializing the full file
  - [ ] MCP writer preserves unknown top-level keys and untouched server definitions
  - [ ] Write-target resolution chooses the correct config or sidecar destination for global and workspace scope
  - [ ] Merged-config validation blocks invalid writes before commit
- Integration tests:
  - [ ] Global settings write updates `config.toml` on disk while preserving unrelated TOML structure
  - [ ] Workspace-scoped write updates `<workspace>/.agh/config.toml` without altering global config
  - [ ] MCP sidecar write updates `mcp.json` on disk and preserves unaffected entries
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the new or modified `internal/config` persistence surface
- Later settings layers can perform canonical writes without re-encoding whole TOML documents
- Unsupported edits fail explicitly instead of mutating user-managed formatting or comments
