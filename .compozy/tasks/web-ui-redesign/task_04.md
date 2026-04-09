---
status: pending
title: Add Skills HTTP endpoints
type: backend
complexity: medium
dependencies: []
---

# Task 04: Add Skills HTTP endpoints

## Overview

Add HTTP endpoints for listing, getting, enabling, and disabling skills. This exposes the existing `skills.Registry` package through the HTTP API, following the established handler patterns (Gin router → BaseHandlers → domain service → contract payload). The Skills frontend system (task_05) depends on these endpoints.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "API Endpoints" and "Core Interfaces" sections for endpoint specs
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `SkillPayload` and `ProvenancePayload` types to `internal/api/contract/contract.go`
- MUST add `SkillsRegistry` field (or equivalent interface) to `BaseHandlers` in `internal/api/core/`
- MUST implement `ListSkills` handler: `GET /api/skills?workspace=:id` returning `{"skills": SkillPayload[]}`
- MUST implement `GetSkill` handler: `GET /api/skills/:name?workspace=:id` returning `{"skill": SkillPayload}`
- MUST implement `EnableSkill` handler: `POST /api/skills/:name/enable?workspace=:id` returning `{"ok": true}`
- MUST implement `DisableSkill` handler: `POST /api/skills/:name/disable?workspace=:id` returning `{"ok": true}`
- MUST add `SkillPayloadFromSkill` conversion helper in `internal/api/core/conversions.go`
- MUST register skill routes in `internal/api/httpapi/server.go`
- MUST follow existing handler error mapping pattern with `StatusForSkillError()` function
- MUST pass `make verify` (fmt → lint → test → build)
</requirements>

## Subtasks
- [ ] 4.1 Add `SkillPayload`, `ProvenancePayload`, and `SkillActionResponse` to contract types
- [ ] 4.2 Add `SkillPayloadFromSkill` conversion helper and `StatusForSkillError` error mapper
- [ ] 4.3 Add skills registry interface and handler methods to BaseHandlers
- [ ] 4.4 Register `/api/skills` route group in HTTP server
- [ ] 4.5 Wire skills registry into daemon composition root
- [ ] 4.6 Write table-driven handler tests and integration tests

## Implementation Details

See TechSpec "API Endpoints" section for the complete endpoint specification table.

Follow the exact same handler pattern as existing workspace/session handlers: bind request → validate → call service → convert → respond. The skills registry's `Get()`, `List()`, and `ForWorkspace()` methods provide the data. Enable/disable operations need to be added or discovered in the registry API.

### Relevant Files
- `internal/api/contract/contract.go` — Add new DTO types
- `internal/api/core/handlers.go` — Add handler methods and SkillsRegistry field
- `internal/api/core/conversions.go` — Add SkillPayloadFromSkill conversion
- `internal/api/core/errors.go` — Add StatusForSkillError mapper
- `internal/api/httpapi/server.go` — Register new route group
- `internal/skills/registry.go` — Registry public API (List, Get, ForWorkspace)
- `internal/skills/skill.go` — Skill type definition
- `internal/daemon/` — Composition root where registry is wired

### Dependent Files
- `internal/api/core/handlers.go` — BaseHandlers struct gets new field
- `internal/daemon/boot.go` or equivalent — Must inject SkillsRegistry into handler config

### Related ADRs
- [ADR-003: Full Systems Architecture for Skills and Knowledge](../adrs/adr-003.md) — Mandates real backend endpoints, no mock data

## Deliverables
- `SkillPayload` and `ProvenancePayload` in contract.go
- Four handler methods (ListSkills, GetSkill, EnableSkill, DisableSkill)
- Conversion helper and error mapper
- Route registration in server.go
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for skills API **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `SkillPayloadFromSkill` converts all fields correctly including optional provenance
  - [ ] `SkillPayloadFromSkill` omits empty optional fields (version, content, metadata)
  - [ ] `StatusForSkillError` returns 404 for not-found errors
  - [ ] `StatusForSkillError` returns 400 for validation errors
  - [ ] `ListSkills` without workspace query param returns 400
  - [ ] `ListSkills` with valid workspace returns skill list JSON
  - [ ] `GetSkill` with unknown name returns 404
  - [ ] `GetSkill` with valid name returns skill detail JSON with content
  - [ ] `EnableSkill` returns `{"ok": true}` on success
  - [ ] `DisableSkill` returns `{"ok": true}` on success
- Integration tests:
  - [ ] Full HTTP request cycle: list → get → enable → disable with real skills registry
- Test coverage target: >=80%
- All tests must pass
- `make verify` passes

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt → lint → test → build)
- Skills endpoints return correct JSON matching contract types
- Endpoints work with real skills registry loaded from test fixtures
