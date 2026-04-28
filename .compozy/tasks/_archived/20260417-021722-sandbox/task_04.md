---
status: completed
title: "Session sandbox integration and daemon wiring"
type: backend
complexity: high
dependencies:
  - task_01
  - task_03
---

# Task 04: Session sandbox integration and daemon wiring

## Overview

Integrate the sandbox system into session lifecycle: inject the provider registry into the session manager, call `Provider.Prepare` and sync methods at the correct lifecycle points, persist `SessionSandboxMeta`, and wire everything through the daemon composition root. Also add sandbox info to session status/list APIs.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add sandbox provider registry as a dependency in `session.Manager` via `WithSandboxRegistry()` option
- MUST allocate daemon-owned `SandboxID` in `startSession()` after workspace resolution and before provider calls
- MUST persist `SessionSandboxMeta` in `creating` state before `Provider.Prepare()`
- MUST call `Provider.Prepare()` in `startSession()` after sandbox metadata persistence and before driver start
- MUST call `Provider.SyncToRuntime(state, SyncReasonStart)` after `Prepare` and before `Launch`
- MUST use `Prepared.RuntimeRootDir` and `Prepared.RuntimeAdditionalDirs` in `acp.StartOpts` instead of local paths
- MUST persist returned provider state (`InstanceID`, `RuntimeRootDir`, `RuntimeAdditionalDirs`, `ProviderState`, `SSHAccessExpiresAt`) before ACP launch
- MUST call `Provider.SyncFromRuntime(state, SyncReasonStop)` in `finalizeStopped` before store close
- MUST call `Provider.SyncFromRuntime(state, SyncReasonCrash)` on crash path (best-effort)
- MUST call `Provider.Destroy()` if sandbox profile has `DestroyOnStop` set
- MUST restore `SessionSandboxMeta` on resume and pass `SandboxID`, `InstanceID`, and `ProviderState` to `Prepare`
- MUST add `sandbox_id`, `sandbox_backend`, `sandbox_instance_id`, `sandbox_state`, and `sandbox_provider_state_json` columns to sessions table
- MUST add `SessionSandboxPayload` to session contract types and conversion functions
- MUST add `Sandbox` field to `SessionInfo`
- MUST wire provider registry, local provider, and session manager options in daemon composition root
- MUST add `SessionManagerDeps.SandboxRegistry` to daemon deps
- MUST add structured log events for sandbox lifecycle: `sandbox.prepare.start/complete/error`, `sandbox.sync.start/complete/error`, `sandbox.transport.connect/disconnect/error`, `sandbox.destroy.start/complete/error` with fields `backend`, `profile`, `sandbox_id`, `instance_id`, `workspace_id`, `session_id`, `duration_ms`
- MUST emit optional observability spans per TechSpec (`sandbox.prepare`, `sandbox.sync.to_runtime`, `sandbox.sync.from_runtime`, `sandbox.destroy`) through the existing observe layer when available
- SHOULD update CLI `session list` to show backend column and `session info` to show sandbox details
</requirements>

## Subtasks

- [x] 4.1 Add `WithSandboxRegistry()` option to session manager
- [x] 4.2 Integrate `allocate SandboxID → persist creating meta → Prepare → SyncToRuntime → Launch` sequence in `startSession`
- [x] 4.3 Integrate `SyncFromRuntime → Destroy` in session stop/crash paths
- [x] 4.4 Persist and restore `SessionSandboxMeta` across session lifecycle
- [x] 4.5 Add sandbox columns and provider-state JSON to sessions DB schema
- [x] 4.6 Add sandbox info to session contract types and API responses
- [x] 4.7 Wire sandbox registry in daemon composition root

## Implementation Details

See TechSpec sections: "Data flow — session create with Daytona (authoritative lifecycle)", build order steps 7-8.

