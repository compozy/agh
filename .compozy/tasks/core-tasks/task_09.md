---
status: pending
title: "Add the `agh task` CLI command group"
type: backend
complexity: medium
dependencies:
  - task_08
---

# Task 09: Add the `agh task` CLI command group

## Overview
Add the user-facing CLI surface for creating, inspecting, and controlling tasks and runs through the daemon. This task makes the new domain operable from the command line without bypassing the shared UDS/contract path.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. The CLI MUST expose task and run workflows through daemon-backed commands rather than direct store access.
2. The command set MUST cover the task and run operations accepted in the TechSpec, including task create/list/get/update, child creation, dependency management, and run lifecycle actions.
3. Flags and output MUST preserve scope, workspace, owner, channel, and lifecycle semantics consistently with the shared contract.
</requirements>

## Subtasks
- [ ] 9.1 Add the top-level `agh task` command group and its subcommands.
- [ ] 9.2 Add create/list/get/update task commands with filters and mutation flags.
- [ ] 9.3 Add child-task and dependency-management subcommands.
- [ ] 9.4 Add run enqueue, claim, start, complete, fail, and cancel subcommands.
- [ ] 9.5 Add CLI output formatting aligned with existing daemon-backed command patterns.

## Implementation Details
Use the TechSpec "API Surface" section and follow the command organization used by `internal/cli/automation.go`. The CLI should stay transport-backed, using the same contract and UDS flows the daemon exposes.

### Relevant Files
- `internal/cli/root.go` — CLI composition point for registering the new command group.
- `internal/cli/automation.go` — Reference for daemon-backed subcommand organization and output patterns.
- `internal/api/contract/` — Shared request/response payloads consumed by the CLI.
- `internal/api/udsapi/routes.go` — UDS route inventory that the CLI must match.

### Dependent Files
- `internal/api/udsapi/server.go` — Must already expose the task handlers the CLI will call.
- `cmd/agh/main.go` — Indirectly depends on the new command registration through the root command.

### Related ADRs
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Requires channel-aware flags and output.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Requires ownership semantics to be represented correctly in CLI inputs and outputs.

## Deliverables
- New `agh task` command group and subcommands backed by the daemon API.
- Flag handling and output helpers for task/runs, ownership, scope, and channel fields.
- CLI tests covering argument parsing and daemon interactions.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for CLI-to-UDS task flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify create and update commands reject invalid flag combinations before issuing daemon calls.
  - [ ] Verify list filters serialize scope, workspace, owner, status, parent, and channel arguments correctly.
  - [ ] Verify run lifecycle subcommands map CLI inputs onto the expected daemon request payloads.
- Integration tests:
  - [ ] Verify `agh task create`, `agh task list`, and `agh task get` work end-to-end against a live UDS daemon.
  - [ ] Verify run lifecycle commands can enqueue and complete a run through the daemon-backed UDS flow.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Users and agents can manage tasks from the CLI without bypassing daemon contracts
- CLI semantics match the shared UDS/API model for tasks and runs
