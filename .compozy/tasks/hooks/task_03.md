---
status: pending
title: Executor contracts and implementations
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 3: Executor contracts and implementations

## Overview

Implement the `Executor` interface and two concrete executors: a native Go callback executor (for in-process hooks) and a subprocess executor (for skill/config/agent shell hooks). The subprocess executor replaces the current `HookRunner.runHook()` in `internal/skills/hooks.go`, reusing its proven patterns for timeout, signal handling, environment allowlisting, and output capture.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define `Executor` interface with `Kind() HookExecutorKind` and `Execute(ctx, RegisteredHook, []byte) ([]byte, error)`
- MUST implement `NativeExecutor` that calls Go callbacks directly, bypassing serialization
- MUST implement `SubprocessExecutor` that runs shell commands with JSON payload via stdin, captures stdout/stderr
- MUST reuse environment allowlist pattern from existing `internal/skills/hooks.go:292-315`
- MUST enforce timeout via `context.WithTimeout` with graceful shutdown (signal, wait, kill)
- MUST capture stdout/stderr with existing 8KB limit pattern
- SHOULD leave a Wasm executor seam (empty `WasmExecutor` struct implementing the interface with `ErrNotImplemented`)
</requirements>

## Subtasks
- [ ] 3.1 Define `Executor` interface and `HookExecutorKind` enum
- [ ] 3.2 Implement `NativeExecutor` for Go callback hooks
- [ ] 3.3 Implement `SubprocessExecutor` with timeout, signal handling, env allowlist, capture
- [ ] 3.4 Add Wasm executor stub returning `ErrNotImplemented`
- [ ] 3.5 Write unit tests for both executors including timeout and error paths

## Implementation Details

Create new files in `internal/hooks/`:
- `executor.go` — Executor interface and kind enum
- `executor_native.go` — Native Go callback executor
- `executor_subprocess.go` — Subprocess executor (port from skills/hooks.go)
- `executor_subprocess_unix.go` / `executor_subprocess_windows.go` — Platform-specific process management (port from skills/hook_process_*.go)
- `executor_wasm.go` — Stub

Reference existing `internal/skills/hooks.go` lines 127-204 for subprocess execution pattern and lines 292-315 for environment allowlist.

### Relevant Files
- `internal/skills/hooks.go:127-204` — Current `runHook()` subprocess execution to port
- `internal/skills/hooks.go:292-315` — Environment allowlist (`hookAllowedEnvVars`)
- `internal/skills/hooks.go:339-401` — `hookCapture` output limiting pattern
- `internal/skills/hook_process_unix.go` — Unix-specific process group/signal handling
- `internal/skills/hook_process_windows.go` — Windows-specific process handling

### Dependent Files
- `internal/hooks/` — Pipeline (task_04) calls executors

### Related ADRs
- [ADR-005: Use Typed Per-Event Dispatch Functions](../adrs/adr-005.md) — Executor uses `[]byte` at serialization boundary

## Deliverables
- `internal/hooks/executor.go` with Executor interface
- `internal/hooks/executor_native.go` with NativeExecutor
- `internal/hooks/executor_subprocess.go` with SubprocessExecutor
- Platform-specific process management files
- `internal/hooks/executor_wasm.go` stub
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] NativeExecutor calls Go callback with correct payload and returns result
  - [ ] NativeExecutor returns error when callback panics (recovered)
  - [ ] SubprocessExecutor runs `echo` command and captures stdout as result
  - [ ] SubprocessExecutor passes JSON payload via stdin
  - [ ] SubprocessExecutor enforces timeout — command exceeding timeout is killed
  - [ ] SubprocessExecutor graceful shutdown — SIGTERM sent before SIGKILL
  - [ ] SubprocessExecutor filters environment to allowlist only
  - [ ] SubprocessExecutor captures stderr on non-zero exit
  - [ ] SubprocessExecutor respects 8KB stdout/stderr capture limit
  - [ ] WasmExecutor returns ErrNotImplemented
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Subprocess executor handles timeout gracefully without zombie processes
- Environment allowlist prevents ambient secret leakage
