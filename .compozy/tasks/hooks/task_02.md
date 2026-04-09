---
status: pending
title: Declaration normalization, matchers, and ordering
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 2: Declaration normalization, matchers, and ordering

## Overview

Implement the declaration normalization pipeline that converts raw `HookDecl` from any source into `ResolvedHook`, the matcher evaluation system for event-specific filtering, and the deterministic ordering function (source → priority → name). This is the resolution engine that the dispatcher depends on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST normalize `HookDecl` into `ResolvedHook` with validated source, mode, matcher, timeout, and executor binding
- MUST reject declarations with `mode: sync` on async-only events at normalization time
- MUST reject declarations with `required: true` and `mode: async`
- MUST implement per-family matcher evaluation: session (type, workspace, agent), tool (name, namespace, read-only), permission (tool, decision class), message (role, delta type), context (reason, strategy)
- MUST implement ordering: source class (Native→Config→AgentDef→Skill) → priority (desc) → name (asc lexicographic)
- MUST apply default priorities per source: native=1000, config=500, agent-definition=100, skill=0
- MUST preserve skill sub-ordering (Bundled→Marketplace→User→Additional→Workspace) before name
</requirements>

## Subtasks
- [ ] 2.1 Implement `HookDecl` → `ResolvedHook` normalization with validation
- [ ] 2.2 Implement matcher types and evaluation functions per event family
- [ ] 2.3 Implement deterministic ordering function with three-level sort
- [ ] 2.4 Implement skill sub-ordering using existing `SkillSource` precedence
- [ ] 2.5 Write unit tests for normalization, matchers, and ordering

## Implementation Details

Create new files in `internal/hooks/`:
- `normalize.go` — Declaration normalization and validation
- `matcher.go` — Matcher types and evaluation per event family
- `ordering.go` — Deterministic sort function

Reference TechSpec "Matcher Model" and "Dispatch Model" sections. Reference ADR-011 for ordering rules.

### Relevant Files
- `internal/hooks/types.go` (task_01) — ResolvedHook, HookDecl, HookSource types
- `internal/hooks/events.go` (task_01) — Sync eligibility for validation
- `internal/skills/hooks.go:220-252` — Current `orderSkillsForHooks` for skill sub-ordering pattern
- `internal/skills/types.go:30-37` — SkillSource enum and ordering

### Dependent Files
- `internal/hooks/` — Pipeline (task_04) and registry (task_06) depend on these functions

### Related ADRs
- [ADR-004: Support Four Declaration Sources with Ordered Dispatch](../adrs/adr-004.md) — Source ordering rules
- [ADR-011: Simplify Ordering to Source, Priority, Name](../adrs/adr-011.md) — Removes specificity, defines ordering
- [ADR-012: Classify Events into Sync-Eligible and Async-Only](../adrs/adr-012.md) — Validation of sync mode vs event eligibility

## Deliverables
- `internal/hooks/normalize.go` with normalization and validation
- `internal/hooks/matcher.go` with per-family matcher evaluation
- `internal/hooks/ordering.go` with deterministic sort
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Normalization rejects `mode: sync` on `message.delta` with clear error
  - [ ] Normalization rejects `required: true` on async hook
  - [ ] Normalization applies default priority 1000 for Native source when priority unset
  - [ ] Normalization applies default priority 0 for Skill source
  - [ ] Session matcher matches on workspace ID and agent name, rejects non-matching
  - [ ] Tool matcher matches on tool name and namespace, handles wildcard
  - [ ] Permission matcher matches on tool name and decision class
  - [ ] Ordering sorts Native before Config before AgentDef before Skill
  - [ ] Ordering sorts higher priority first within same source
  - [ ] Ordering sorts names ascending for equal source and priority
  - [ ] Skill sub-ordering: Bundled before Marketplace before User before Workspace
  - [ ] Ordering is stable across multiple sorts with same input
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Invalid declarations fail with descriptive error messages
- Ordering is fully deterministic — same input always produces same output
