---
status: completed
title: "Core sandbox types, config profiles, and workspace resolution"
type: backend
complexity: high
dependencies: []
---

# Task 01: Core sandbox types, config profiles, and workspace resolution

## Overview

Create the foundational `internal/sandbox/` package with core type definitions, add sandbox profile configuration to the config system with validation and merge support, and extend workspace resolution to select and resolve sandbox profiles. This establishes the data model that all subsequent tasks depend on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/sandbox/types.go` with all core type definitions per TechSpec "Core Interfaces" section: `Backend`, `Provider`, `Launcher`, `Handle`, `ToolHost`, `Resolved`, `SessionState`, `PrepareRequest`, `Prepared`, `SyncReason`, `LaunchSpec`
- MUST add `SandboxProfile`, `DaytonaProfile`, `NetworkProfile` to config per TechSpec "Data Models" section
- MUST add `DaytonaProfile.Snapshot` and validate snapshot/image precedence per TechSpec
- MUST add `Defaults.Sandbox` string field to `DefaultsConfig`
- MUST add `Sandboxes map[string]SandboxProfile` to `Config`
- MUST add validation for sandbox profiles (valid backend, valid sync mode, valid persistence)
- MUST add overlay merge support for `[sandboxes.*]` TOML sections
- MUST add `SandboxRef` to `Workspace` struct and `RegisterOptions`/`UpdateOptions`
- MUST add `Environment sandbox.Resolved` to `ResolvedWorkspace`
- MUST resolve environment during `buildResolvedWorkspace` with cascade: `Workspace.SandboxRef` → `Config.Defaults.Sandbox` → implicit `local`
- MUST add `sandbox_ref` column to workspaces table in globaldb schema
- MUST add `SandboxRef` to workspace CRUD contract types (`CreateWorkspaceRequest`, `UpdateWorkspaceRequest`, `WorkspacePayload`)
- MUST add `--sandbox` flag to CLI `workspace add` and `workspace edit` commands
- MUST add `Env map[string]string` field to `SandboxProfile` for profile-level env var injection
- MUST add `SessionSandboxMeta` type with `SandboxID`, `State`, `RuntimeAdditionalDirs`, `ProviderState`, `SSHAccessExpiresAt`, `LastSyncAt`, and `LastSyncError` per TechSpec
- NOTE: `ToolHost` interface is defined in task 01 types but implemented in `internal/acp/tool_host.go` by task 02. Task 01 defines the contract, task 02 owns the file location.
</requirements>

## Subtasks

- [x] 1.1 Create `internal/sandbox/` package with core types and interfaces
- [x] 1.2 Add sandbox profile types and validation to config package
- [x] 1.3 Add sandbox overlay merge logic to config merge system
- [x] 1.4 Extend workspace domain types with `SandboxRef` and `ResolvedWorkspace.Sandbox`
- [x] 1.5 Add `sandbox_ref` column to workspace DB schema and persistence
- [x] 1.6 Extend workspace CRUD contract types and CLI flags
- [x] 1.7 Add `SessionSandboxMeta` to store types

## Implementation Details

See TechSpec sections: "Core Interfaces", "Data Models", "API Endpoints — Workspace contract changes".

### Relevant Files

- `internal/config/config.go` — Add `SandboxProfile`, `DaytonaProfile`, `NetworkProfile`, `Defaults.Sandbox`, `Config.Sandboxes`
- `internal/config/merge.go` — Add overlay merge for `[sandboxes.*]`
- `internal/config/config_test.go` — Validation and parsing tests
- `internal/workspace/workspace.go:28-46` — Add `SandboxRef` to `Workspace`, `Environment` to `ResolvedWorkspace`
- `internal/workspace/resolver.go:18-30` — Add `SandboxRef` to `RegisterOptions`, `UpdateOptions`
- `internal/workspace/resolver.go:218-242` — Resolve environment in `buildResolvedWorkspace`
- `internal/store/globaldb/global_db.go:16+` — Add `sandbox_ref` column to workspace schema
- `internal/store/globaldb/global_db_workspace.go` — Persist/load `SandboxRef`
- `internal/store/types.go` — Add `SessionSandboxMeta`
- `internal/api/contract/contract.go:482-510` — Add `SandboxRef` to workspace contract types
- `internal/api/core/conversions.go:381-395` — Map `SandboxRef` in `WorkspacePayloadFromWorkspace`
- `internal/cli/workspace.go` — Add `--sandbox` flag

### Dependent Files

- `internal/sandbox/local/` — Will consume core types (task 03)
- `internal/sandbox/daytona/` — Will consume core types (task 06)
- `internal/session/manager_start.go` — Will consume `ResolvedWorkspace.Sandbox` (task 04)
- `internal/acp/` — Will implement `Launcher`/`ToolHost` interfaces (task 02)

### Related ADRs

- [ADR-001: Daemon-Native Environment Providers](adrs/adr-001.md) — Provider types are daemon-native, not extensions
- [ADR-003: Session-Scoped Sandbox](adrs/adr-003.md) — SessionState and sync types defined here

## Deliverables

- `internal/sandbox/types.go` with all core types and interfaces
- Config types with validation and merge support
- Daytona profile snapshot field with validation and snapshot/image precedence
- Workspace types with `SandboxRef` field and resolution logic
- DB schema migration for `sandbox_ref` column
- Updated workspace CRUD contracts and CLI
- `SessionSandboxMeta` type
- Unit tests with >=80% coverage
- Integration test for config→workspace→environment resolution round-trip

## Tests

- Unit tests:
  - [x] Config: valid sandbox profile parses correctly from TOML
  - [x] Config: `DaytonaProfile.Snapshot` parses correctly and wins over `Image` in resolved profile policy
  - [x] Config: invalid backend in profile returns validation error
  - [x] Config: invalid sync_mode in profile returns validation error
  - [x] Config: environment overlay merge preserves provider-specific fields
  - [x] Config: `SandboxProfile.Env` map parses and preserves key-value pairs
  - [x] Store: `SessionSandboxMeta` preserves `SandboxID`, `ProviderState`, SSH expiry, and sync status fields through JSON round-trip
  - [x] Config: `Defaults.Sandbox` cascade resolves to profile
  - [x] Workspace: `SandboxRef` persists through register/update/load cycle
  - [x] Workspace: resolution cascade `SandboxRef` → `Defaults.Sandbox` → `local`
  - [x] Workspace: missing `SandboxRef` with no default resolves to `local`
  - [x] Contract: `CreateWorkspaceRequest` with `sandbox_ref` serializes correctly
  - [x] Contract: `WorkspacePayload` includes `sandbox_ref` in JSON
- Integration tests:
  - [x] Full TOML config with `[sandboxes.daytona-dev]` section loads and validates
  - [x] Workspace register with `SandboxRef` persists to DB and resolves correctly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes with zero warnings
- Environment types compile and are importable by other packages
- Config with sandbox profiles loads, validates, and merges correctly
- Workspace CRUD exposes `sandbox_ref` via CLI, HTTP, and UDS
