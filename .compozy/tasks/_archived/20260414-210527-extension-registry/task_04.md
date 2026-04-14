---
status: completed
title: Add extension registry schema, config, and CLI commands
type: backend
complexity: critical
dependencies:
  - task_02
  - task_03
---

# Task 04: Add extension registry schema, config, and CLI commands

## Overview

Add three nullable columns to the `extensions` SQLite table for remote install tracking, create the `[extensions.marketplace]` config section with validation, and wire the `search`, `install`, `remove`, and `update` CLI commands for extensions using MultiRegistry and Installer. The `install` command performs domain-specific SQLite registration after the domain-agnostic Installer returns. The `remove` command handles both filesystem cleanup and DB record deletion.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `registry_slug TEXT`, `registry_name TEXT`, `remote_version TEXT` columns to `extensions` table schema in `global_db.go:92-102`
- MUST add `RegistrySlug`, `RegistryName`, `RemoteVersion` fields to `ExtensionInfo` struct
- MUST update `scanExtensionInfo` and `installWithSource` to handle new columns
- MUST create `[extensions.marketplace]` config section with `registry` and `base_url` fields
- MUST include `Validate()` method following existing `MarketplaceConfig.Validate()` pattern at `config.go:582-611`
- MUST log warning when `base_url` uses `http://` scheme
- `agh extension search` MUST use MultiRegistry, skip sources with `Search: false`
- `agh extension install <slug>` MUST call `Installer.Install()` then `extension.Registry.Install()` with `SourceMarketplace`
- `agh extension remove <name>` MUST delete directory with `os.RemoveAll()` then call `Registry.Uninstall()`
- `agh extension update` MUST use `MultiRegistry.CheckUpdate()` and re-install if newer version found
- Phase 1: NO daemon notification — print "Restart daemon to activate" message
- MUST pass `make verify` after completion
</requirements>

## Subtasks
- [x] 4.1 Update extensions table schema in `global_db.go` and `ExtensionInfo` struct in `extension/registry.go`
- [x] 4.2 Update `scanExtensionInfo`, `installWithSource`, and SQL queries for new columns
- [x] 4.3 Add `ExtensionsMarketplaceConfig` struct and `Validate()` to `internal/config/config.go`
- [x] 4.4 Wire `agh extension search` command using MultiRegistry
- [x] 4.5 Wire `agh extension install` command with Installer + SQLite registration
- [x] 4.6 Wire `agh extension remove` command with filesystem + DB cleanup
- [x] 4.7 Wire `agh extension update` command with CheckUpdate + re-install

## Implementation Details

See TechSpec "Data Models > ExtensionInfo additions", "API Endpoints > New CLI Commands", and "Build Order Steps 7-9".

### Relevant Files
- `internal/store/globaldb/global_db.go:92-102` — Extensions table CREATE TABLE string to modify
- `internal/extension/registry.go:39-50` — `ExtensionInfo` struct to extend
- `internal/extension/registry.go:353-398` — `scanExtensionInfo` to update
- `internal/extension/registry.go:229-323` — `installWithSource` to update
- `internal/extension/registry.go:84-101` — `Uninstall()` is DB-only, CLI must add filesystem cleanup
- `internal/config/config.go:582-611` — `MarketplaceConfig.Validate()` pattern to follow
- `internal/config/config.go:117-124` — `SkillsConfig` struct pattern
- `internal/cli/extension.go:32-44` — `newExtensionCommand` registration pattern
- `internal/extension/capability.go:51-58` — `SourceMarketplace` security ceiling

### Dependent Files
- `internal/store/globaldb/global_db.go` — Schema string modified
- `internal/extension/registry.go` — Struct and queries modified
- `internal/config/config.go` — New config section added
- `internal/cli/extension.go` — New subcommands added

### Related ADRs
- [ADR-004: Reuse Existing SQLite extensions Table](adrs/adr-004.md) — Three nullable columns for remote tracking

## Deliverables
- Updated `global_db.go` with new schema columns
- Updated `extension/registry.go` with new fields and queries
- New `ExtensionsMarketplaceConfig` in `config.go` with Validate()
- 4 new CLI subcommands in `extension.go`
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for CLI commands **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `ExtensionInfo` with registry_slug/registry_name/remote_version round-trips through insert and query
  - [x] `ExtensionInfo` with nil registry fields (local install) works correctly
  - [x] Concurrent install of same extension → one success, one `ErrExtensionExists`
  - [x] `ExtensionsMarketplaceConfig.Validate()` accepts valid HTTPS config
  - [x] `ExtensionsMarketplaceConfig.Validate()` accepts empty config (disabled)
  - [x] `ExtensionsMarketplaceConfig.Validate()` logs warning for HTTP base_url
  - [x] `ExtensionsMarketplaceConfig.Validate()` rejects invalid URL (no host)
- Integration tests:
  - [x] `agh extension search <query>` returns results from configured sources
  - [x] `agh extension install <slug>` downloads, extracts, registers in DB with `SourceMarketplace`
  - [x] `agh extension install <slug>` prints "Restart daemon to activate" (no Reload call)
  - [x] `agh extension remove <name>` deletes directory AND DB record
  - [x] `agh extension remove <name>` for non-existent extension returns clear error
  - [x] `agh extension update --check` shows available updates without installing
  - [x] `agh extension update <name>` re-installs with newer version
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Extensions installed remotely show `source: marketplace` in `agh extension list`
- `SourceMarketplace` security ceiling enforced (no capability escalation)
