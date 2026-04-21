---
status: pending
title: Session Provider Runtime Plumbing and On-Disk Persistence
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Session Provider Runtime Plumbing and On-Disk Persistence

## Overview

Thread the effective session provider through AGH's runtime and persisted session metadata. This task makes the chosen provider part of the durable session read model, ensures create/resume validate before writing metadata or starting a driver, and gives later transport layers a stable field to expose.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and task_01 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Data Models", "Testing Approach", "Build Order", and "Monitoring and Observability"
- VALIDATION MUST HAPPEN BEFORE PERSISTENCE - do not write metadata or touch the global index for invalid provider selections
- `provider` MUST FLOW THROUGH RUNTIME AND STORAGE TOGETHER - do not add API-only fields detached from session persistence
- NO SILENT FALLBACKS - if the requested or persisted provider is unavailable, the lifecycle must fail explicitly
- GREENFIELD: nao aceitar `provider` apenas em memoria; o estado da sessao precisa sobreviver a resume e query
</critical>

<requirements>
- MUST add `Provider string` to `session.CreateOpts`, `session.Session`, `session.Info`, `store.SessionMeta`, and `store.SessionInfo`
- MUST thread `CreateOpts.Provider` through create, start, resume, query, and status conversion paths
- MUST call the session-aware config helper from task_01 inside `prepareSessionStartRuntime`
- MUST validate the selected provider against `spec.workspace.Config` before `writeMeta`, before global registration, and before `driver.Start`
- MUST persist the effective provider via `Session.Meta()` and restore it through `sessionInfoFromMeta()` and other read-model assembly helpers
- MUST return an explicit error when a persisted provider can no longer be resolved during resume
- SHOULD add structured logging fields for create/resume failures and legacy repair handoff points as described in the TechSpec
</requirements>

## Subtasks
- [ ] 2.1 Add `Provider` to the runtime and persistence structs in `internal/session` and `internal/store`
- [ ] 2.2 Thread `CreateOpts.Provider` through create/start/resume flows
- [ ] 2.3 Enforce provider validation ordering before metadata writes and driver startup
- [ ] 2.4 Round-trip provider through `Session.Meta()`, metadata parsing, and session info conversion
- [ ] 2.5 Add lifecycle tests for create, resume, query, and unavailable-provider failures

## Implementation Details

See TechSpec "Data Models", "Testing Approach", and ADR-003. This task should stop short of global DB migration and transport contracts; its job is to make the session runtime itself authoritative and durable so later tasks can project it outward safely.

### Relevant Files
- `internal/session/manager.go` - session creation entrypoint and `CreateOpts` ownership
- `internal/session/manager_start.go` - runtime preparation, validation ordering, and driver startup path
- `internal/session/session.go` - in-memory session state and metadata serialization
- `internal/session/query.go` - list/get/status read-model assembly
- `internal/store/types.go` - shared persistence/read-model structs for sessions
- `internal/store/meta.go` - metadata read/write helpers for session state
- `internal/session/manager_test.go` - lifecycle behavior coverage
- `internal/session/session_test.go` - metadata round-trip coverage
- `internal/session/query_test.go` - read-model conversion coverage

### Dependent Files
- `internal/store/globaldb/global_db_session.go` - task_03 will persist provider into the global session index
- `internal/api/core/conversions.go` - task_04 will convert the new `session.Info.Provider` into transport payloads
- `internal/api/contract/contract.go` - task_04 will expose provider over explicit create/read surfaces
- `.compozy/tasks/session-driver-override/task_03.md` - migration and repair depend on this runtime shape

### Related ADRs
- [ADR-001: Model Session Driver Selection As A Provider Override](adrs/adr-001.md) - keeps provider override scoped to session runtime
- [ADR-002: Re-Resolve Provider-Owned Runtime Fields On Session Override](adrs/adr-002.md) - ensures lifecycle code consumes coherent resolution output
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - defines persistence and failure semantics

## Deliverables
- Runtime and storage structs updated to carry `provider`
- Create/start/resume/query lifecycle paths wired to the effective session provider
- Validation ordering that rejects invalid providers before metadata or driver side effects
- Session metadata round-trip coverage for the provider field **(REQUIRED)**
- Explicit resume failures for unavailable persisted providers **(REQUIRED)**
- Test coverage >=80% for the touched `internal/session` and `internal/store` package(s) **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `CreateOpts.Provider` propagates into the started session runtime
  - [ ] `Session.Meta()` serializes the effective provider and metadata parsing restores it
  - [ ] `sessionInfoFromMeta()` and related conversion helpers include provider in the read model
  - [ ] Validation failure happens before `writeMeta` is called
  - [ ] Resume with an unavailable persisted provider returns a descriptive error instead of falling back
- Integration tests:
  - [ ] Starting a session with an invalid provider fails before metadata/global state is written
  - [ ] A valid session persists provider state across stop/resume and read-model queries
  - [ ] Structured logs and error payloads include `session_id`, `agent_name`, `provider`, and phase where applicable
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The effective provider is part of AGH's durable session runtime model
- Invalid or unavailable providers fail before side effects and without fallback
