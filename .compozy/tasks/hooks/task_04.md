---
status: pending
title: Generic pipeline with sync composition and guards
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 4: Generic pipeline with sync composition and guards

## Overview

Implement the core `pipeline[P, R]` generic type that executes sync hooks as a sequential pipeline (each hook sees the output of the previous) and includes the dispatch depth guard and permission deny-only invariant. This is the most complex component — it ties together ordering, executors, and typed patch composition into a single execution engine.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `pipeline[P, R]` as a package-private generic type with `apply`, `encode`, `decode` functions
- MUST execute sync hooks sequentially — hook N receives the payload patched by hook N-1
- MUST short-circuit on explicit deny from any hook
- MUST short-circuit on `required` hook failure (error or timeout) with error
- MUST skip non-required hook failures and continue pipeline
- MUST implement dispatch depth guard via `context.WithValue` counter, max depth 3
- MUST enforce permission deny-only invariant: reject any patch that attempts deny→allow, log as `hook.dispatch.permission_escalation_blocked`
- MUST provide `encode`/`decode` functions for subprocess executor serialization boundary
- MUST use native executor path (no serialization) for Go callback hooks
</requirements>

## Subtasks
- [ ] 4.1 Implement `pipeline[P, R]` generic struct with `execute(ctx, payload) (P, error)` method
- [ ] 4.2 Implement sequential sync hook composition with patch application loop
- [ ] 4.3 Implement pipeline short-circuit on deny and required-hook failure
- [ ] 4.4 Implement dispatch depth guard with context counter (max 3)
- [ ] 4.5 Implement permission deny-only invariant check
- [ ] 4.6 Implement encode/decode bridge for subprocess executors vs native bypass

## Implementation Details

Create new files in `internal/hooks/`:
- `pipeline.go` — Generic pipeline type, sequential execution, short-circuit logic
- `depth.go` — Dispatch depth context key and guard functions
- `permission.go` — Permission invariant check

Reference TechSpec "Core Interfaces" section for `pipeline[P, R]` design. Reference ADR-006 for sequential composition, ADR-009 for permission invariant, ADR-012 for depth guard.

### Relevant Files
- `internal/hooks/ordering.go` (task_02) — Provides sorted hook list for pipeline execution
- `internal/hooks/executor.go` (task_03) — Executor interface called by pipeline
- `internal/hooks/executor_native.go` (task_03) — Native executor bypasses serialization
- `internal/hooks/executor_subprocess.go` (task_03) — Subprocess executor uses encode/decode
- `internal/hooks/events.go` (task_01) — HookEvent for depth guard context key

### Dependent Files
- `internal/hooks/` — Hooks struct (task_06) wraps pipelines for each event

### Related ADRs
- [ADR-005: Use Typed Per-Event Dispatch Functions](../adrs/adr-005.md) — Pipeline is the internal engine behind typed dispatch
- [ADR-006: Sequential Pipeline for Sync Hook Patch Composition](../adrs/adr-006.md) — Defines sequential composition model
- [ADR-007: Use Go Generics for Internal Dispatcher Type Safety](../adrs/adr-007.md) — Defines generic pipeline approach
- [ADR-009: Permission Hooks Are Deny-Only](../adrs/adr-009.md) — Permission invariant enforcement
- [ADR-012: Classify Events into Sync-Eligible and Async-Only](../adrs/adr-012.md) — Dispatch depth guard

## Deliverables
- `internal/hooks/pipeline.go` with generic pipeline implementation
- `internal/hooks/depth.go` with depth guard
- `internal/hooks/permission.go` with deny-only invariant
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Pipeline with 3 sync hooks — hook 2 sees payload patched by hook 1, hook 3 sees payload patched by hook 2
  - [ ] Pipeline with explicit deny from hook 2 — hook 3 never executes, deny returned
  - [ ] Pipeline with required hook timeout — pipeline returns error, subsequent hooks skipped
  - [ ] Pipeline with non-required hook failure — hook skipped, pipeline continues
  - [ ] Pipeline with no matching hooks — returns original payload unchanged
  - [ ] Depth guard: dispatch at depth 1 succeeds
  - [ ] Depth guard: dispatch at depth 3 succeeds (at limit)
  - [ ] Depth guard: dispatch at depth 4 returns error immediately
  - [ ] Depth guard: nested dispatch increments depth from parent context
  - [ ] Permission invariant: patch that keeps deny returns deny (allowed)
  - [ ] Permission invariant: patch that changes deny→allow is rejected and logged
  - [ ] Permission invariant: patch that changes allow→deny is allowed (deny escalation ok)
  - [ ] Native executor path skips serialization — callback receives typed payload directly
  - [ ] Subprocess executor path uses encode/decode for JSON serialization
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Sequential composition is deterministic — same hooks + same input always produces same output
- Permission escalation is architecturally impossible via pipeline enforcement
- Depth guard prevents stack overflow from circular dispatch
