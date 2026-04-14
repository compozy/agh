---
status: pending
title: Implement ClawHub and GitHub adapters
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 03: Implement ClawHub and GitHub adapters

## Overview

Create two `RegistrySource` implementations: a ClawHub adapter (refactoring the existing client at `internal/skills/marketplace/clawhub/`) and a new GitHub Releases adapter. ClawHub is skills-only with `SourceCaps{Search: true}`. GitHub is a slug-only source with `SourceCaps{Search: false}` — `Search()` returns `ErrNotSupported`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- ClawHub adapter MUST implement `RegistrySource` and return `SourceCaps{Search: true}`
- ClawHub adapter MUST adapt `Download(ctx, slug) → *SkillArchive` to `Download(ctx, slug, opts) → *DownloadResult`
- ClawHub adapter MUST map `DownloadOpts.Version`: empty → `/skills/<slug>/download`, specified → versioned endpoint
- ClawHub adapter MUST ignore `DownloadOpts.Asset` (single-asset downloads)
- GitHub adapter MUST implement `RegistrySource` and return `SourceCaps{Search: false}`
- GitHub adapter `Search()` MUST return `ErrNotSupported`
- GitHub adapter MUST use GitHub Releases API: `/repos/{owner}/{repo}/releases/latest` and `/releases/tags/{tag}`
- GitHub adapter MUST use `DownloadOpts.Asset` for multi-asset disambiguation
- GitHub adapter MUST handle auto-generated source archives (`<repo>-<tag>/` prefix directory)
- GitHub adapter MUST validate Content-Type before extraction
- GitHub adapter MUST handle rate limits (`X-RateLimit-Remaining` header)
- GitHub adapter MUST exclude pre-release and draft releases by default
- MUST pass `make verify` after completion
</requirements>

## Subtasks
- [ ] 3.1 Create `internal/registry/clawhub/` adapter wrapping existing client logic
- [ ] 3.2 Adapt ClawHub `Download` signature to new `DownloadOpts` / `*DownloadResult` types
- [ ] 3.3 Create `internal/registry/github/` adapter with Releases API integration
- [ ] 3.4 Implement GitHub asset resolution with naming convention and `--asset` fallback
- [ ] 3.5 Implement GitHub rate-limit handling and `GITHUB_TOKEN` authentication
- [ ] 3.6 Write unit tests for both adapters using httptest servers

## Implementation Details

See TechSpec "Integration Points > ClawHub" and "Integration Points > GitHub Releases" sections for API details.

### Relevant Files
- `internal/skills/marketplace/clawhub/client.go` — Existing client to refactor (Search, Download, Info methods)
- `internal/skills/marketplace/registry.go:7-12` — Existing `marketplace.Registry` interface being replaced
- `internal/skills/marketplace/types.go:15-20` — `SkillArchive` type (Slug, Version, Data)
- `internal/registry/source.go` — `RegistrySource` interface from task_01
- `internal/registry/types.go` — `DownloadOpts`, `DownloadResult` types from task_01

### Dependent Files
- `internal/cli/extension.go` — Will use GitHub adapter (task_04)
- `internal/cli/skill_commands.go` — Will use ClawHub adapter (task_05)

### Related ADRs
- [ADR-001: Multi-Source RegistrySource Interface](adrs/adr-001.md) — Interface that adapters implement
- [ADR-003: tar.gz Archive as Universal Distribution Format](adrs/adr-003.md) — Archive format both adapters produce

## Deliverables
- `internal/registry/clawhub/` — ClawHub RegistrySource adapter
- `internal/registry/github/` — GitHub Releases RegistrySource adapter
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests (ClawHub):
  - [ ] Search with query returns parsed SkillListing results
  - [ ] Search with empty results returns empty slice
  - [ ] Info returns SkillDetail with MCPServers field preserved
  - [ ] Download with empty version calls `/skills/<slug>/download`
  - [ ] Download with specified version calls versioned endpoint
  - [ ] Download populates DownloadResult.Version from `X-Skill-Version` header
  - [ ] Capabilities() returns `SourceCaps{Search: true}`
  - [ ] Retry logic on 5xx errors (exponential backoff)
  - [ ] Close() is safe to call multiple times
- Unit tests (GitHub):
  - [ ] Search returns `ErrNotSupported`
  - [ ] Capabilities() returns `SourceCaps{Search: false}`
  - [ ] Info calls `/repos/{owner}/{repo}/releases/latest` and parses response
  - [ ] Info populates Versions list from page 1 of releases
  - [ ] Info excludes pre-release and draft releases
  - [ ] Download with single tar.gz asset succeeds
  - [ ] Download with multiple tar.gz assets and no `DownloadOpts.Asset` returns error listing assets
  - [ ] Download with `DownloadOpts.Asset` specified selects correct asset
  - [ ] Download with no tar.gz asset falls back to source archive
  - [ ] Download validates Content-Type — rejects `text/html` responses
  - [ ] Rate limit handling: `X-RateLimit-Remaining: 0` → clear error suggesting GITHUB_TOKEN
  - [ ] Private repo without GITHUB_TOKEN → 401 with clear error
  - [ ] Repository with no releases → clear error
  - [ ] GITHUB_TOKEN from env is sent as Bearer token in Authorization header
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- ClawHub adapter passes all existing marketplace test scenarios
- GitHub adapter handles all documented edge cases (multi-asset, no-asset, auth, rate-limit)
