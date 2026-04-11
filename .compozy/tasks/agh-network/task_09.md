---
status: pending
title: Bundled agh-network skill and prompt injection
type: backend
complexity: medium
dependencies:
  - task_04
  - task_08
---

# Task 09: Bundled agh-network skill and prompt injection

## Overview

Add the bundled `agh-network` skill content and wire it into session startup and resume flows for space-participating sessions. This task gives agents the prompt-side guidance they need to use the new CLI control plane safely and consistently.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add a bundled `agh-network` skill asset with CLI usage examples, retry guidance, wrapper expectations, and prompt-injection defense instructions
- MUST append the bundled skill content after prompt assembly and before ACP start when a session opts into a network space or resumes with persisted network metadata
- MUST avoid injecting the skill for sessions that are not participating in a space
- MUST ensure the bundled guidance stays aligned with the actual CLI and API surface introduced by task 08
</requirements>

## Subtasks
- [ ] 9.1 Add the bundled `agh-network` skill asset under the bundled skills tree
- [ ] 9.2 Wire space-aware skill injection into session start and resume flows
- [ ] 9.3 Validate that injected content references only supported CLI commands and safe wrapper semantics
- [ ] 9.4 Add tests for skill discovery, injection timing, and resume behavior

## Implementation Details

Keep the skill content grounded in the corrected tech spec and the real CLI surface. Injection belongs in the session startup flow, but only for sessions that explicitly opted into network participation.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Bundled skill, delimiter, and session integration sections
- `internal/skills/bundled/skills/agh-network/SKILL.md` - New bundled skill content
- `internal/skills/bundled/embed.go` - Ensure the new bundled asset is embedded and discoverable
- `internal/session/manager_start.go` - Append skill content after prompt assembly and before ACP startup
- `internal/session/manager.go` - Space-aware startup inputs flow from here
- `internal/cli/network.go` - Skill examples must match the CLI surface exactly

### Dependent Files
- `internal/network/delivery.go` - Inbound wrapper instructions referenced by the bundled skill must stay aligned with runtime delivery format
- `internal/session/manager_integration_test.go` - Startup and resume integration flows should prove injection behavior

### Related ADRs
- [ADR-003: CLI + Bundled Skill for Agent Network Communication](adrs/adr-003.md) - This task implements the chosen outbound guidance model
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Skill injection is gated by explicit session opt-in

## Deliverables
- New bundled `agh-network` skill asset and embedding coverage
- Session-start and resume wiring for conditional skill injection
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for skill injection and resume behavior **(REQUIRED)**

## Tests
- Unit tests:
- [ ] Bundled skill registry can load `agh-network` successfully
- [ ] Sessions without `Space` do not receive network skill content
- [ ] Sessions with `Space` receive the network skill content exactly once per startup or resume
- [ ] Skill examples stay consistent with supported CLI command names and flags
- Integration tests:
- [ ] A resumed space-participating session receives the bundled skill guidance again before ACP start
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Network-participating sessions consistently receive bundled guidance at startup and resume time
- Bundled instructions match the real CLI and wrapper behavior implemented in the runtime
