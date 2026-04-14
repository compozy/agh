---
status: completed
title: Extract shared extraction logic and define registry interfaces
type: refactor
complexity: high
dependencies: []
---

# Task 01: Extract shared extraction logic and define registry interfaces

## Overview

Move ~300 lines of archive extraction, path validation, version comparison, and directory-move logic from `internal/cli/skill_marketplace.go` into a new `internal/registry/` package. Add decompression-size and file-count limits to the extraction pipeline (currently unprotected `io.Copy` at line 414). Then define all shared types and interfaces (`RegistrySource`, `DownloadOpts`, `SourceCaps`, `Downloader`, `ErrNotSupported`) that subsequent tasks depend on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST move `extractMarketplaceArchive`, `pathWithinRoot`, `cleanArchiveEntryPath`, `versionIsNewer`, `moveInstalledSkillDir` into `internal/registry/`
- MUST add `maxDecompressedSize` (500MB default) via counting writer wrapping `io.Copy`
- MUST add `maxFileCount` (10000 default) enforced during tar iteration
- MUST update `skill_marketplace.go` callers to use the new shared functions
- MUST NOT break existing integration tests in `skill_marketplace_integration_test.go`
- MUST define `RegistrySource` interface with `Capabilities() SourceCaps` method
- MUST define `DownloadOpts` struct with `Version` and `Asset` fields
- MUST define `ErrNotSupported` sentinel following project error pattern
- MUST pass `make verify` after completion
</requirements>

## Subtasks
- [x] 1.1 Create `internal/registry/extract.go` with moved extraction functions and decompression limits
- [x] 1.2 Create `internal/registry/version.go` with moved `versionIsNewer` logic
- [x] 1.3 Update `internal/cli/skill_marketplace.go` to call new shared functions
- [x] 1.4 Verify `skill_marketplace_integration_test.go` passes with no changes
- [x] 1.5 Create `internal/registry/types.go` with all shared types (Listing, Detail, DownloadOpts, DownloadResult, SourceCaps, SearchOpts, PackageType, UpdateInfo, InstallResult)
- [x] 1.6 Create `internal/registry/source.go` with `RegistrySource` interface and `ErrNotSupported`
- [x] 1.7 Write unit tests for extraction limits, version comparison, and type validation

## Implementation Details

See TechSpec "Core Interfaces" and "Build Order Step 1-2" sections for interface definitions and type specifications.

### Relevant Files
- `internal/cli/skill_marketplace.go` — Source of functions to extract (lines 365-607)
- `internal/cli/skill_marketplace_integration_test.go` — Integration tests that must continue passing
- `internal/skills/marketplace/types.go` — Existing SkillArchive/SearchOpts patterns to align with
- `internal/extension/registry.go:23-30` — Sentinel error pattern to follow for `ErrNotSupported`
- `internal/skills/marketplace/clawhub/client.go:315-317` — Existing `io.LimitReader` pattern

### Dependent Files
- `internal/cli/skill_marketplace.go` — Must be updated to import and call new shared functions
- All future `internal/registry/` files — Depend on types and interfaces defined here

### Related ADRs
- [ADR-001: Multi-Source RegistrySource Interface](adrs/adr-001.md) — Defines the interface shape
- [ADR-003: tar.gz Archive as Universal Distribution Format](adrs/adr-003.md) — Extraction pipeline requirements

## Deliverables
- `internal/registry/extract.go` — Extraction functions with decompression limits
- `internal/registry/version.go` — Version comparison logic
- `internal/registry/types.go` — All shared types
- `internal/registry/source.go` — RegistrySource interface and sentinels
- Updated `internal/cli/skill_marketplace.go` — Thin wrappers calling shared functions
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests passing **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `extractArchive` with valid tar.gz produces expected directory structure
  - [x] `extractArchive` exceeding `maxDecompressedSize` (500MB) returns error before exhausting disk
  - [x] `extractArchive` exceeding `maxFileCount` (10000) returns error
  - [x] `extractArchive` rejects symlinks in archives
  - [x] `extractArchive` rejects path traversal (`../` in entry names)
  - [x] `pathWithinRoot` accepts valid child paths, rejects escape attempts
  - [x] `cleanArchiveEntryPath` rejects absolute paths and `..` components
  - [x] `versionIsNewer` with semver: "1.2.0" newer than "1.1.0" → true
  - [x] `versionIsNewer` with prerelease: "1.0.0-beta" older than "1.0.0" → true
  - [x] `versionIsNewer` with invalid version strings → false (no panic)
  - [x] `SourceCaps` zero value has `Search: false`
  - [x] `ErrNotSupported` matches via `errors.Is`
- Integration tests:
  - [x] Existing `TestSkillInstallCommandIntegrationCreatesSkillDirectoryAndSidecar` passes unchanged
  - [x] Existing `TestSkillInstallAndRemoveIntegrationRefreshesRegistry` passes unchanged
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt, lint, test, build)
- Existing skill marketplace integration tests pass without modification
- No new external dependencies added
