---
status: completed
title: "Implement Provenance hash verification and sidecar"
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 6: Implement Provenance hash verification and sidecar

## Overview

Implement the `.agh-meta.json` sidecar system for marketplace skill provenance. This includes SHA-256 hash computation, sidecar file read/write, and tamper detection on load. Skills installed from marketplaces store their hash at install time; on every load, the hash is recomputed and compared.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/provenance.go` with hash and sidecar functions
- MUST compute SHA-256 hash of SKILL.md content
- MUST write `.agh-meta.json` sidecar containing Provenance struct (Hash, Registry, Slug, Version, InstalledAt)
- MUST read `.agh-meta.json` and unmarshal into Provenance struct
- MUST detect hash mismatch between stored and recomputed hash
- MUST provide function to check if a skill directory has a sidecar (marketplace detection)
- MUST use `encoding/json` for sidecar serialization with proper error wrapping
</requirements>

## Subtasks
- [x] 6.1 Implement SHA-256 hash computation for SKILL.md content
- [x] 6.2 Implement sidecar write (Provenance → .agh-meta.json)
- [x] 6.3 Implement sidecar read (.agh-meta.json → Provenance)
- [x] 6.4 Implement hash verification (recompute + compare)
- [x] 6.5 Implement HasSidecar() detection function
- [x] 6.6 Write unit tests for all provenance operations

## Implementation Details

New file `internal/skills/provenance.go`. The sidecar file lives alongside SKILL.md in the skill directory. The registry (task_07) calls these functions during skill loading.

See TechSpec "Provenance" and "Data Flow — Marketplace" sections.

### Relevant Files
- `internal/skills/types.go` — Provenance struct (from task_01)

### Dependent Files
- `internal/skills/registry.go` — calls provenance functions during loadGlobalSkills (task_07)
- `internal/skills/marketplace/clawhub/client.go` — writes sidecar after install (task_08)

### Related ADRs
- [ADR-004: Hash-Based Provenance](adrs/adr-004.md) — defines hash verification approach and quarantine semantics

## Deliverables
- `internal/skills/provenance.go` with hash, sidecar read/write, verification functions
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] ComputeHash returns consistent SHA-256 for same content
  - [x] ComputeHash returns different hash for different content
  - [x] WriteSidecar creates .agh-meta.json with correct JSON structure
  - [x] ReadSidecar parses valid .agh-meta.json into Provenance
  - [x] ReadSidecar returns descriptive error for malformed JSON
  - [x] ReadSidecar returns fs.ErrNotExist for missing sidecar
  - [x] VerifyHash returns nil when hash matches
  - [x] VerifyHash returns error with expected vs actual hash when tampered
  - [x] HasSidecar returns true when .agh-meta.json exists
  - [x] HasSidecar returns false when .agh-meta.json missing
  - [x] Round-trip: write then read produces identical Provenance
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Sidecar JSON is human-readable and stable across writes
