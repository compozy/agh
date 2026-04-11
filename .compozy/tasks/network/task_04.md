---
status: completed
title: Session space opt-in and metadata
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 04: Session space opt-in and metadata

## Overview

Add `Space` as a first-class session attribute across session creation, persistence, query surfaces, and resume flows. This task makes network participation explicit and durable without coupling the session package to transport or routing implementations.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add optional `Space` support to session creation inputs, shared API contracts, and CLI flags
- MUST persist `Space` through `SessionMeta`, session read models, resume flows, and any session index/read APIs that surface session details
- MUST keep sessions without `Space` isolated and unchanged in behavior
- MUST avoid importing `internal/network` into `internal/session`; session code should only store and surface metadata at this stage
</requirements>

## Subtasks
- [x] 4.1 Add `Space` to session create/resume inputs and session runtime read models
- [x] 4.2 Persist `Space` in `SessionMeta` and any indexed session metadata needed by list/get operations
- [x] 4.3 Extend CLI and shared contract payloads to accept and return `Space`
- [x] 4.4 Add unit and integration coverage for create, list, stop, and resume flows with and without `Space`

## Implementation Details

This task should only handle opt-in metadata and persistence. Joining a space, booting network services, and auto-injecting network-specific prompt content are handled in later tasks.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Session manager integration and runtime-created space sections
- `internal/session/manager.go` - Extend `CreateOpts` and manager-owned state
- `internal/session/session.go` - Add `Space` to runtime/session info snapshots
- `internal/session/manager_start.go` - Restore persisted session metadata during start and resume flows
- `internal/store/types.go` - Persist `Space` in `SessionMeta`
- `internal/store/globaldb/global_db.go` - Extend indexed session storage if session queries need to surface `Space`
- `internal/api/contract/contract.go` - Add `Space` to shared DTOs
- `internal/cli/session.go` - Add `--space` to session creation commands

### Dependent Files
- `internal/network/manager.go` - Will later join and leave spaces using this metadata
- `internal/session/manager_prompt.go` - Prompt provenance will use session-level network opt-in state
- `internal/api/udsapi/routes.go` - UDS handlers will surface the new contract fields
- `internal/skills/bundled/skills/agh-network/SKILL.md` - Bundled skill is activated only for sessions that opt into a space

### Related ADRs
- [ADR-002: Session-as-Peer Identity Model](adrs/adr-002.md) - Network participation remains session-scoped
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Defines explicit `--space` participation as the v0 model

## Deliverables
- Session model, persistence, and contract updates for optional `Space`
- CLI support for explicit network opt-in at session creation time
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for create and resume behavior with `Space` **(REQUIRED)**

## Tests
- Unit tests:
- [x] `CreateOpts`, `SessionMeta`, and session read models preserve `Space` when provided
- [x] Sessions created without `Space` remain behaviorally unchanged
- [x] Resume flows reload persisted `Space` metadata accurately
- [x] Contract and parser conversions preserve `Space` consistently
- Integration tests:
- [x] `agh session` creation and resume surfaces round-trip `Space` through CLI, UDS, and manager layers
- Test coverage target: >=80%
- All tests must pass

## Validation Evidence
- `go test ./internal/session ./internal/store ./internal/store/globaldb ./internal/observe ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- `go test -cover ./internal/session ./internal/store ./internal/store/globaldb ./internal/observe ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- Coverage results:
  `internal/session` 83.0%, `internal/store` 80.3%, `internal/store/globaldb` 80.4%, `internal/observe` 81.6%, `internal/api/core` 80.3%, `internal/api/httpapi` 82.1%, `internal/api/udsapi` 81.8%, `internal/cli` 80.1%
- `make codegen`
  Required after contract changes so `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` matched the new `Space` field.
- `make verify`
  Passed after regenerating the OpenAPI artifacts.

## Success Criteria
- All tests passing
- Test coverage >=80%
- Session metadata supports explicit network opt-in without pulling in transport concerns
- Later daemon and prompt tasks can consume `Space` directly from canonical session state
