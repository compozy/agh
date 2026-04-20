---
status: pending
title: Preserve Recipe Operational Semantics Under Capability
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 03: Preserve Recipe Operational Semantics Under Capability

## Overview

Carry over the useful runtime behavior that `recipe` already had so unification does not regress delivery, interaction, or lifecycle semantics. This task rewires router, delivery, and lifecycle behavior around `kind:"capability"` while keeping the protocol surface conceptually simpler and operationally unchanged where it matters.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Impact Analysis", "Testing Approach", and "Build Order"
- PRESERVE RECIPE'S USEFUL BEHAVIOR WITHOUT PRESERVING RECIPE ITSELF - this task is about operational continuity under a new single concept
- KEEP ROUTER, DELIVERY, AND LIFECYCLE IN SYNC - partial renames across only one subsystem are not acceptable
- TESTS REQUIRED - broadcast, directed delivery, interaction opening, and terminal lifecycle behavior all need explicit coverage
- GREENFIELD: remove stale branches and dead code paths instead of deferring cleanup to follow-up work
</critical>

<requirements>
- MUST make `kind:"capability"` support the same delivery modes that `recipe` currently supports: broadcast and directed
- MUST preserve interaction-opening behavior and lifecycle participation for transferred capabilities
- MUST update router dispatch, delivery summaries, and lifecycle bookkeeping so they no longer refer to recipe terminology
- MUST keep `direct` semantics separate from `capability` semantics rather than tunneling transfer back through conversation flows
- MUST retain or improve observable logs and audit trails for artifact transfer behavior under the new kind
- SHOULD leave router and lifecycle tests proving interoperability between `direct` and `capability` when both appear in the same interaction flow
</requirements>

## Subtasks
- [ ] 3.1 Update router dispatch so `kind:"capability"` follows the supported artifact-transfer paths
- [ ] 3.2 Rewrite delivery bookkeeping and summaries to treat capabilities as the transferred artifact
- [ ] 3.3 Replace lifecycle creation and terminal-state handling that still references recipe interactions
- [ ] 3.4 Remove obsolete recipe-specific runtime branches and dead helper logic
- [ ] 3.5 Add regression coverage for broadcast, directed, and lifecycle flows under the new kind

## Implementation Details

See TechSpec "Impact Analysis", "Testing Approach", and "Build Order" item 5. The central invariant is that a unified capability transfer remains a dedicated artifact flow with the same operational reach as the former recipe path, while `direct` remains the conversational/message primitive.

### Relevant Files
- `internal/network/router.go` - dispatch layer that decides how transferred envelopes are handled
- `internal/network/delivery.go` - delivery summaries, fan-out behavior, and directed/broadcast transfer handling
- `internal/network/lifecycle.go` - interaction lifecycle rules that must inherit current recipe semantics under the new kind
- `internal/network/router_test.go` - router-level behavioral regressions for new kind dispatch
- `internal/network/delivery_test.go` - delivery bookkeeping and artifact transfer assertions
- `internal/network/lifecycle_test.go` - interaction opening, progression, and terminal-state coverage
- `internal/network/delivery_integration_test.go` - real delivery behavior across channel and peer targets

### Dependent Files
- `internal/network/router_integration_test.go` - end-to-end interaction and delivery coverage will need the new operational behavior
- `internal/network/tasks.go` - any task or interaction metadata consuming envelope kinds may need terminology updates
- `internal/network/tasks_integration_test.go` - interaction flows that coexist with transferred capabilities may need regression coverage
- `internal/network/audit.go` - audit/event output should describe capability transfer instead of recipe transfer
- `internal/network/manager.go` - manager-level orchestration may consume updated router and lifecycle behaviors

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - requires the runtime to stop modeling recipe as a separate object
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - defines the exact behavioral carry-over this task must preserve

## Deliverables
- Router, delivery, and lifecycle behavior updated for `kind:"capability"`
- Removal of recipe-specific operational branches in the runtime network flow
- Regression tests for broadcast and directed capability transfer plus interaction lifecycle handling **(REQUIRED)**
- Integration coverage proving capability transfer still opens and completes interactions correctly **(REQUIRED)**
- Test coverage >=80% for the touched network packages **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Router dispatch sends `kind:"capability"` envelopes through the same operational branch that formerly handled recipe transfer
  - [ ] Delivery summaries and audit metadata label transferred artifacts as capabilities rather than recipes
  - [ ] Lifecycle helpers open interactions for directed capability transfers and enforce terminal rules correctly
  - [ ] Mixed `direct` and `capability` flows do not collapse into a single overloaded runtime path
- Integration tests:
  - [ ] Broadcast capability transfer reaches the expected channel peers and records the right delivery state
  - [ ] Directed capability transfer opens an interaction and progresses through lifecycle updates correctly
  - [ ] Terminal lifecycle handling for capabilities matches the prior recipe behavior without stale recipe references
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Capability transfer preserves the useful operational behavior recipe had before unification
- Runtime network code no longer treats recipe as a first-class lifecycle or delivery concept
