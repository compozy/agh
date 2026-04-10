---
status: completed
title: CLI commands (list, install, enable, disable)
type: backend
complexity: medium
dependencies:
  - task_05
  - task_06
---

# Task 09: CLI commands (list, install, enable, disable)

## Overview

Add the `agh extension` subcommand tree to the existing Cobra CLI. Users install local extensions via `agh extension install <path>`, list what is registered, enable or disable extensions without uninstalling, and inspect runtime status. The commands talk to the daemon via the existing UDS API transport. Output honors the standard AGH `--format` flag supporting human-readable, JSON, and TOON.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/cli/extension.go` with Cobra command tree: `agh extension {list, install, enable, disable, status}`
- MUST implement `install` subcommand accepting a local directory path, parsing the manifest, computing checksum, and registering via the extension registry
- MUST implement `list` subcommand showing name, version, source, enabled state, capabilities
- MUST implement `enable <name>` and `disable <name>` subcommands that update registry state and trigger a daemon reload if daemon is running
- MUST implement `status <name>` showing runtime state, PID, uptime, health, last error
- MUST honor the existing `--format` flag (human, json, toon) used by other CLI commands
- MUST communicate with the running daemon via the existing UDS API client if the daemon is running, and operate directly on the registry when daemon is offline
- MUST return clear error messages when extension directory is missing, manifest is invalid, or checksum verification fails
- MUST NOT implement git URL installation, marketplace fetch, or remote install in this task (deferred)
</requirements>

## Subtasks
- [x] 9.1 Create `internal/cli/extension.go` with root `agh extension` command and subcommands
- [x] 9.2 Implement `install <path>` subcommand with manifest parsing and registry write
- [x] 9.3 Implement `list` subcommand with --format support
- [x] 9.4 Implement `enable`, `disable`, `status` subcommands
- [x] 9.5 Add `agh extension` command tree to `internal/cli/root.go`
- [x] 9.6 Write unit and integration tests using CLI test harness

## Implementation Details

New file `internal/cli/extension.go`. Extend `internal/cli/root.go` to register the new command tree. Follow the existing command patterns used by `skill`, `agent`, and `workspace` commands.

See TechSpec "Development Sequencing" section for the CLI scope. See `_examples.md` section 8 for the expected output format.

### Relevant Files
- `internal/cli/root.go` — Root command registration for CLI tree
- `internal/cli/skill.go` — Existing skill command tree to mirror
- `internal/cli/agent.go` — Existing agent command tree for format handling pattern
- `internal/extension/registry.go` — Registry CRUD operations (task 05)
- `internal/extension/manager.go` — Manager for runtime status queries (task 06)
- `internal/extension/manifest.go` — Manifest loader used by install (task 03)
- `internal/api/udsapi/` — UDS client for communicating with running daemon

### Dependent Files
- Nothing depends on this task; it is a user-facing leaf

### Related ADRs
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — CLI exposes the resource/capability/action model to users

## Deliverables
- New `internal/cli/extension.go` with full command tree
- Updated `internal/cli/root.go` registering the extension command
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for install → list → disable → uninstall flow via CLI **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `install` parses valid manifest directory and calls registry.Install
  - [x] `install` with missing directory returns clear error
  - [x] `install` with invalid manifest returns parsing error
  - [x] `install` with checksum mismatch returns verification error
  - [x] `list` outputs human format with columns: name, version, type, state, capabilities
  - [x] `list --format=json` outputs valid JSON array
  - [x] `list --format=toon` outputs TOON format
  - [x] `enable <name>` sets registry enabled=true
  - [x] `enable <unknown>` returns `ErrExtensionNotFound`
  - [x] `disable <name>` sets registry enabled=false
  - [x] `status <name>` shows runtime state when daemon is running
  - [x] `status <name>` shows registry-only state when daemon is offline
- Integration tests:
  - [x] Full CLI flow: install test extension → list shows it → status shows active → disable → list shows disabled
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- All 5 subcommands functional with format support
- `make verify` passes
