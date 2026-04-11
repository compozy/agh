---
status: completed
title: Prompt provenance and ACP guardrails
type: backend
complexity: high
dependencies:
  - task_04
---

# Task 05: Prompt provenance and ACP guardrails

## Overview

Introduce prompt-source metadata so the daemon can distinguish user turns from network-originated turns and enforce the stricter tool policy required by the tech spec. This task closes the security gap around terminal/file operations without coupling ACP handlers directly to network internals.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add prompt options that carry `TurnSource` through the session prompt pipeline and runtime session state
- MUST expose a dedicated network prompt path or equivalent surface that later delivery workers can call explicitly
- MUST enforce network-turn restrictions in ACP handlers, including `network_owned` terminal tracking and structural allowlisting for `agh network` subcommands
- MUST keep regular user turns and existing permission flows unchanged outside network-originated execution
</requirements>

## Subtasks
- [x] 5.1 Extend session prompt APIs to carry `TurnSource` and record current prompt provenance
- [x] 5.2 Add runtime helpers that mark terminals as network-owned when created by allowlisted control-plane commands
- [x] 5.3 Enforce file and terminal restrictions for network-originated turns in ACP handlers
- [x] 5.4 Add coverage for prompt provenance, allowlist enforcement, and ownership-aware terminal operations

## Implementation Details

This task should make ACP restrictions depend only on session/runtime metadata, not on direct imports from `internal/network`. Follow the allowlist and ownership model from the corrected tech spec.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Network-originated turn metadata, ACP restrictions, and delimiter sections
- `internal/session/interfaces.go` - Extend prompt-related interfaces cleanly
- `internal/session/manager_prompt.go` - Add prompt options and network prompt entry point
- `internal/session/manager_hooks.go` - Preserve hook semantics while carrying prompt provenance
- `internal/session/session.go` - Store current turn source and terminal ownership state
- `internal/acp/handlers.go` - Enforce network-turn write and terminal guardrails

### Dependent Files
- `internal/network/delivery.go` - Delivery workers will call the network prompt path introduced here
- `internal/network/manager.go` - Manager wiring will depend on turn-end and provenance surfaces
- `internal/daemon/hooks_bridge.go` - Hook bridge may need to observe the new prompt path semantics

### Related ADRs
- [ADR-003: CLI + Bundled Skill for Agent Network Communication](adrs/adr-003.md) - Outbound control-plane commands are issued through CLI access
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - Network runtime integrates with sessions via late-bound interfaces, not package coupling

## Deliverables
- Prompt provenance support in the session package
- ACP restrictions for network-originated turns and network-owned terminal semantics
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for network-turn guardrails **(REQUIRED)**

## Tests
- Unit tests:
- [x] User turns and network turns are distinguishable through session prompt metadata
- [x] Non-allowlisted terminal invocations are rejected for network-originated turns
- [x] Network-owned terminal checks gate output, wait, kill, and release operations correctly
- [x] Existing user-turn permission flows still behave exactly as before
- Integration tests:
- [x] A simulated network turn can invoke only allowlisted `agh network` commands and is blocked from arbitrary file writes or shell wrappers
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- ACP handlers can safely distinguish network-originated turns from normal user interaction
- Later delivery tasks can inject messages without weakening the daemon's local safety model
