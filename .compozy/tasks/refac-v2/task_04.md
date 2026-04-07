---
status: pending
title: Re-root shared API core into internal/api/core and merge apisupport
type: refactor
complexity: high
dependencies:
  - task_02
---

# Task 04: Re-root shared API core into internal/api/core and merge apisupport

## Overview
This task moves the current shared transport layer into the target `internal/api/core` package and folds `internal/apisupport` into it. It establishes the stable server-side API boundary that HTTP and UDS transports will consume in the next phase.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- Shared handler logic, parsers, SSE helpers, and transport-facing interfaces MUST move from `internal/apicore` into `internal/api/core`.
- `internal/apisupport` MUST be merged into the new `internal/api/core` boundary rather than retained as a vestigial package.
- Public behavior for status mapping, SSE event shapes, and handler responses MUST remain unchanged during the package move.
- Any temporary re-export or bridge introduced during the move MUST be removed before the task is complete.
</requirements>

## Subtasks
- [ ] 4.1 Create `internal/api/core` and migrate shared interfaces, handlers, parsers, conversions, and SSE helpers into it.
- [ ] 4.2 Move `apisupport` functionality into the new package and remove the vestigial package boundary.
- [ ] 4.3 Update imports in shared API tests and dependent packages.
- [ ] 4.4 Remove any task-local bridges before declaring the task complete.

## Implementation Details
Follow the TechSpec `Component Overview`, `Core Interfaces`, and `Build Order` sections. Keep the new package focused on shared server-side API behavior. Route registration and transport lifecycle stay out of scope for this task.

### Relevant Files
- `internal/apicore/interfaces.go` — Defines the shared runtime service interfaces.
- `internal/apicore/handlers.go` — Contains the shared handlers that become the core package.
- `internal/apicore/parsers.go` — Shared query parsing logic for both transports.
- `internal/apicore/sse.go` — Shared SSE utilities that must move with the core.
- `internal/apisupport/session_workspace.go` — Vestigial helper package to merge into the new boundary.

### Dependent Files
- `internal/apicore/handlers_test.go` — Core handler behavior must remain covered after the move.
- `internal/apicore/conversions_parsers_test.go` — Validates conversions and parsers that will move to `api/core`.
- `internal/httpapi/shared.go` — Will change imports to the new core package.
- `internal/udsapi/shared.go` — Will change imports to the new core package.
- `internal/httpapi/server.go` — Uses shared interfaces and helpers that will be re-rooted.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Establishes the `internal/api/*` subtree target.
- [ADR-002: Make internal/api/contract the Canonical Shared API Contract](../adrs/adr-002.md) — Separates DTO ownership from the new core package.

## Deliverables
- `internal/api/core` package containing the shared server-side API layer.
- `internal/apisupport` merged and retired.
- Shared API tests updated to run against the new package path.
- Unit tests and required validation proving handler, parser, and SSE behavior remain stable with at least 80% coverage in touched packages.

## Tests
- Unit tests:
  - [ ] Shared handler tests still return the same status codes and payloads after the package move.
  - [ ] Shared parser tests still validate `since`, `last`, and cursor query semantics unchanged.
  - [ ] Shared SSE helpers still emit correctly formatted messages and IDs.
  - [ ] Session/workspace support helpers from `apisupport` still behave the same after merge.
- Integration tests:
  - [ ] Shared API behavior remains green when exercised through existing transport tests after imports are updated.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/api/core` owns the shared server-side API layer
- `internal/apisupport` no longer exists as a separate package boundary
