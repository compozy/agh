---
status: pending
title: Extension registry (SQLite)
type: backend
complexity: medium
dependencies:
  - task_03
  - task_04
---

# Task 05: Extension registry (SQLite)

## Overview

Create the extension registry backed by SQLite in the existing global database (`~/.agh/agh.db`). The registry persists extension installation state, version, source, enabled/disabled state, declared capabilities, and SHA-256 checksums for artifact verification. This is the durable source of truth for what extensions are installed on a daemon.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/extension/registry.go` with `Registry` struct operating on `*sql.DB`
- MUST add new `extensions` table to `internal/store/globaldb/global_db.go` schema
- MUST define table columns per TechSpec Extension Registry table: `name` (PK), `version`, `source`, `enabled`, `manifest_path`, `installed_at`, `capabilities` (JSON), `actions` (JSON), `checksum`
- MUST implement CRUD operations: `Install(manifest, path, checksum) error`, `Uninstall(name) error`, `Enable(name) error`, `Disable(name) error`, `List() ([]ExtensionInfo, error)`, `Get(name) (*ExtensionInfo, error)`
- MUST use `IF NOT EXISTS` schema migration pattern (matching existing globaldb pattern)
- MUST verify checksum on Install against provided manifest artifact
- MUST serialize capabilities and actions as JSON in DB and deserialize back to typed structs
- MUST return typed `ErrExtensionNotFound` when extension doesn't exist
- MUST prevent duplicate installations by name (return `ErrExtensionExists`)
</requirements>

## Subtasks
- [ ] 5.1 Add `extensions` table to `globalSchemaStatements` in `internal/store/globaldb/global_db.go`
- [ ] 5.2 Create `internal/extension/registry.go` with `Registry` struct and `ExtensionInfo` type
- [ ] 5.3 Implement CRUD operations with parameterized SQL queries
- [ ] 5.4 Implement SHA-256 checksum verification on install
- [ ] 5.5 Write unit and integration tests using `t.TempDir()` for isolated SQLite instances

## Implementation Details

Add schema statement to `internal/store/globaldb/global_db.go`. Create `internal/extension/registry.go` and `internal/extension/registry_test.go`.

See TechSpec "Data Models" section for the Extension Registry table schema.

Follow the existing pattern in `internal/store/globaldb/global_db.go` for schema declaration and table access. Use parameterized queries exclusively (no string concatenation).

### Relevant Files
- `internal/store/globaldb/global_db.go` — Existing schema pattern and connection management
- `internal/extension/manifest.go` — `Manifest` struct provides values to persist (task 03)
- `internal/extension/capability.go` — `ExtensionSource` enum used in table column (task 04)
- `internal/skills/provenance.go` — Existing checksum verification pattern to follow

### Dependent Files
- `internal/extension/manager.go` — Will use Registry to list and load enabled extensions at boot (task 06)
- `internal/cli/extension.go` — Will use Registry for list/install/enable/disable commands (task 09)

### Related ADRs
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Registry persists the extension identity

## Deliverables
- Extended `internal/store/globaldb/global_db.go` with `extensions` table
- New `internal/extension/registry.go` with `Registry` struct, `ExtensionInfo`, CRUD methods
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for full install → enable → disable → uninstall lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `Install()` persists extension to DB with correct fields
  - [ ] `Install()` rejects duplicate name with `ErrExtensionExists`
  - [ ] `Install()` with wrong checksum returns verification error
  - [ ] `Get()` returns `ExtensionInfo` for existing extension
  - [ ] `Get()` returns `ErrExtensionNotFound` for missing extension
  - [ ] `List()` returns all installed extensions
  - [ ] `List()` returns empty slice when none installed (not nil)
  - [ ] `Enable()` sets `enabled=true` in DB
  - [ ] `Disable()` sets `enabled=false` in DB
  - [ ] `Uninstall()` removes extension from DB
  - [ ] `Uninstall()` on missing extension returns `ErrExtensionNotFound`
  - [ ] Capabilities JSON round-trip preserves all fields
  - [ ] Actions JSON round-trip preserves all fields
  - [ ] Schema migration runs idempotently (IF NOT EXISTS)
- Integration tests:
  - [ ] Full lifecycle: install → list → enable → disable → uninstall
  - [ ] Two extensions with different sources coexist in same DB
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `extensions` table created on first daemon boot after upgrade
- `make verify` passes
