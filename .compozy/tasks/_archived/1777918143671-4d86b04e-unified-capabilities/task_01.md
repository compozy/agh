---
status: completed
title: Unified Capability Schema, Canonicalization, and Session Projection
type: backend
complexity: high
dependencies: []
---

# Task 01: Unified Capability Schema, Canonicalization, and Session Projection

## Overview

Unify the local capability model so `internal/config` and `internal/session` become the single source of truth for the new structured capability artifact. This task establishes the authored schema extensions, canonical digest generation, and session/runtime projection that every later network, API, and documentation task depends on.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", "Testing Approach", and "Development Sequencing"
- KEEP LOCAL AUTHORING AND CANONICALIZATION INSIDE `internal/config` AND `internal/session` - do not leak agent-file parsing into `internal/network`
- FOCUS ON THE UNIFIED `capability` MODEL - this task must not preserve `recipe` semantics through parallel data types
- TESTS REQUIRED - this task must land with deterministic unit and integration coverage, not only helper-level smoke checks
- GREENFIELD: delete obsolete branches of the old split instead of adding compat aliases or dual paths
</critical>

<requirements>
- MUST extend the authored capability schema with `version` and `requirements` while keeping existing local storage layouts unchanged
- MUST compute `digest` from one canonical capability representation owned by the runtime, not from authored user input
- MUST normalize and validate `requirements` entries, duplicate IDs, and any new schema fields with hard errors in `internal/config`
- MUST project the unified capability data from config-owned structs into session-owned runtime structs without sharing mutable references
- MUST keep missing capability catalogs optional and deterministic, matching the existing capability loading contract
- SHOULD define digest behavior so equivalent normalized capability documents produce the same digest across TOML and JSON sources
</requirements>

## Subtasks
- [x] 1.1 Extend the config-owned capability schema and validation rules for `version`, `requirements`, and canonical digest inputs
- [x] 1.2 Implement canonicalization and runtime-computed digest generation for unified capabilities
- [x] 1.3 Update agent loading so the unified catalog remains optional but normalized consistently across supported layouts
- [x] 1.4 Extend session/runtime projection structs so downstream discovery and transfer use the same normalized capability content
- [x] 1.5 Add focused unit and integration coverage for schema, digest stability, and projection invariants

## Implementation Details

See TechSpec "Core Interfaces", "Data Models", and "Build Order" items 1-2. The key architectural constraint is that authored capability files remain local-first and field-based, while `digest` becomes a derived runtime property that downstream network code can trust without re-reading files.

### Relevant Files
- `internal/config/capabilities.go` - primary schema, normalization, and validation entrypoint for authored capability catalogs
- `internal/config/capabilities_test.go` - existing schema tests to extend for `version`, `requirements`, and digest-related invariants
- `internal/config/agent.go` - agent-directory loading path that attaches optional capability catalogs
- `internal/config/agent_capabilities_test.go` - integration coverage for real agent-directory capability fixtures
- `internal/session/network_peer.go` - session-owned runtime projection boundary for network-visible capability data
- `internal/session/manager_lifecycle.go` - session activation path where projected peer data becomes live runtime state

### Dependent Files
- `internal/network/capability_brief.go` - brief discovery projection will consume the normalized runtime capability shape from this task
- `internal/network/capability_catalog.go` - rich discovery and filtering will rely on the extended schema and digest values introduced here
- `internal/network/envelope.go` - transferable `kind:"capability"` payloads will depend on the canonical fields defined here
- `internal/api/contract/contract.go` - API-visible peer and discovery contracts will need the unified runtime shape
- `internal/api/core/network.go` - API handlers will later expose the projected unified capability data

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - establishes capability as the only surviving authored and network artifact
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - governs local layouts, structured fields, runtime digesting, and `requirements`

## Deliverables
- Extended unified capability schema and validation in `internal/config`
- Runtime-owned canonicalization and digest generation for capabilities
- Session projection updates carrying all fields required by discovery and transfer
- Updated loader and projection tests covering valid, invalid, and optional-catalog paths **(REQUIRED)**
- Integration coverage proving TOML and JSON sources normalize to the same runtime shape where semantically equivalent **(REQUIRED)**
- Test coverage >=80% for the touched backend packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Loading a capability catalog with `version` and `requirements` preserves those fields in normalized runtime form
  - [x] Equivalent TOML and JSON capability catalogs produce identical canonical digests after normalization
  - [x] Changing a meaningful structured field such as `execution_outline` or `requirements` changes the computed digest
  - [x] Duplicate `requirements` entries are normalized or rejected according to the final schema rule
  - [x] Missing or malformed `version` and `requirements` fields return descriptive validation errors
- Integration tests:
  - [x] Real agent-directory fixtures load unified capability catalogs through `internal/config` without breaking optional-catalog behavior
  - [x] Session projection clones config-owned capability data so later network mutations cannot affect the source catalog
  - [x] Existing single-file and directory capability layouts remain supported without introducing dual parsing paths
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Unified capabilities have one canonical runtime representation with deterministic digesting
- Session-owned peer capability data is ready for downstream discovery and transfer tasks without reparsing local files
