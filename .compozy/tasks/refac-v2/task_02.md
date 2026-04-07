---
status: pending
title: Create internal/api/contract and migrate shared DTOs
type: refactor
complexity: high
dependencies:
  - task_01
---

# Task 02: Create internal/api/contract and migrate shared DTOs

## Overview
This task introduces `internal/api/contract` as the canonical shared daemon API contract and moves transport-agnostic DTOs into it. It establishes a single source of truth before the API subtree is re-rooted and before CLI migration begins.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- `internal/api/contract` MUST own shared daemon request and response DTOs used across transports and CLI.
- Transport-agnostic payloads currently in `internal/apicore/payloads.go` MUST be migrated into `internal/api/contract`.
- Transport-specific envelopes, including HTTP-only AI SDK streaming payloads, SHOULD remain outside `internal/api/contract`.
- Existing JSON field names and route-level contract shapes MUST remain backward-compatible during the migration.
</requirements>

## Subtasks
- [ ] 2.1 Create `internal/api/contract` and move shared session, workspace, agent, memory, observe, and daemon DTOs into it.
- [ ] 2.2 Update `internal/apicore` to consume the contract package instead of owning shared DTO definitions.
- [ ] 2.3 Preserve transport-specific payload types outside the shared contract package.
- [ ] 2.4 Add parity tests that confirm moved DTOs keep the same JSON shape as the current API surface.

## Implementation Details
Follow the TechSpec `System Architecture`, `Data Models`, and `API Endpoints` sections. This task only establishes contract ownership. Do not couple CLI to handler logic and do not pull transport-only SSE payloads into the shared package.

### Relevant Files
- `internal/apicore/payloads.go` — Current home for the shared transport DTO set.
- `internal/apicore/conversions.go` — Converts runtime models into the DTOs that will move.
- `internal/httpapi/shared.go` — Aliases several shared payloads that should switch to `api/contract`.
- `internal/udsapi/shared.go` — Consumes shared event payloads from the current API core.
- `internal/httpapi/prompt.go` — Demonstrates HTTP-only streaming payloads that should stay local.

### Dependent Files
- `internal/apicore/handlers.go` — Will need imports updated to the new contract package.
- `internal/apicore/handlers_test.go` — Must continue validating payload semantics after the move.
- `internal/httpapi/shared_test.go` — Covers shared payload behavior from the HTTP transport side.
- `internal/udsapi/shared_test.go` — Covers shared payload behavior from the UDS transport side.
- `internal/cli/client.go` — Will become a downstream consumer in the next task.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Places the API under an explicit subtree.
- [ADR-002: Make internal/api/contract the Canonical Shared API Contract](../adrs/adr-002.md) — Defines ownership for shared daemon DTOs.

## Deliverables
- `internal/api/contract` package containing the canonical shared daemon DTOs.
- `internal/apicore` updated to import the new contract package for shared types.
- Transport-specific payloads explicitly kept out of `api/contract`.
- Unit tests proving JSON parity and shared payload usage with at least 80% coverage in touched packages.

## Tests
- Unit tests:
  - [ ] Session DTOs in `api/contract` serialize to the same JSON field names currently returned by the transports.
  - [ ] Workspace DTOs preserve optional fields and omit-empty behavior after migration.
  - [ ] Agent-event payloads that are part of the shared daemon contract still round-trip correctly through conversions.
  - [ ] HTTP-only AI SDK prompt payloads remain outside the shared contract package.
- Integration tests:
  - [ ] Shared API handler tests continue returning the same response shapes for session, agent, workspace, and memory endpoints.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/api/contract` is the only owner of shared daemon DTOs
- Shared contract shapes remain backward-compatible
