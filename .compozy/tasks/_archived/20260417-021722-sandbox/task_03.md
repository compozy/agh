---
status: completed
title: "Local provider implementation"
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 03: Local provider implementation

## Overview

Implement the `local` provider in `internal/sandbox/local/` that wraps the existing `localLauncher` and `localToolHost` from task 02 into a full `Provider` implementation. This is the baseline provider where `Prepare` and sync are no-ops, preserving current AGH behavior as the default sandbox. Also create the provider registry.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `Provider` interface from `internal/sandbox/types.go` for the `local` backend
- MUST return `BackendLocal` from `Backend()`
- MUST implement `Prepare()` as a no-op that returns `Prepared` with local paths unchanged
- MUST implement `SyncToRuntime()` and `SyncFromRuntime()` as no-ops (local doesn't need sync)
- MUST implement `Destroy()` as a no-op
- MUST compose `localLauncher` and `localToolHost` from task 02 into the `Prepared` result
- MUST create a provider registry (simple `map[Backend]Provider`) in `internal/sandbox/`
- MUST register `local` as the default provider
- SHOULD use compile-time interface verification: `var _ environment.Provider = (*localProvider)(nil)`
</requirements>

## Subtasks

- [x] 3.1 Create `internal/sandbox/local/provider.go` implementing `Provider`
- [x] 3.2 Create provider registry in `internal/sandbox/registry.go`
- [x] 3.3 Register `local` as default provider
- [x] 3.4 Add shared provider test suite for interface compliance

## Implementation Details

See TechSpec section: build order step 6.

The local provider is intentionally trivial. `Prepare` returns the same local paths. Sync is a no-op. The value is proving the interface works end-to-end before remote providers add complexity.

### Relevant Files

- `internal/sandbox/types.go` — Interfaces to implement (from task 01)
- `internal/acp/launcher.go` — `localLauncher` to compose (from task 02)
- `internal/acp/tool_host.go` — `localToolHost` to compose (from task 02)

### Dependent Files

- `internal/session/manager.go` — Will inject provider registry (task 04)
- `internal/daemon/daemon.go` — Will wire provider registry (task 04)
- `internal/sandbox/daytona/provider.go` — Will follow same pattern (task 06)

### Related ADRs

- [ADR-001: Daemon-Native Environment Providers](adrs/adr-001.md) — Local is first-class, not special-cased

## Deliverables

- `internal/sandbox/local/provider.go` — Local provider implementation
- `internal/sandbox/registry.go` — Provider registry
- Shared provider compliance test suite
- Unit tests with >=80% coverage

## Tests

- Unit tests:
  - [x] `localProvider.Backend()` returns `BackendLocal`
  - [x] `localProvider.Prepare()` returns `Prepared` with unchanged local paths
  - [x] `localProvider.Prepare()` returns `Prepared` with `RuntimeRootDir` == input `LocalRootDir`
  - [x] `localProvider.Prepare()` returns `Prepared` with `RuntimeAdditionalDirs` == input `LocalAdditionalDirs`
  - [x] `localProvider.SyncToRuntime()` returns nil (no-op)
  - [x] `localProvider.SyncFromRuntime()` returns nil (no-op)
  - [x] `localProvider.Destroy()` returns nil (no-op)
  - [x] Provider registry returns `local` provider for `BackendLocal`
  - [x] Provider registry returns error for unregistered backend
  - [x] Shared compliance suite: `Prepare` → `SyncToRuntime` → `SyncFromRuntime` → `Destroy` lifecycle
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- Compile-time interface verification passes
- Provider registry correctly resolves `local` backend
