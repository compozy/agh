---
status: pending
title: Add CLI automation command group
type: backend
complexity: medium
dependencies:
  - task_07
---

# Task 08: Add CLI automation command group

## Overview

Add the `agh automation` command tree so operators and agents can manage automation without talking to raw API endpoints. This task should follow existing CLI patterns for output formatting, daemon-client usage, and workspace-aware flag handling while exposing the job, trigger, and run workflows defined in the TechSpec.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add an `automation` command group to the root CLI with subcommands for jobs, triggers, and runs as listed in the TechSpec.
2. MUST add daemon-client methods that target the automation API paths rather than duplicating request logic inside Cobra commands.
3. MUST parse scope, workspace, schedule, retry, and filter flags consistently with the hardened automation model, including helpful validation errors for config-backed mutation restrictions.
4. SHOULD support the repo's standard output modes (`human`, `json`, `toon`) for list, detail, and history commands where existing CLI patterns allow it.
</requirements>

## Subtasks
- [ ] 8.1 Add daemon-client request methods for automation jobs, triggers, runs, and manual job fires
- [ ] 8.2 Add the `agh automation` Cobra command tree and subcommands
- [ ] 8.3 Add flag parsing and validation for scope, workspace, schedule, filter, and retry inputs
- [ ] 8.4 Add renderers and error messages consistent with the rest of the CLI
- [ ] 8.5 Add command and client tests for happy-path and invalid-usage cases

## Implementation Details

Follow the TechSpec sections "CLI Commands" and "API Endpoints". Reuse the existing CLI split between `internal/cli/client.go` for daemon transport, `internal/cli/root.go` for command registration, and per-domain command files for Cobra command composition.

### Relevant Files
- `internal/cli/root.go` — The new `automation` command group must be registered here
- `internal/cli/client.go` — All daemon-client automation methods should live here
- `internal/cli/memory.go` — Existing command and flag parsing patterns are a good reference for a new domain command group
- `internal/cli/helpers_test.go` — CLI test helpers and stub client patterns should be reused for automation coverage

### Dependent Files
- `internal/api/httpapi/routes.go` — The CLI will consume the transport surface added in the previous task
- `internal/api/contract/` — CLI request and response decoding should reuse the shared DTOs rather than custom structs

### Related ADRs
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — The CLI should expose one coherent automation surface for both schedules and triggers

## Deliverables
- New `agh automation` command group and daemon-client methods
- CLI output and validation behavior for jobs, triggers, and runs
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for CLI-to-daemon automation flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Client methods issue the expected HTTP verbs and `/api/automation/*` paths for list, create, update, delete, trigger, and history operations
  - [ ] Job creation flag parsing accepts `--scope workspace --workspace <id>` and rejects missing workspace input
  - [ ] Trigger creation flag parsing accepts `data.branch=main` style filters and rejects malformed filter expressions
  - [ ] CLI commands surface a descriptive error when attempting to mutate a config-backed definition beyond the enabled overlay
- Integration tests:
  - [ ] `agh automation jobs create` round-trips through a stub daemon client and prints a created job in human and JSON output modes
  - [ ] `agh automation triggers history <id>` and `agh automation runs` render run history correctly for both human and JSON output
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Operators can manage automation end-to-end through the CLI without calling raw API endpoints
- CLI validation and output stay consistent with the existing AGH command surface