The authoritative lifecycle sequence is: `sandbox.prepare hook → allocate/persist sandbox metadata → Prepare → SyncToRuntime(start) → sandbox.ready hook → Launch → ... → sandbox.sync.before hook → SyncFromRuntime(stop/crash) → sandbox.sync.after hook → sandbox.stop hook → Destroy`. This must be followed exactly.

### Relevant Files

- `internal/session/manager.go:56-81` — Add `WithSandboxRegistry` option
- `internal/session/manager_start.go:101-220` — Integrate prepare/sync/launch sequence
- `internal/session/manager_lifecycle.go:142-261` — Add sync-from/destroy in `finalizeStopped`
- `internal/session/session.go:44-60` — Add `Sandbox` to `SessionInfo`
- `internal/session/interfaces.go` — May need sandbox-related interface additions
- `internal/store/globaldb/global_db.go` — Add session sandbox columns
- `internal/store/globaldb/global_db_session.go` — Persist/load sandbox fields
- `internal/api/contract/contract.go:30-46` — Add `SessionSandboxPayload` to `SessionPayload`
- `internal/api/core/conversions.go:22-48` — Map sandbox in `SessionPayloadFromInfo`
- `internal/daemon/daemon.go:194-204` — Add `SandboxRegistry` to `SessionManagerDeps`
- `internal/daemon/daemon.go:376-389` — Wire sandbox into session manager creation

### Dependent Files

- `internal/daemon/boot.go` — Will use sandbox registry for cleanup (task 07)
- `internal/hooks/` — Will dispatch sandbox hooks (task 08)

### Related ADRs

- [ADR-003: Session-Scoped Sandbox](adrs/adr-003.md) — Session owns sync lifecycle, one sandbox per session

## Deliverables

- Updated session manager with sandbox lifecycle integration
- Updated session stop/crash paths with sync-from and destroy
- Updated session resume with sandbox state restoration using `SandboxID`, `InstanceID`, and `ProviderState`
- Updated DB schema with sandbox columns and provider-state JSON
- Updated session API contracts with sandbox info
- Updated daemon composition root with sandbox wiring
- Structured sandbox logs and observability spans
- Unit tests with >=80% coverage
- Integration test for full session lifecycle with local provider

## Tests

- Unit tests:
  - [x] Session start allocates an `SandboxID` before provider `Prepare`
  - [x] Session start persists `SessionSandboxMeta` in `creating` state before provider `Prepare`
  - [x] Session start calls `Provider.Prepare()` with correct `PrepareRequest` fields
  - [x] Session start calls `SyncToRuntime(SyncReasonStart)` after Prepare
  - [x] Session start uses `RuntimeRootDir` from Prepared in `StartOpts.Cwd`
  - [x] Session start uses `RuntimeAdditionalDirs` from Prepared in `StartOpts.AdditionalDirs`
  - [x] Session stop calls `SyncFromRuntime(SyncReasonStop)` before store close
  - [x] Session crash calls `SyncFromRuntime(SyncReasonCrash)` best-effort
  - [x] Session stop calls `Destroy()` when `DestroyOnStop` is true
  - [x] Session stop skips `Destroy()` when `DestroyOnStop` is false
  - [x] `SessionSandboxMeta` persists correctly in session metadata
  - [x] Session resume restores `SessionSandboxMeta` and passes `SandboxID`, `InstanceID`, and `ProviderState` to Prepare
  - [x] Session list API includes `environment` field in response
  - [x] `SessionInfo` includes sandbox ID, backend, profile, state, instance ID, and last sync error
  - [x] Sandbox lifecycle logs/spans include session ID, workspace ID, sandbox ID, backend, profile, duration, and error kind
- Integration tests:
  - [x] Full session create → prompt → stop lifecycle with local provider preserves current behavior
  - [x] Session resume with local provider works correctly
  - [x] Concurrent sessions on same workspace both complete without error (local provider)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing (including existing session tests unmodified)
- Test coverage >=80%
- `make verify` passes
- Session list/get/status show sandbox info
- Full local session lifecycle works end-to-end through daemon
