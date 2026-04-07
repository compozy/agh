---
status: completed
title: Utility packages + inline quick wins
type: refactor
complexity: high
dependencies: []
---

# Task 01: Utility packages + inline quick wins

## Overview

Create three shared utility packages (`procutil`, `fileutil`, `testutil`) to eliminate cross-package duplication, then apply a batch of small inline deduplication fixes across the codebase. This is the foundation step for the larger refactor set, but it still spans multiple packages and requires careful behavior-preserving edits before the file splits in task 02.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/procutil/` with `Alive(pid int) bool` and `Signal(pid int, sig syscall.Signal) error`, replacing the duplicated process utility implementations across daemon, cli, and memory
- MUST create `internal/fileutil/` with `AtomicWriteFile(path string, content []byte, perm os.FileMode) error`, replacing both atomic-write implementations. The shared helper MUST call `Sync` before rename so it preserves the current `store/meta.go` durability behavior and fixes the `memory/store.go` variant
- MUST create `internal/testutil/` with `Context(t) context.Context` and `EqualStringSlices(a, b []string) bool`, replacing the duplicated `testContext` and string-slice comparison helpers across the test suite
- MUST add an exported helper in `config/home.go` for resolving the user agents/skills directory and update both daemon and CLI consumers to use it
- MUST consolidate daemon path normalization onto a shared `config/home.go` path-resolution helper; do not couple daemon code to unexported config internals
- MUST merge `cleanupFailedCreate`/`cleanupFailedResume` into `cleanupFailedStart(sessionDir, ...)`
- MUST extract `processSkill` method in skills registry (3x duplicated load-verify-overlay loop)
- MUST replace `reflect.DeepEqual` in `skills/registry.go:201` with snapshot-based comparison
- MUST merge `startingDaemonStatus`/`stoppedDaemonStatus` into parameterized function
- MUST fix typo `defaultReadHeaderTimout` in `udsapi/server.go:29`
- MUST remove custom `max()` in `cli/format.go:279` (use Go builtin)
- New utility packages MUST have >95% test coverage
- MUST NOT change any external behavior
</requirements>

## Subtasks

- [x] 1.1 Create `internal/procutil/` and update consumers (`daemon/lock.go:195`, `daemon/daemon.go:1390`, `cli/root.go:247-258`, `memory/lock.go:274`)
- [x] 1.2 Create `internal/fileutil/` and update consumers (`store/meta.go:36-79`, `memory/store.go:489`)
- [x] 1.3 Create `internal/testutil/` and update the duplicated `testContext` helpers plus string-slice comparison helpers in the test suite
- [x] 1.4 Consolidate config path utilities (`config/home.go:138`, `daemon/daemon.go:882,1338`, `cli/skill.go:348`)
- [x] 1.5 Merge session cleanup functions (`session/manager.go:964-1005`)
- [x] 1.6 Extract `processSkill` in skills registry + replace `reflect.DeepEqual` (`skills/registry.go:201,228-328`)
- [x] 1.7 CLI/UDS misc fixes (`cli/daemon.go:296-322`, `cli/format.go:279`, `udsapi/server.go:29`)

## Implementation Details

See TechSpec "Phase 1: Quick Wins" items 1.1–1.10 and "Core Interfaces" section for function signatures.

### Relevant Files

**procutil sources:**
- `internal/daemon/lock.go:195` — `processAlive` (canonical implementation)
- `internal/daemon/daemon.go:1390` — `signalProcess`
- `internal/memory/lock.go:274` — `processAlive` duplicate
- `internal/cli/root.go:247-258` — `signalProcess` + `processAlive` duplicates

**fileutil sources:**
- `internal/store/meta.go:36-79` — inline atomic write (current durability baseline with `Sync`)
- `internal/memory/store.go:489` — `atomicWriteFile` duplicate (missing `Sync` before rename)

**testutil sources:**
- `internal/acp/client_test.go:778` — `testContext`
- `internal/cli/helpers_test.go:274` — `testContext`
- `internal/daemon/daemon_test.go:1591` — `testContext`
- `internal/memory/dream_test.go:775` — `testContext`
- `internal/observe/observer_test.go:488` — `testContext`
- `internal/session/manager_test.go:993` — `testContext`
- `internal/store/session_db_test.go:322` — `testContext`
- `internal/daemon/daemon_test.go:1775` — `equalStrings`
- `internal/observe/reconcile_test.go:201` — `equalStrings`
- `internal/store/session_db_test.go:417` — `equalStringSlices`

**Inline dedup sources:**
- `internal/config/home.go:138` — `expandUserPath` (reuse target)
- `internal/daemon/daemon.go:882` — `userAgentsSkillsDir` (remove)
- `internal/daemon/daemon.go:1338` — `normalizeAbsolutePath` (remove)
- `internal/cli/skill.go:348` — `cliUserAgentsSkillsDir` (remove)
- `internal/session/manager.go:964` — `cleanupFailedCreate` (merge)
- `internal/session/manager.go:988` — `cleanupFailedResume` (merge)
- `internal/skills/registry.go:201` — `reflect.DeepEqual` (replace)
- `internal/skills/registry.go:228-328` — 3 duplicated load loops (extract)
- `internal/cli/daemon.go:296-322` — two near-identical status functions (merge)
- `internal/udsapi/server.go:29` — typo `defaultReadHeaderTimout`
- `internal/cli/format.go:279` — custom `max()` shadowing builtin

### Dependent Files

- `internal/daemon/lock.go` — imports `procutil`
- `internal/daemon/daemon.go` — imports `procutil` and the shared config path helper, removes local path utilities
- `internal/memory/lock.go` — imports `procutil`
- `internal/cli/root.go` — imports `procutil`
- `internal/store/meta.go` — imports `fileutil`
- `internal/memory/store.go` — imports `fileutil`
- `internal/config/home.go` — gains exported path helpers for daemon/CLI reuse
- `internal/cli/skill.go` — imports config for user agents/skills directory resolution
- `internal/skills/registry.go` — `reflect` import removed

## Deliverables

- `internal/procutil/procutil.go` + `procutil_test.go`
- `internal/fileutil/atomic.go` + `atomic_test.go`
- `internal/testutil/testutil.go`
- Updated imports in all consumer files
- All inline dedup changes applied
- Unit tests with >95% coverage for new packages **(REQUIRED)**
- `make verify` passes **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `procutil.Alive` with current PID returns true
  - [x] `procutil.Alive` with PID 0 and negative PID returns false
  - [x] `procutil.Signal` with valid PID and signal 0 succeeds
  - [x] `fileutil.AtomicWriteFile` writes correct content and permissions
  - [x] `fileutil.AtomicWriteFile` does not corrupt target on write failure
  - [x] `testutil.Context` returns a context cancelled after cleanup
  - [x] `testutil.EqualStringSlices` correctness for equal and unequal inputs
  - [x] `config.ResolveUserAgentsSkillsDir` with HOME set and unset
  - [x] `cleanupFailedStart` with and without sessionDir
  - [x] `processSkill` applies disabled, verifies, overlays; skips critical warnings
  - [x] Skills reload with unchanged snapshots skips map update
- Test coverage target: >=95% for procutil/fileutil, >=80% for modified packages
- All existing tests must pass unchanged

## Success Criteria

- All tests passing
- `make verify` passes
- Zero local copies of `processAlive`, `signalProcess`, `atomicWriteFile`, `testContext`, and the ad-hoc string-slice comparison helpers remain where the shared utilities should apply
- `reflect` import removed from `skills/registry.go`
- No duplicate path resolution or cleanup functions remain
