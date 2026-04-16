---
status: completed
title: "Environment extension hooks and Host API"
type: backend
complexity: high
dependencies:
  - task_04
---

# Task 08: Environment extension hooks and Host API

## Overview

Register 5 environment lifecycle hooks and 3 Host API methods in AGH's extension system, enabling extensions to observe and influence environment lifecycle events. This follows the established hook pattern (events, payloads, dispatch, matcher) and the Host API method registration pattern.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST register 5 hook events: `environment.prepare` (sync), `environment.ready` (async), `environment.sync.before` (sync), `environment.sync.after` (async), `environment.stop` (sync)
- MUST create payload types following `PayloadBase` + `SessionContext` convention per TechSpec
- MUST include `environment_id` in lifecycle payloads after it has been allocated, and include sync stats (`files_synced`, `bytes_transferred`, `duration_ms`, `errors`) in `environment.sync.after`
- MUST create patch types for sync hooks supporting `ControlPatch.Deny` per TechSpec
- MUST add dispatch methods in `hooks/dispatch.go`
- MUST add matcher functions for environment hook events
- MUST dispatch hooks from session environment lifecycle points (Prepare, SyncTo, SyncFrom, Stop)
- MUST register 3 Host API methods: `environment/list`, `environment/info`, `environment/exec`
- MUST add method specs to `internal/extension/contract/host_api.go` method registry
- MUST add security capability mapping in `internal/extension/capability.go` for `environment.exec`
- MUST add protocol handler registration in `internal/extension/protocol/host_api.go`
- MUST require `environment.exec` security capability for the exec method; `environment/list` and `environment/info` require no special capability
- MUST follow request/response shapes from TechSpec "API Endpoints — Host API methods"
- MUST include `environment_id`, `sync_state`, and `last_sync_error` in `environment/info` responses
- MUST handle `Deny` patches in sync hooks: if `environment.prepare` is denied, abort session creation with error; if `environment.sync.before` is denied, skip sync
</requirements>

## Subtasks

- [x] 8.1 Define 5 environment hook events in `hooks/events.go`
- [x] 8.2 Create payload and patch types in `hooks/payloads.go`
- [x] 8.3 Add dispatch methods and matchers in `hooks/dispatch.go` and `hooks/matcher.go`
- [x] 8.4 Dispatch hooks from session environment lifecycle in `session/manager_start.go` and `session/manager_lifecycle.go`
- [x] 8.5 Register 3 Host API methods in extension Host API contract
- [x] 8.6 Implement Host API handlers for `environment/list`, `environment/info`, `environment/exec`

## Implementation Details

See TechSpec sections: "API Endpoints — Extension hooks (lifecycle)", "API Endpoints — Host API methods", build order step 13.

Adding a new hook event requires changes across: `events.go` (event definition), `payloads.go` (payload/patch types), `dispatch.go` (dispatch method), `matcher.go` (matcher function). Follow the existing `session.pre_create` pattern as a template.

### Relevant Files

- `internal/hooks/events.go` — Add 5 new events to `allHookEvents`
- `internal/hooks/payloads.go` — Add environment payload/patch types
- `internal/hooks/dispatch.go` — Add dispatch methods
- `internal/hooks/matcher.go` — Add matcher functions
- `internal/hooks/types.go` — Event type constants
- `internal/session/manager_start.go` — Dispatch `environment.prepare` and `environment.ready`
- `internal/session/manager_lifecycle.go` — Dispatch `environment.sync.*` and `environment.stop`
- `internal/extension/contract/host_api.go` — Register 3 new Host API method specs
- `internal/extension/protocol/host_api.go` — Add protocol handler registration for new methods
- `internal/extension/capability.go` — Add `environment.exec` security capability mapping
- `internal/daemon/daemon.go` — Wire Host API method handlers
- `internal/daemon/extensions.go` — Inject environment method handlers into extension manager

### Dependent Files

- Extension manifests (external) — Can now declare `environment.*` hooks

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
  - [x] `environment.prepare` event fires during session start with correct payload fields
  - [x] `environment.prepare` hook deny aborts session creation with error
  - [x] `environment.prepare` hook patch `env_overrides` merges into environment config
  - [x] `environment.ready` event fires after Prepare succeeds with environment ID and instance ID
  - [x] `environment.sync.before` event fires before sync with correct direction and reason
  - [x] `environment.sync.before` deny skips sync operation
  - [x] `environment.sync.before` patch `exclude_patterns` passes to sync
  - [x] `environment.sync.after` event fires with files synced, bytes transferred, duration, and errors
  - [x] `environment.stop` event fires before sandbox teardown with environment ID
  - [x] `environment.stop` deny prevents destroy but still stops session
  - [x] Host API `environment/list` returns active environment instances
  - [x] Host API `environment/info` returns environment ID, runtime root, sync state, and last sync error for valid session
  - [x] Host API `environment/info` returns error for invalid session
  - [x] Host API `environment/exec` requires `environment.exec` capability
  - [x] Host API `environment/exec` executes command and returns exit code + output
- Integration tests:
  - [x] Session lifecycle with native hook registered fires all environment events in correct order
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- All 5 hook events fire at correct lifecycle points
- Deny patches correctly abort or skip operations
- Host API methods accessible from extension subprocess
