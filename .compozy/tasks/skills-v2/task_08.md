---
status: completed
title: "Implement marketplace interface and ClawHub client"
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 8: Implement marketplace interface and ClawHub client

## Overview

Create the pluggable marketplace `Registry` interface and the ClawHub HTTP client implementation. The interface defines Search, Download, and Info methods. The ClawHub client implements these against the ClawHub API with exponential backoff, context-aware cancellation, and proper error handling.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/marketplace/registry.go` with `Registry` interface
- MUST define `SkillListing`, `SkillArchive`, `SkillDetail`, `SearchOpts` types in `internal/skills/marketplace/types.go`
- MUST create `internal/skills/marketplace/clawhub/client.go` implementing `Registry` interface
- MUST implement exponential backoff (1s initial, 30s max, 3 retries) for HTTP requests
- MUST use `context.Context` for cancellation propagation
- MUST set 30s HTTP timeout per request
- MUST implement `Search()` → `GET /api/v1/skills?q=<query>&limit=<n>`
- MUST implement `Info()` → `GET /api/v1/skills/<slug>`
- MUST implement `Download()` → `GET /api/v1/skills/<slug>/download` returning tar.gz stream
- MUST handle HTTP error responses with descriptive error messages
- MUST provide compile-time interface verification: `var _ marketplace.Registry = (*Client)(nil)`
</requirements>

## Subtasks
- [x] 8.1 Create marketplace/registry.go with Registry interface definition
- [x] 8.2 Create marketplace/types.go with SkillListing, SkillArchive, SkillDetail, SearchOpts
- [x] 8.3 Implement clawhub/client.go with Search, Info, Download methods
- [x] 8.4 Implement exponential backoff and context-aware HTTP requests
- [x] 8.5 Write unit tests using httptest.NewServer for all API interactions

## Implementation Details

New packages: `internal/skills/marketplace/` (interface + types) and `internal/skills/marketplace/clawhub/` (implementation). Uses stdlib `net/http` only — no external HTTP libraries.

See TechSpec "Marketplace registry interface", "ClawHub Client", and "Data Models > Marketplace types" sections.

### Relevant Files
- `internal/config/config.go` — MarketplaceConfig with Registry and BaseURL fields (from task_02)

### Dependent Files
- `internal/cli/skill.go` — marketplace CLI commands call Registry methods (task_10)
- `internal/skills/provenance.go` — install flow writes sidecar after download (task_10)

### Related ADRs
- [ADR-003: Pluggable Registry Interface](adrs/adr-003.md) — defines the interface contract and ClawHub as default

## Deliverables
- `internal/skills/marketplace/registry.go` with Registry interface
- `internal/skills/marketplace/types.go` with marketplace types
- `internal/skills/marketplace/clawhub/client.go` with ClawHub implementation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Search with query returns parsed []SkillListing from JSON response
  - [x] Search with limit param sends correct query parameter
  - [x] Search with empty results returns empty slice, not nil
  - [x] Info returns parsed SkillDetail from JSON response
  - [x] Info with unknown slug returns descriptive error
  - [x] Download returns SkillArchive with readable tar.gz stream
  - [x] Download with unknown slug returns descriptive error
  - [x] HTTP 404 → descriptive "skill not found" error
  - [x] HTTP 500 → retried with backoff, eventually returns error
  - [x] HTTP timeout → context deadline exceeded error
  - [x] Context cancelled → request aborted promptly
  - [x] Retry exhaustion after 3 attempts → final error returned
  - [x] Compile-time interface check: `var _ marketplace.Registry = (*Client)(nil)`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- All HTTP tests use httptest.NewServer (no real network calls)
