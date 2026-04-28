---
status: completed
title: "Sandbox extension hooks and Host API"
type: backend
complexity: high
dependencies:
  - task_04
---

# Task 08: Sandbox extension hooks and Host API

## Overview

Register 5 sandbox lifecycle hooks and 3 Host API methods in AGH's extension system, enabling extensions to observe and influence sandbox lifecycle events. This follows the established hook pattern (events, payloads, dispatch, matcher) and the Host API method registration pattern.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST register 5 hook events: `sandbox.prepare` (sync), `sandbox.ready` (async), `sandbox.sync.before` (sync), `sandbox.sync.after` (async), `sandbox.stop` (sync)
- MUST create payload types following `PayloadBase` + `SessionContext` convention per TechSpec
- MUST include `sandbox_id` in lifecycle payloads after it has been allocated, and include sync stats (`files_synced`, `bytes_transferred`, `duration_ms`, `errors`) in `sandbox.sync.after`
- MUST create patch types for sync hooks supporting `ControlPatch.Deny` per TechSpec
- MUST add dispatch methods in `hooks/dispatch.go`
- MUST add matcher functions for sandbox hook events
- MUST dispatch hooks from session sandbox lifecycle points (Prepare, SyncTo, SyncFrom, Stop)
- MUST register 3 Host API methods: `sandbox/list`, `sandbox/info`, `sandbox/exec`
- MUST add method specs to `internal/extension/contract/host_api.go` method registry
- MUST add security capability mapping in `internal/extension/capability.go` for `sandbox.exec`
- MUST add protocol handler registration in `internal/extension/protocol/host_api.go`
- MUST require `sandbox.exec` security capability for the exec method; `sandbox/list` and `sandbox/info` require no special capability
- MUST follow request/response shapes from TechSpec "API Endpoints — Host API methods"
- MUST include `sandbox_id`, `sync_state`, and `last_sync_error` in `sandbox/info` responses
- MUST handle `Deny` patches in sync hooks: if `sandbox.prepare` is denied, abort session creation with error; if `sandbox.sync.before` is denied, skip sync
</requirements>

## Subtasks

- [x] 8.1 Define 5 sandbox hook events in `hooks/events.go`
- [x] 8.2 Create payload and patch types in `hooks/payloads.go`
- [x] 8.3 Add dispatch methods and matchers in `hooks/dispatch.go` and `hooks/matcher.go`
- [x] 8.4 Dispatch hooks from session sandbox lifecycle in `session/manager_start.go` and `session/manager_lifecycle.go`
- [x] 8.5 Register 3 Host API methods in extension Host API contract
- [x] 8.6 Implement Host API handlers for `sandbox/list`, `sandbox/info`, `sandbox/exec`

## Implementation Details

See TechSpec sections: "API Endpoints — Extension hooks (lifecycle)", "API Endpoints — Host API methods", build order step 13.

Adding a new hook event requires changes across: `events.go` (event definition), `payloads.go` (payload/patch types), `dispatch.go` (dispatch method), `matcher.go` (matcher function). Follow the existing `session.pre_create` pattern as a template.

### Relevant Files

- `internal/hooks/events.go` — Add 5 new events to `allHookEvents`
- `internal/hooks/payloads.go` — Add sandbox payload/patch types
- `internal/hooks/dispatch.go` — Add dispatch methods
- `internal/hooks/matcher.go` — Add matcher functions
- `internal/hooks/types.go` — Event type constants
- `internal/session/manager_start.go` — Dispatch `sandbox.prepare` and `sandbox.ready`
- `internal/session/manager_lifecycle.go` — Dispatch `sandbox.sync.*` and `sandbox.stop`
- `internal/extension/contract/host_api.go` — Register 3 new Host API method specs
- `internal/extension/protocol/host_api.go` — Add protocol handler registration for new methods
- `internal/extension/capability.go` — Add `sandbox.exec` security capability mapping
- `internal/daemon/daemon.go` — Wire Host API method handlers
- `internal/daemon/extensions.go` — Inject environment method handlers into extension manager

### Dependent Files

- Extension manifests (external) — Can now declare `sandbox.*` hooks

### Related ADRs

- [ADR-001: Daemon-Native Environment Providers](adrs/adr-001.md) — Extension hooks provide extensibility without compromising hot path

## Deliverables

- 5 new hook events with payload/patch types
- Dispatch methods and matchers for all 5 events
- Hook dispatch calls at correct session lifecycle points
- 3 Host API methods registered and implemented
- Unit tests with >=80% coverage
- Integration test for hook dispatch during session lifecycle

## Tests

- Unit tests:
  - [x] `sandbox.prepare` event fires during session start with correct payload fields
  - [x] `sandbox.prepare` hook deny aborts session creation with error
  - [x] `sandbox.prepare` hook patch `env_overrides` merges into environment config
  - [x] `sandbox.ready` event fires after Prepare succeeds with sandbox ID and instance ID
  - [x] `sandbox.sync.before` event fires before sync with correct direction and reason
  - [x] `sandbox.sync.before` deny skips sync operation
  - [x] `sandbox.sync.before` patch `exclude_patterns` passes to sync
  - [x] `sandbox.sync.after` event fires with files synced, bytes transferred, duration, and errors
  - [x] `sandbox.stop` event fires before sandbox teardown with sandbox ID
  - [x] `sandbox.stop` deny prevents destroy but still stops session
  - [x] Host API `sandbox/list` returns active sandbox instances
  - [x] Host API `sandbox/info` returns sandbox ID, runtime root, sync state, and last sync error for valid session
  - [x] Host API `sandbox/info` returns error for invalid session
  - [x] Host API `sandbox/exec` requires `sandbox.exec` capability
  - [x] Host API `sandbox/exec` executes command and returns exit code + output
- Integration tests:
  - [x] Session lifecycle with native hook registered fires all sandbox events in correct order
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- All 5 hook events fire at correct lifecycle points
- Deny patches correctly abort or skip operations
- Host API methods accessible from extension subprocess
