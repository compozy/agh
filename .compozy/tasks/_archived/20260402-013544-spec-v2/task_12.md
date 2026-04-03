---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_05
    - task_06
    - task_11
---

# Task 12: Workgroup Management

## Overview
Implement the workgroup lifecycle management including creation, destruction, listing, topology view, recursive nesting, master requirement enforcement, safety limit checks, and snapshot-on-close behavior that preserves workgroup outcomes for subsequent phases.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST enforce workgroup state machine: create → active → closing → closed per docs/spec-v2/04-workgroups.md
- MUST enforce exactly one master per workgroup per docs/spec-v2/04-workgroups.md
- MUST block master removal while workgroup is active
- MUST enforce safety limits: max_workgroup_depth, max_agents_per_workgroup, max_total_agents per docs/spec-v2/04-workgroups.md
- MUST implement snapshot-on-close: query scoped state, generate summary, insert snapshot row, notify parent per docs/spec-v2/04-workgroups.md
- MUST support recursive workgroup nesting at any depth (up to limit)
- MUST count depth correctly (root=0)
- MUST count agents per workgroup (direct members only, not sub-workgroup agents)
- MUST return clear error messages matching spec examples on limit violations
</requirements>

## Subtasks
- [x] 12.1 Implement workgroup create with state machine (create → active on master spawn)
- [x] 12.2 Implement master requirement enforcement (exactly one, no removal while active)
- [x] 12.3 Implement safety limit checks (depth, agents per workgroup, total agents)
- [x] 12.4 Implement workgroup destroy with snapshot-on-close
- [x] 12.5 Implement recursive nesting with correct depth counting
- [x] 12.6 Implement workgroup list and topology tree view

## Implementation Details
Refer to docs/spec-v2/04-workgroups.md for all workgroup rules, state machine, limits, and snapshot behavior.

### Relevant Files
- `docs/spec-v2/04-workgroups.md` — complete workgroup spec
- `docs/spec-v2/02-kernel.md` — snapshot-on-close SQLite queries

### Dependent Files
- `internal/registry/workgroups.go` — workgroup registry
- `internal/state/` — SQLite queries for snapshot, state writes
- `internal/transport/` — NATS subjects for workgroup events

## Deliverables
- Workgroup lifecycle management (create, activate, destroy)
- Safety limit enforcement
- Snapshot-on-close logic
- Topology tree builder
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for workgroup lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Workgroup create starts in "create" state
  - [x] Workgroup transitions to "active" when master reaches ready
  - [x] Second master spawn rejected with error
  - [x] Master kill rejected while workgroup active
  - [x] Max agents per workgroup enforced with correct error message
  - [x] Max total agents enforced with correct error message
  - [x] Max workgroup depth enforced with correct error message
  - [x] Depth counting: root=0, child=1, grandchild=2
  - [x] Agent counting: only direct members, not sub-workgroup agents
- Integration tests:
  - [x] Full lifecycle: create → spawn master → spawn workers → destroy → snapshot
  - [x] Snapshot contains blackboard decisions and milestones from workgroup scope
  - [x] Parent master notified with snapshot summary on child destroy
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Error messages match docs/spec-v2/04-workgroups.md examples
- Snapshot-on-close preserves workgroup outcome data
