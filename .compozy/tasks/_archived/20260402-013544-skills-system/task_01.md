---
status: completed
title: Skills Package Core
type: ""
complexity: medium
dependencies: []
---

# Task 1: Skills Package Core

## Overview

Create the `internal/skills/` package with foundational types, SKILL.md parser, security verification, and eligibility filtering. This is the base layer that all other skills tasks depend on — it defines the data structures, parsing logic, and filtering rules that the registry, kernel, and CLI will use.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define all skill types per AgentSkills spec (SkillMeta, Skill, SkillSource, SkillSnapshot, LoadConfig, SnapshotFilter, Warning)
- MUST parse SKILL.md files with YAML frontmatter extraction and Markdown body separation
- MUST implement lenient YAML parsing (warn on issues, skip only if description missing or YAML unparseable)
- MUST detect prompt injection patterns in skill content with Critical/Warning/Info severity levels
- MUST filter skills by OS, disabled list, and optional allowlist
- MUST add YAML dependency via `go get` (prefer `gopkg.in/yaml.v3` or use existing `github.com/goccy/go-yaml`)
</requirements>

## Subtasks
- [x] 1.1 Create `internal/skills/types.go` with all type definitions (SkillMeta, Skill, SkillSource, SkillSnapshot, LoadConfig, SnapshotFilter, Warning, WarningSeverity)
- [x] 1.2 Create `internal/skills/loader.go` with SKILL.md parser (frontmatter extraction, body separation, lenient validation, max 256KB file size)
- [x] 1.3 Create `internal/skills/verify.go` with prompt injection scanning (critical patterns block loading, warning patterns log only)
- [x] 1.4 Create `internal/skills/eligibility.go` with filtering logic (OS, disabled skills, allowlist)
- [x] 1.5 Add YAML dependency to go.mod via `go get`

## Implementation Details

### Relevant Files
- `internal/skills/types.go` — New file: all type definitions
- `internal/skills/loader.go` — New file: SKILL.md parser
- `internal/skills/verify.go` — New file: security scanning
- `internal/skills/eligibility.go` — New file: eligibility filtering
- `go.mod` — Add YAML dependency

### Dependent Files
- `internal/skills/registry.go` (task_02) — will import types and loader
- `internal/skills/catalog.go` (task_02) — will import types

### Related ADRs
- [ADR-002: SKILL.md Native Format](adrs/adr-002.md) — Defines the file format this loader must parse
- [ADR-004: Four-Level Loading Hierarchy](adrs/adr-004.md) — Defines SkillSource enum values

## Deliverables
- `internal/skills/types.go` with all types and constants
- `internal/skills/loader.go` with ParseSkillFile() and parseFrontmatter()
- `internal/skills/verify.go` with VerifyContent()
- `internal/skills/eligibility.go` with IsEligible()
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Parse valid SKILL.md with all frontmatter fields
  - [x] Parse SKILL.md with only required fields (name, description)
  - [x] Parse SKILL.md with malformed YAML (lenient fallback)
  - [x] Skip SKILL.md with missing description
  - [x] Skip SKILL.md with completely unparseable YAML
  - [x] Handle empty body content after frontmatter
  - [x] Enforce 256KB max file size
  - [x] Validate name constraints (max 64 chars, lowercase, no consecutive hyphens)
  - [x] Detect critical prompt injection patterns (block loading)
  - [x] Detect warning-level patterns (log, allow loading)
  - [x] Pass clean content through verification
  - [x] Filter by OS (include matching, exclude non-matching)
  - [x] Filter by disabled list
  - [x] Filter by allowlist (empty = all, populated = restricted)
  - [x] SkillSource String() method returns human-readable source name
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All types compile and are importable by other packages
- ParseSkillFile successfully parses AgentSkills-compliant SKILL.md files
- VerifyContent detects all critical injection patterns
- IsEligible correctly filters skills by all criteria
- `make verify` passes (fmt + lint + test + build)
