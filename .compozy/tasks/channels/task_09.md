---
status: pending
title: Add CLI channel management commands
type: backend
complexity: medium
dependencies:
  - task_08
---

# Task 09: Add CLI channel management commands

## Overview

Add a first-class `agh channel` command group so operators and agents can manage channel instances from the CLI without calling raw transport endpoints. This task should mirror the new API surface while reusing the existing CLI client and output patterns already used for sessions, extensions, and hooks.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add an `agh channel` command group with list, get, create, update, enable, disable, restart, routes, and test-delivery operations backed by the shared API/client layer.
2. MUST reuse the existing CLI output modes and client transport patterns instead of inventing a channel-specific ad hoc transport path.
3. MUST surface scope, workspace, status, and route information in a way that is usable from both human and machine-oriented CLI output.
4. SHOULD keep local-fallback and daemon-connected behavior aligned with existing extension and session commands where possible.
</requirements>

## Subtasks
- [ ] 9.1 Add a `channel` command group to the root CLI and implement subcommands for the channel lifecycle
- [ ] 9.2 Extend the CLI client with channel transport methods matching the new API surface
- [ ] 9.3 Add human, JSON, and toon-style output for channel records and route inspection
- [ ] 9.4 Add CLI unit and integration tests for the new command group

## Implementation Details

Follow the TechSpec sections "HTTP / UDS API", "Operational visibility", and the existing CLI patterns already present for sessions and extensions. This task should be a thin CLI layer over the channel API surface introduced in task 08.

### Relevant Files
- `internal/cli/root.go` — The new `agh channel` command group must be registered here
- `internal/cli/client.go` — Shared daemon client methods for channel endpoints belong here
- `internal/cli/format.go` — Existing output helpers should be reused for channel payload rendering
- `internal/cli/extension.go` — Extension management is the closest CLI pattern reference for channel lifecycle commands
- `internal/cli/session.go` — Session subcommand structure is the closest reference for route-oriented read operations

### Dependent Files
- `internal/api/contract/contract.go` — CLI methods depend on the shared channel DTOs added in task 08
- `internal/api/udsapi/routes.go` — CLI transport behavior depends on the UDS server exposing the new channel endpoints
- `internal/cli/cli_integration_test.go` — Existing CLI integration coverage should expand for the new command group

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — The daemon-owned channel substrate should be operable from the CLI
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — CLI output must reflect registry-owned instance and route data

## Deliverables
- New `agh channel` command group and client methods for channel lifecycle operations
- Reused output formatting for human, JSON, and toon channel output
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for channel CLI flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `agh channel list` renders scope, platform, and status fields in human output
  - [ ] `agh channel get <id>` returns structured JSON output using the shared channel DTOs
  - [ ] `agh channel routes <id>` renders route information without flattening peer and thread fields incorrectly
  - [ ] `agh channel test-delivery` forwards the typed delivery-target payload rather than a free-form string
- Integration tests:
  - [ ] CLI create and get commands round-trip a channel instance through the daemon-backed API
  - [ ] Enable, disable, and restart commands mutate channel instance state through the same transport path as the API
  - [ ] Route-inspection commands return data for an existing channel instance and fail cleanly for an unknown instance
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Channel instances are fully operable from the CLI without manual API calls
- CLI output remains consistent with the existing AGH command patterns
