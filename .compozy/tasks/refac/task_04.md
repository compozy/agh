---
status: pending
title: Domain-level deduplication
type: refactor
complexity: medium
dependencies:
  - task_02
---

# Task 04: Domain-level deduplication

## Overview

Apply targeted method extractions and utility consolidation across domain packages: deduplicate the Create/Resume activation sequence in session manager, extract permission event emission in ACP handlers, add validation helpers in store, extract shared `fileSnapshot` type, consolidate JSON clone utilities, and introduce a generic `listBundle[T]` helper for CLI output. These are independent improvements that collectively reduce ~200 lines of duplication across 6 packages.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST extract `activateAndWatch` method in session manager to deduplicate Create (lines 332-343) and Resume (lines 497-510)
- MUST extract `emitPermissionEvent` helper in `acp/handlers.go` to deduplicate 3 identical event-emission blocks in `handleRequestPermission`
- MUST extract `requireField(value, label)` and `requirePositiveLimit(limit, label)` validation helpers in `store/`
- MUST extract `checkReady(ctx)` nil-guard helper on `GlobalDB` (replaces 18+ identical guard blocks)
- MUST consolidate `cloneRawMessage` (`session/transcript.go:595`) and `cloneRawJSON` (`acp/handlers.go:753`) — single canonical copy
- SHOULD extract shared `fileSnapshot` type from `skills/types.go:66-71` and `workspace/resolver.go:64-67` into `fileutil/` (with `Equal`, `Clone`, `FromPath`)
- SHOULD introduce generic `listBundle[T]` helper in `cli/format.go` to reduce output bundle boilerplate (~20 instances)
- MUST NOT change any external behavior
</requirements>

## Subtasks

- [ ] 4.1 Extract `activateAndWatch` in session manager + extract `emitPermissionEvent` in ACP handlers
- [ ] 4.2 Extract store validation helpers (`requireField`, `requirePositiveLimit`) and `checkReady(ctx)` on GlobalDB
- [ ] 4.3 Consolidate `cloneRawMessage`/`cloneRawJSON` into single utility
- [ ] 4.4 Extract shared `fileSnapshot` type from skills and workspace into `fileutil/`
- [ ] 4.5 Introduce generic `listBundle[T]` in CLI and update existing bundle functions
- [ ] 4.6 Run `make verify` to confirm all tests pass

## Implementation Details

See TechSpec "Phase 4: Domain-Level Deduplication" items 4.1–4.8. See individual reports for before/after sketches:
- [Core report](./20260406-core-session-acp.md) F2, F4, F5 — session/ACP extractions
- [Storage report](./20260406-storage-observe-memory.md) F2, F6 — store helpers
- [New report](./20260406-skills-workspace.md) F1 — shared fileSnapshot
- [Infra report](./20260406-config-daemon-cli.md) F6 — listBundle pattern

### Relevant Files

**Session/ACP:**
- `internal/session/manager.go:332-343` — Create activation sequence (duplicated)
- `internal/session/manager.go:497-510` — Resume activation sequence (duplicated)
- `internal/acp/handlers.go:228-324` — `handleRequestPermission` with 3 duplicated event emissions
- `internal/session/transcript.go:595-602` — `cloneRawMessage`
- `internal/acp/handlers.go:753-760` — `cloneRawJSON` (identical function)

**Store:**
- `internal/store/store.go:79-349` — 13 `Validate()` methods with repeated `strings.TrimSpace` pattern
- `internal/store/global_db.go` — 18+ methods with identical nil-receiver + nil-context guards

**Skills/Workspace:**
- `internal/skills/types.go:66-71` — `fileSnapshot` struct (with `path` field)
- `internal/workspace/resolver.go:64-67` — `fileSnapshot` struct (without `path` field)
- `internal/skills/registry.go:453-557` — `snapshotsEqual`, `cloneFileSnapshots`, `snapshotFile`
- `internal/workspace/resolver.go:844-880` — `snapshotPath`, `snapshotsEqual`, `cloneSnapshots`

**CLI:**
- `internal/cli/agent.go:60-90` — `agentListBundle` (pattern example)
- `internal/cli/memory.go:494-508` — `memoryListBundle` (pattern example)
- `internal/cli/session.go:403-419` — `sessionListBundle` (pattern example)

### Dependent Files

- `internal/session/manager.go` — Create and Resume call `activateAndWatch`
- `internal/acp/handlers.go` — `handleRequestPermission` calls `emitPermissionEvent`
- `internal/store/store.go` — Validate methods use helpers
- `internal/store/global_db.go` — all public methods call `checkReady`
- `internal/skills/registry.go` — imports shared `fileSnapshot` from fileutil
- `internal/workspace/resolver.go` — imports shared `fileSnapshot` from fileutil
- `internal/cli/*.go` — list bundle functions use `listBundle[T]`

## Deliverables

- Extracted `activateAndWatch` + `emitPermissionEvent` methods
- `requireField`, `requirePositiveLimit`, `checkReady` helpers in store
- Consolidated JSON clone utility
- Shared `fileSnapshot` type in `fileutil/`
- Generic `listBundle[T]` in CLI
- All existing tests pass **(REQUIRED)**
- `make verify` passes **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `activateAndWatch` correctly updates process, activates session, writes meta, starts watcher, notifies
  - [ ] `emitPermissionEvent` emits correct fields for each decision type (auto, interactive, timeout)
  - [ ] `requireField` returns error for empty/whitespace strings, nil for valid
  - [ ] `requirePositiveLimit` returns error for negative, nil for zero/positive
  - [ ] `checkReady` returns error for nil receiver and nil context
  - [ ] `fileutil.SnapshotEqual` correctly compares maps (equal, different sizes, different values)
  - [ ] `fileutil.CloneSnapshots` returns independent copy
  - [ ] `listBundle[T]` produces correct JSON, human table, and toon output
- Integration tests:
  - [ ] Session Create and Resume flows work through `activateAndWatch`
  - [ ] Permission handling tests pass through `emitPermissionEvent`
  - [ ] All store validation tests pass with new helpers
- Test coverage target: >=80%

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- Create/Resume activation exists in exactly one method
- Permission event emission exists in exactly one helper
- No duplicated `cloneRawMessage`/`cloneRawJSON`
- No duplicated `fileSnapshot` types across packages
- No duplicated nil-guard blocks in GlobalDB
- CLI bundle boilerplate reduced by ~40%
