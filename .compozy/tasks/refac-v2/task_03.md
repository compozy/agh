---
status: pending
title: Migrate CLI to internal/api/contract
type: refactor
complexity: medium
dependencies:
  - task_02
---

# Task 03: Migrate CLI to internal/api/contract

## Overview
This task removes the CLI’s duplicate daemon contract definitions and makes `internal/cli` consume `internal/api/contract`. It narrows the contract split before the API transport packages are re-rooted, reducing the risk of two divergent client surfaces during the broader move.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- `internal/cli/client.go` MUST stop owning duplicate shared daemon DTO definitions that now belong in `internal/api/contract`.
- CLI request and response handling MUST remain compatible with the daemon API paths and JSON shapes defined in the TechSpec.
- CLI-specific helper types MAY remain local only when they are not part of the shared daemon contract.
- CLI integration behavior MUST remain unchanged for session, workspace, agent, observe, memory, and daemon commands.
</requirements>

## Subtasks
- [ ] 3.1 Replace CLI-owned shared DTO usage with imports from `internal/api/contract`.
- [ ] 3.2 Remove duplicate shared contract definitions from `internal/cli/client.go`.
- [ ] 3.3 Keep any CLI-only helper or presentation types local when they are not part of the daemon contract.
- [ ] 3.4 Update CLI tests to validate the migrated contract usage.

## Implementation Details
Use the TechSpec `System Architecture`, `Data Models`, and `API Endpoints` sections. This task should not re-root packages yet. Its purpose is to make CLI a consumer of the shared contract package ahead of transport moves.

### Relevant Files
- `internal/cli/client.go` — Holds the duplicate request and response types that must be removed or narrowed.
- `internal/cli/format.go` — Depends on the shapes of CLI records returned from the daemon client surface.
- `internal/cli/cli_integration_test.go` — Validates end-to-end CLI behavior against the daemon runtime.
- `internal/apicore/payloads.go` — Useful comparison point to confirm contract parity during migration.
- `internal/api/contract` — New canonical source for shared DTOs after task 02.

### Dependent Files
- `internal/cli/session.go` — Consumes session records and event records returned by the client.
- `internal/cli/workspace.go` — Consumes workspace records and detail payloads.
- `internal/cli/agent.go` — Consumes agent payloads.
- `internal/cli/memory.go` — Consumes memory request and response shapes.
- `internal/cli/observe.go` — Consumes observe event and health payloads.

### Related ADRs
- [ADR-002: Make internal/api/contract the Canonical Shared API Contract](../adrs/adr-002.md) — Requires CLI to depend on the shared contract package.

## Deliverables
- CLI daemon client updated to use `internal/api/contract` for shared request and response types.
- Duplicate shared DTO definitions removed from CLI code.
- Updated unit and integration tests confirming CLI behavior is unchanged.
- Coverage of at least 80% across touched CLI package areas.

## Tests
- Unit tests:
  - [ ] Decoding a session list response into CLI-facing types still succeeds with `api/contract` DTOs.
  - [ ] Memory write and read request/response handling still uses the same JSON shape after migration.
  - [ ] Observe event and daemon status decoding remain stable after removing local DTO copies.
- Integration tests:
  - [ ] CLI integration flows for session creation, prompting, workspace listing, and daemon status still pass against the runtime.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- CLI no longer owns duplicate shared daemon contract DTOs
- CLI behavior remains unchanged against the daemon API
