---
status: pending
title: Autonomy Hook Taxonomy And Task Hook Bridge
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Autonomy Hook Taxonomy And Task Hook Bridge

## Overview
Add the typed hook surface for coordinator lifecycle, task-run ownership, and spawn lifecycle before autonomy behavior depends on extension points. This task also introduces the narrow task-domain hook bridge so audit events and hooks stay distinct but can be co-emitted at the authoritative transition sites.

<critical>
- ALWAYS READ `_techspec.md` and ADR-009 before changing hooks
- REFERENCE TECHSPEC for event names and mutability rules
- FOCUS ON "WHAT" - typed events, payloads, patches, introspection, and bridge contracts
- MINIMIZE CODE - hooks must not become a generic event bus or scheduler authority
- TESTS REQUIRED - event taxonomy, dispatch, introspection, and safety guards are mandatory
- NO WORKAROUNDS - do not tail task event tables to fake pre-commit hooks
</critical>

<requirements>
- MUST add `coordinator.*`, `task.run.*`, and `spawn.*` hook events with families, sync eligibility, payloads, patches, dispatch methods, matchers, and introspection descriptors.
- MUST keep scheduler wake/no-match/recovery as metrics/logs/observability, not hooks, for the MVP.
- MUST add a narrow dispatcher interface consumed by `internal/task` and implemented in daemon wiring.
- MUST dispatch pre-claim/pre-spawn hooks at the call site before transactional state changes and post hooks after committed state/audit events.
- MUST prevent hook patches from widening permissions, broadening claim criteria, mutating committed claim state, or bypassing `ClaimNextRun`.
- MUST preserve existing hook declaration providers: config, agent, skill, extension, and hook binding resources.
</requirements>

## Subtasks
- [ ] 3.1 Add autonomy hook events/families and update event catalog/introspection.
- [ ] 3.2 Add payloads, patches, matchers, patch guards, and dispatch methods.
- [ ] 3.3 Add a task-domain hook dispatcher interface plus no-op default implementation.
- [ ] 3.4 Wire daemon hook bridge adapters without creating package import cycles.
- [ ] 3.5 Add tests for taxonomy validation, hook binding resources, deny/narrow behavior, and forbidden patch widening.
- [ ] 3.6 Confirm generated contracts/web/docs impact and defer documentation details to task_16 if no DTO changes occur.

## Implementation Details
Extend the existing `internal/hooks` architecture rather than adding autonomy-specific callback registries. The task service should depend only on a narrow interface, not on `daemon` or the hook runtime implementation.

### Relevant Files
- `internal/hooks/events.go` - event taxonomy and family validation.
- `internal/hooks/payloads.go` - typed payload and patch structs.
- `internal/hooks/dispatch.go` - hook dispatch methods and guards.
- `internal/hooks/introspection.go` - hook catalog descriptors.
- `internal/hooks/matchers.go` - matching behavior for new payloads.
- `internal/daemon/hooks_bridge.go` - daemon hook adapter pattern.
- `internal/daemon/hook_binding_resources.go` - resource-backed hook declarations.
- `internal/task/manager.go` - task run transitions that later co-emit hooks.
- `.resources/hermes/agent/shell_hooks.py` - reference for hook invocation around agent actions.
- `.resources/paperclip/.agents/skills/prcheckloop/SKILL.md` - reference for explicit review/loop extension points.
- `.resources/openclaw/.agents/skills/openclaw-qa-testing/SKILL.md` - reference for QA-hookable execution discipline.

### Dependent Files
- `internal/session/hooks.go` - session hook patterns to avoid duplicating dispatch concepts.
- `internal/api/contract/*` - hook event catalog payloads if exposed to API.
- `packages/site/content/runtime/core/hooks/*` - task_16 documents new hook taxonomy.

### Related ADRs
- [ADR-004: Split Semantic Coordination from Mechanical Scheduling](adrs/adr-004.md) - hooks observe/shape, scheduler owns mechanics.
- [ADR-009: Autonomy Hooks and Extension Points Are First-Class Contracts](adrs/adr-009.md) - first-class extension surface and safety boundaries.
- [ADR-011: Generated Contracts and Documentation Co-Ship with Autonomy MVP Steps](adrs/adr-011.md) - contract/docs parity when hook catalog surfaces change.

## Deliverables
- Typed autonomy hook events, payloads, patches, dispatch, and introspection descriptors.
- Task-domain hook dispatcher interface with daemon adapter and no-op test path.
- Hook safety tests covering deny/narrow and no-widen invariants.
- Unit tests with 80%+ coverage for touched hook code **(REQUIRED)**.
- Integration tests for hook binding resources receiving autonomy payloads **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Hook catalog lists every new `coordinator.*`, `task.run.*`, and `spawn.*` event with the correct family and sync eligibility.
  - [ ] `task.run.pre_claim` can deny or narrow criteria only in the allowed directions.
  - [ ] Spawn pre-create patch rejects permission widening and unknown child atoms after hook mutation.
  - [ ] Scheduler wake/no-match names are absent from hook taxonomy.
  - [ ] No-op task hook dispatcher preserves current task behavior.
- Integration tests:
  - [ ] Hook binding resource registers one autonomy hook and receives a typed payload through the daemon bridge.
  - [ ] Post-commit task-run hook dispatch occurs after the corresponding audit event write.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Autonomy has typed extension points before behavior lands.
- Hooks cannot bypass claim/lease/spawn safety invariants.
