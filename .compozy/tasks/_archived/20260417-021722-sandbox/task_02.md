---
status: completed
title: "Extract ACP Launcher and ToolHost interfaces"
type: refactor
complexity: critical
dependencies:
  - task_01
---

# Task 02: Extract ACP Launcher and ToolHost interfaces

## Overview

Refactor the ACP package to replace hardcoded local subprocess spawning and file/terminal/permission handlers with injected `Launcher` and `ToolHost` interfaces. This is the most critical refactor — it must produce zero observable behavior change while creating the seam that remote providers plug into.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST extract `spawnProcess` from `acp/client.go:143-191` into a `localLauncher` implementing the `Launcher` interface from task 01
- MUST inject `Launcher` into `acp.Driver` via functional option `WithLauncher()`
- MUST extract file IO handlers (`handleReadTextFile`, `handleWriteTextFile`) from `acp/handlers.go:180-213` into a `localToolHost`
- MUST extract terminal handlers (`handleCreateTerminal`, `handleKillTerminal`, etc.) from `acp/handlers.go:323-405` into the same `localToolHost`
- MUST extract permission methods (`Authorize`, `PermissionDecision`) from `acp/permission.go` into `localToolHost` per TechSpec ToolHost interface
- MUST inject `ToolHost` into ACP handler dispatch
- MUST preserve the existing `permissionPolicy` root-path-based validation inside `localToolHost`
- MUST produce zero observable behavior change — all existing ACP tests must pass unmodified
- MUST NOT change any ACP protocol wire types or session negotiation
</requirements>

## Subtasks

- [x] 2.1 Create `localLauncher` implementing `Launcher` by extracting `spawnProcess` logic
- [x] 2.2 Add `WithLauncher()` option to `acp.Driver` constructor, defaulting to `localLauncher`
- [x] 2.3 Create `localToolHost` implementing `ToolHost` by extracting file IO, permission, and terminal handlers
- [x] 2.4 Inject `ToolHost` into `AgentProcess` handler dispatch, replacing direct `os.*` calls
- [x] 2.5 Verify all existing ACP tests pass with zero modification
- [x] 2.6 Add tests for `localLauncher` and `localToolHost` interface compliance

## Implementation Details

See TechSpec sections: "Core Interfaces" (ToolHost definition), build order steps 4-5.

The `localLauncher` wraps the existing `subprocess.Launch` call. The `localToolHost` wraps `os.ReadFile`, `os.WriteFile`, `os.MkdirAll`, the existing `permissionPolicy`, and the terminal manager. Both implementations preserve the exact current behavior.

### Relevant Files

- `internal/acp/client.go:143-191` — `spawnProcess` to extract into `localLauncher`
- `internal/acp/client.go:87` — `New()` constructor where `WithLauncher` option is added
- `internal/acp/handlers.go:180-213` — File IO handlers to extract
- `internal/acp/handlers.go:215-265` — `handleRequestPermission` to extract
- `internal/acp/handlers.go:323-405` — Terminal handlers to extract
- `internal/acp/permission.go:94-181` — Permission policy to wrap in `localToolHost`
- `internal/acp/types.go:45-56` — `StartOpts` type (unchanged)
- `internal/acp/client_test.go` — Existing tests must pass unchanged
- `internal/acp/handlers_test.go` — Existing tests must pass unchanged
- `internal/acp/client_integration_test.go` — Existing integration tests must pass

### Dependent Files

- `internal/sandbox/local/provider.go` — Will compose `localLauncher` + `localToolHost` (task 03)
- `internal/sandbox/daytona/provider.go` — Will provide alternative implementations (task 06)
- `internal/daemon/daemon.go` — Will inject Launcher/ToolHost into Driver (task 04)

### Related ADRs

- [ADR-001: Daemon-Native Environment Providers](adrs/adr-001.md) — ToolHost is in-process for zero-latency
- [ADR-002: SSH as Primary Transport](adrs/adr-002.md) — Launcher interface enables SSH transport swap

## Deliverables

- `internal/acp/launcher.go` — `localLauncher` implementation
- `internal/acp/tool_host.go` — `ToolHost` interface + `localToolHost` implementation
- Updated `acp.Driver` with `WithLauncher()` and `WithToolHost()` options
- Updated handler dispatch routing through `ToolHost`
- All existing ACP tests passing with zero modifications
- New unit tests for interface compliance with >=80% coverage

## Tests

- Unit tests:
  - [x] `localLauncher.Launch` spawns subprocess and returns valid `Handle` with working Stdin/Stdout
  - [x] `localLauncher.Launch` with invalid command returns error
  - [x] `localToolHost.ReadTextFile` reads existing file content correctly
  - [x] `localToolHost.ReadTextFile` with non-existent file returns error
  - [x] `localToolHost.WriteTextFile` creates file with correct content and permissions
  - [x] `localToolHost.WriteTextFile` creates parent directories
  - [x] `localToolHost.ResolvePath` resolves relative path against root
  - [x] `localToolHost.ResolvePath` rejects path outside workspace root
  - [x] `localToolHost.Authorize` with `approve-all` mode permits all operations
  - [x] `localToolHost.Authorize` with `deny-all` mode rejects all operations
  - [x] `localToolHost.CreateTerminal` spawns terminal process with correct cwd
  - [x] `Handle.Stop` sends SIGTERM and waits for exit
  - [x] `Handle.Done` channel closes when process exits
- Integration tests:
  - [x] Full ACP session create → prompt → file read/write → terminal → stop works through new interfaces
  - [x] Existing `client_integration_test.go` passes without modification (regression gate)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing (including ALL existing ACP tests unmodified)
- Test coverage >=80%
- `make verify` passes with zero warnings
- Zero observable behavior change from user perspective
- `Launcher` and `ToolHost` are injectable via functional options
