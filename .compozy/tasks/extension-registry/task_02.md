---
status: pending
title: Implement MultiRegistry and Installer pipeline
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Implement MultiRegistry and Installer pipeline

## Overview

Implement the `MultiRegistry` aggregator that queries multiple `RegistrySource` implementations concurrently with priority-based deduplication, and the domain-agnostic `Installer` pipeline that handles download, size-limited extraction, manifest validation, and content verification. The Installer accepts a `Downloader` interface (not `*MultiRegistry`) for testability.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `MultiRegistry` that does NOT implement `RegistrySource` interface
- MUST query sources concurrently and merge results with priority-based dedup (later sources override earlier, following `overlaySkill()` pattern)
- MUST skip sources where `Capabilities().Search == false` during `Search()` calls
- MUST implement `CheckUpdate()` using `Info()` + `versionIsNewer()` (not delegated to backends)
- MUST implement `Installer` accepting `Downloader` interface
- `Installer.Install()` MUST wrap download stream in `io.LimitReader(maxArchiveSize)`
- `Installer.Install()` MUST validate manifest presence (extension.toml or SKILL.md at root)
- `Installer` MUST NOT import `internal/extension/` or `internal/skills/` — domain-agnostic only
- MUST pass `make verify` after completion
</requirements>

## Subtasks
- [ ] 2.1 Create `internal/registry/multi.go` with concurrent query logic and priority-based dedup
- [ ] 2.2 Implement `Capabilities()`-aware `Search()` that skips non-searchable sources
- [ ] 2.3 Implement `CheckUpdate()` using `Info()` + `versionIsNewer()`
- [ ] 2.4 Create `internal/registry/installer.go` with download-extract-verify pipeline
- [ ] 2.5 Write unit tests for MultiRegistry (concurrent queries, dedup, partial failure)
- [ ] 2.6 Write unit tests for Installer (extraction, limits, manifest validation, cleanup)

## Implementation Details

See TechSpec "Core Interfaces" section for `MultiRegistry` and `Installer` signatures.

### Relevant Files
- `internal/registry/types.go` — Types defined in task_01
- `internal/registry/source.go` — RegistrySource interface from task_01
- `internal/registry/extract.go` — Extraction functions from task_01
- `internal/registry/version.go` — `versionIsNewer` from task_01
- `internal/skills/registry.go:502-515` — `overlaySkill()` pattern for dedup strategy
- `internal/skills/verify.go` — `VerifyContent()` for content security scanning

### Dependent Files
- `internal/cli/extension.go` — Will use MultiRegistry and Installer (task_04)
- `internal/cli/skill_commands.go` — Will use MultiRegistry and Installer (task_05)

### Related ADRs
- [ADR-001: Multi-Source RegistrySource Interface](adrs/adr-001.md) — MultiRegistry aggregation design

## Deliverables
- `internal/registry/multi.go` — MultiRegistry aggregator
- `internal/registry/installer.go` — Domain-agnostic Installer pipeline
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests (MultiRegistry):
  - [ ] Search with two sources returns merged results, later source overrides on slug collision
  - [ ] Search skips source where `Capabilities().Search == false`
  - [ ] Search with one erroring source still returns results from healthy sources
  - [ ] Search with all sources erroring returns combined error
  - [ ] Search with empty sources list returns empty slice (not nil)
  - [ ] Info resolves from first source that has the slug (priority order)
  - [ ] Download delegates to the correct source based on slug resolution
  - [ ] CheckUpdate returns `HasUpdate: true` when remote version is newer
  - [ ] CheckUpdate returns `HasUpdate: false` when versions are equal
  - [ ] Close() calls Close() on all sources
- Unit tests (Installer):
  - [ ] Install with valid tar.gz containing extension.toml returns InstallResult with correct checksum
  - [ ] Install with valid tar.gz containing SKILL.md returns InstallResult
  - [ ] Install where archive exceeds `maxArchiveSize` (compressed) fails with clear error
  - [ ] Install where archive exceeds `maxDecompressedSize` fails with clear error
  - [ ] Install where archive has no manifest (no extension.toml or SKILL.md) fails with clear error
  - [ ] Install cleans up temp dir on failure
  - [ ] Install with context cancellation mid-download closes reader and cleans up
  - [ ] Content-Type validation rejects `text/html` responses
  - [ ] Stale temp dir cleanup removes dirs older than 1 hour
- Integration tests:
  - [ ] Full pipeline with in-memory Downloader mock: download → extract → validate → result
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- `internal/registry/` has zero imports from `internal/extension/` or `internal/skills/`
- Installer works with any Downloader mock without needing real HTTP calls
