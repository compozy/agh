---
status: pending
title: Extension manifest parser (TOML and JSON)
type: backend
complexity: medium
dependencies: []
---

# Task 03: Extension manifest parser (TOML and JSON)

## Overview

Create the extension manifest parser in a new `internal/extension/` package. The parser reads `extension.toml` (primary) or `extension.json` (fallback), validates the schema, and produces a `Manifest` struct. Manifest-first discovery is the foundation of the extension loading pipeline — extensions can be listed and validated without executing any code.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/extension/` package with `manifest.go` containing the `Manifest` struct and related config types
- MUST support parsing `extension.toml` using `github.com/BurntSushi/toml` (AGH convention)
- MUST support parsing `extension.json` using `encoding/json` as fallback
- MUST implement a loader that tries `extension.toml` first, then `extension.json`, returning a typed error if neither exists
- MUST validate required fields: `name`, `version`, `min_agh_version`
- MUST validate semver format for `version` and `min_agh_version`
- MUST validate `min_agh_version` compatibility with current daemon version
- MUST parse `[resources]`, `[capabilities]`, `[actions]`, `[subprocess]`, `[security]` sections into typed structs
- MUST produce identical `Manifest` structs from equivalent TOML and JSON inputs
- MUST NOT execute any extension code during parsing
</requirements>

## Subtasks
- [ ] 3.1 Create `internal/extension/` package with `Manifest` struct and section configs
- [ ] 3.2 Implement TOML parser using `BurntSushi/toml`
- [ ] 3.3 Implement JSON parser using `encoding/json`
- [ ] 3.4 Implement dual-format loader with TOML-first precedence
- [ ] 3.5 Implement schema validation (required fields, semver, capability names)
- [ ] 3.6 Write table-driven tests for both formats and all validation paths

## Implementation Details

New package `internal/extension/` with `manifest.go` and `manifest_test.go`.

See TechSpec "Data Models" section for the `Manifest` struct and the TOML/JSON examples. See `_examples.md` for full manifest examples across multiple extension types.

Manifest struct mirrors the existing AGH config pattern (BurntSushi/toml with `toml:` tags, also JSON-compatible via `json:` tags on the same fields).

### Relevant Files
- `internal/config/config.go` — Existing TOML parsing pattern to follow for consistency
- `internal/config/hooks.go` — Existing hook declaration schema that extension manifests will reference
- `internal/skills/types.go` — `Skill` type for resource registration (extensions bundle skills)

### Dependent Files
- `internal/extension/capability.go` — Will consume `Manifest.Security.Capabilities` (task 04)
- `internal/extension/registry.go` — Will store `Manifest` fields in DB (task 05)
- `internal/extension/manager.go` — Will orchestrate manifest loading (task 06)

### Related ADRs
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Resources/capabilities/actions dimensions map to manifest sections

## Deliverables
- New `internal/extension/manifest.go` with `Manifest`, `ResourcesConfig`, `CapabilitiesConfig`, `ActionsConfig`, `SubprocessConfig`, `SecurityConfig` structs
- Dual-format loader function `LoadManifest(dir string) (*Manifest, error)`
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Parse valid `extension.toml` produces correct Manifest struct
  - [ ] Parse valid `extension.json` produces identical Manifest to equivalent TOML
  - [ ] Missing `name` field returns validation error
  - [ ] Missing `version` field returns validation error
  - [ ] Invalid semver `version` returns validation error
  - [ ] `min_agh_version` newer than current daemon returns compatibility error
  - [ ] Loader returns TOML manifest when both TOML and JSON exist
  - [ ] Loader returns JSON manifest when only JSON exists
  - [ ] Loader returns typed `ErrManifestNotFound` when neither exists
  - [ ] Parse extension with resources (skills, agents, hooks, mcp_servers) sections
  - [ ] Parse extension with capabilities.provides and actions.requires
  - [ ] Parse extension with subprocess env var substitution placeholders
  - [ ] Unknown top-level sections are accepted for forward compatibility (ignored)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/extension/manifest.go` exists and compiles
- TOML and JSON loaders produce identical structs from equivalent inputs
- `make verify` passes
