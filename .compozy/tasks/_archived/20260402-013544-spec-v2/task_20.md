---
status: completed
title: CLI Commands — Lifecycle & Hooks
type: ""
complexity: low
dependencies:
    - task_17
    - task_18
---

# Task 20: CLI Commands — Lifecycle & Hooks

## Overview
Implement the remaining Cobra CLI subcommands for agent lifecycle management (wait, done) and the internal hook-event command used by driver hook scripts to forward tool usage events to the kernel.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement agh wait that blocks until agent or workgroup reaches done/closed per docs/spec-v2/06-cli.md
- MUST implement agh done that signals agent completion with reason, sets state to done, notifies master per docs/spec-v2/06-cli.md
- MUST implement agh hook-event as internal command reading JSON from stdin, publishing to NATS per docs/spec-v2/06-cli.md
- MUST implement agh help (built-in via Cobra)
- MUST handle researcher auto-destroy on agh done (kernel terminates process)
- MUST write status entry and log event on agh done
</requirements>

## Subtasks
- [x] 20.1 Implement wait command with blocking until target reaches done/closed
- [x] 20.2 Implement done command with state transition and master notification
- [x] 20.3 Implement hook-event command (read stdin JSON, publish to NATS)
- [x] 20.4 Handle researcher auto-destroy behavior on done
- [x] 20.5 Verify help command works via Cobra built-in

## Implementation Details
Refer to docs/spec-v2/06-cli.md for wait, done, and hook-event specs. Refer to docs/spec-v2/03-agents.md for researcher auto-destroy.

### Relevant Files
- `docs/spec-v2/06-cli.md` — CLI reference for lifecycle commands
- `docs/spec-v2/03-agents.md` — researcher auto-destroy behavior

### Dependent Files
- `internal/cli/root.go` — HTTP-over-UDS connection helper (~/.agh/daemon.sock)
- `internal/transport/` — NATS messaging
- `internal/registry/` — agent state updates

## Deliverables
- internal/cli/lifecycle.go — wait, done commands
- internal/cli/hooks.go — hook-event command
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Wait blocks until target agent state is done
  - [x] Wait blocks until target workgroup state is closed
  - [x] Done sets agent state to done and writes status entry
  - [x] Done notifies workgroup master
  - [x] Done triggers researcher auto-destroy (kernel terminates process)
  - [x] Hook-event reads JSON from stdin correctly
  - [x] Hook-event publishes to correct NATS subject (agh.wg.{wg}.hook)
  - [x] Help command lists all available commands
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All lifecycle commands match docs/spec-v2/06-cli.md spec
