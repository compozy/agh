---
status: completed
title: Extract shared frontmatter package
type: refactor
complexity: high
dependencies: []
---

# Task 01: Extract shared frontmatter package

## Overview
This task creates `internal/frontmatter` as the single parser for YAML frontmatter used by `config`, `memory`, and `skills`. It removes duplicated parsing logic and shared error-shape drift before larger package moves begin.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- `internal/frontmatter` MUST become the only package that owns line-ending normalization, delimiter detection, and shared frontmatter decoding behavior.
- `internal/config`, `internal/memory`, and `internal/skills` MUST stop carrying local copies of frontmatter parsing logic and sentinel errors.
- The extracted package MUST preserve current parsing behavior for valid files, missing delimiters, unterminated frontmatter, and YAML decode failures.
- Existing callers MUST continue to return package-appropriate wrapped errors while relying on the shared parser underneath.
</requirements>

## Subtasks
- [x] 1.1 Create `internal/frontmatter` with the shared parse surface required by config, memory, and skills.
- [x] 1.2 Migrate `internal/config` agent-definition parsing to the shared package.
- [x] 1.3 Migrate `internal/memory` document parsing to the shared package.
- [x] 1.4 Migrate `internal/skills` loader and registry paths to the shared package.
- [x] 1.5 Remove duplicated parser helpers and obsolete local sentinels once all call sites are migrated.

## Implementation Details
Use the TechSpec `System Architecture` and `Development Sequencing` sections as the source of truth. Keep the extracted API small and generic enough for both byte-oriented and string-oriented callers. Preserve current behavior first; this task is about ownership and deduplication, not format changes.

### Relevant Files
- `internal/config/agent.go` — Contains one of the duplicated frontmatter parsers used for AGENT definitions.
- `internal/memory/store.go` — Contains duplicated parsing helpers and sentinel errors for memory documents.
- `internal/memory/document.go` — Consumes frontmatter parsing for document headers.
- `internal/skills/loader.go` — Contains duplicated parser logic for `SKILL.md`.
- `internal/skills/registry.go` — Reuses skill parsing during workspace/global skill loading.

### Dependent Files
- `internal/config/agent_test.go` — Must keep agent frontmatter behavior stable after the extraction.
- `internal/memory/store_test.go` — Must continue validating memory parsing and document operations.
- `internal/skills/loader_test.go` — Contains existing parser behavior cases that should move or be reused.
- `internal/skills/registry_test.go` — Exercises loader integration through registry workflows.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Establishes extraction of cross-package parsing into `internal/frontmatter`.

## Deliverables
- `internal/frontmatter` package with shared parsing entry points and error behavior.
- `config`, `memory`, and `skills` migrated to the shared package with duplicated helpers removed.
- Unit tests covering the shared parser and updated callers with at least 80% coverage in touched packages.
- Integration-safe behavior preserved for memory and skill loading call paths.

## Tests
- Unit tests:
  - [x] Parsing a valid AGENT frontmatter document returns the decoded metadata and body unchanged.
  - [x] Parsing a valid SKILL frontmatter document preserves body separation and metadata fields.
  - [x] Missing opening delimiter returns the expected missing-frontmatter failure.
  - [x] Unterminated frontmatter returns the expected unterminated-frontmatter failure.
  - [x] Invalid YAML returns a wrapped decode failure without silently succeeding.
- Integration tests:
  - [x] Loading agent definitions through `internal/config` still resolves the same agent metadata as before.
  - [x] Loading skills through `Registry.LoadAll` and workspace skill resolution still succeeds with shared parsing.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- No duplicated frontmatter parser implementations remain in `config`, `memory`, or `skills`
- Frontmatter behavior remains stable for valid and invalid documents
