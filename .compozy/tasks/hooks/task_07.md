---
status: completed
title: Migrate skills hook parsing to new declarations
type: refactor
complexity: medium
dependencies:
  - task_01
---

# Task 7: Migrate skills hook parsing to new declarations

## Overview

Migrate `internal/skills` from owning hook dispatch to supplying typed declarations to the new hooks platform. This is the hard cut-over for skills: delete `HookRunner`, old `HookDecl`/`HookEvent`/`HookPayload`/`HookResult`, rewrite the loader to parse the new schema (dotted event names, mode, priority, matcher), and update all tests.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST delete `internal/skills/hooks.go` entirely (HookRunner, RunHooks, runHook, orderSkillsForHooks, all helpers)
- MUST delete `internal/skills/hook_process_unix.go` and `hook_process_windows.go`
- MUST delete `internal/skills/hooks_test.go`
- MUST remove `HookDecl`, `HookEvent`, `HookPayload`, `HookResult` from `internal/skills/types.go`
- MUST add new `Hooks []hooks.HookDecl` field to `Skill` struct using types from `internal/hooks`
- MUST rewrite `parseHookDecls()` in `loader.go` to parse new schema: `event` (dotted), `command`, `args`, `timeout`, `env`, `mode`, `priority`, `matcher`
- MUST rewrite `validHookEvent()` to validate against `internal/hooks` event taxonomy
- MUST remove `cloneHookDecls()` from `registry.go` and update `cloneSkill()`
- MUST update all test files that reference old hook types
- MUST update testdata YAML files with new event names (`on_session_created` → `session.post_create`)
</requirements>

## Subtasks
- [x] 7.1 Delete `hooks.go`, `hook_process_unix.go`, `hook_process_windows.go`, `hooks_test.go`
- [x] 7.2 Remove old hook types from `types.go`, add new `Hooks` field using `hooks.HookDecl`
- [x] 7.3 Rewrite `parseHookDecls()` and `validHookEvent()` in `loader.go`
- [x] 7.4 Remove `cloneHookDecls()` from `registry.go`, update `cloneSkill()`
- [x] 7.5 Update testdata YAML and all affected tests
- [x] 7.6 Write new unit tests for the rewritten hook parsing

## Implementation Details

Modify existing files in `internal/skills/`:
- Delete: `hooks.go`, `hook_process_unix.go`, `hook_process_windows.go`, `hooks_test.go`
- Modify: `types.go` (remove old types, add hooks.HookDecl reference), `loader.go` (rewrite parsing), `registry.go` (remove cloneHookDecls)
- Update testdata: `testdata/loader/hooks-only/SKILL.md`, `testdata/loader/invalid-hook-command/SKILL.md`, `testdata/loader/combined/SKILL.md`

Reference TechSpec "Migration from Current Hooks Implementation" section for exact mapping.

### Relevant Files
- `internal/skills/types.go:55-70` — Old HookDecl, HookEvent to delete
- `internal/skills/hooks.go` — Entire file to delete
- `internal/skills/loader.go:290-343` — parseHookDecls, validHookEvent to rewrite
- `internal/skills/registry.go:762-779` — cloneHookDecls to delete
- `internal/skills/registry.go:704` — cloneSkill hook clone call to update
- `internal/skills/hooks_test.go` — Entire file to delete
- `internal/skills/testdata/loader/hooks-only/SKILL.md` — Update event name
- `internal/hooks/types.go` (task_01) — New HookDecl type to import

### Dependent Files
- `internal/daemon/notifier.go` — skillsHookDispatcher references HookRunner (deleted in task_09)
- `internal/daemon/boot.go` — hookRunner creation (deleted in task_09)

### Related ADRs
- [ADR-002: Use a Dotted Hook Taxonomy](../adrs/adr-002.md) — New event names

## Deliverables
- Deleted files: `hooks.go`, `hook_process_*.go`, `hooks_test.go`
- Modified files: `types.go`, `loader.go`, `registry.go`
- Updated testdata files
- New unit tests for rewritten parsing with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Parse YAML with `event: session.post_create` succeeds
  - [x] Parse YAML with old `event: on_session_created` fails with descriptive error mentioning new name
  - [x] Parse YAML with unknown event `event: foo.bar` fails validation
  - [x] Parse YAML with new optional fields: mode, priority, matcher
  - [x] Parse YAML with minimal fields (just event + command) uses defaults
  - [x] `cloneSkill()` correctly deep-copies the new Hooks field
  - [x] Skill struct `Hooks` field is `[]hooks.HookDecl` type
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes — no references to deleted types
- `make build` passes — no broken imports
- Zero references to `HookRunner`, old `HookDecl`, `HookPayload` in `internal/skills/`
