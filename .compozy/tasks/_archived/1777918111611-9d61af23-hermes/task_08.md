---
status: completed
title: CLI Config and Setup Lifecycle
type: chore
complexity: high
dependencies:
  - task_01
  - task_05
---

# Task 08: CLI Config and Setup Lifecycle

## Overview

Harden operator setup, configuration, and managed lifecycle commands. This task adds or completes `agh config *`, uninstall, completion, install/update behavior, `AGH_MANAGED` detection, and config redaction integration, using MCP auth redaction from task_05 as the secure baseline.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, task_01, and task_05 outputs before changing config commands
- DO NOT print secrets in config, setup, install, update, or diagnostics output
- DO NOT hardcode config paths outside existing home/workspace resolution rules
- DO NOT add shell-specific behavior without tests or documented fallback
- Install/update/uninstall commands must be idempotent and clear about managed vs unmanaged state
</critical>

<requirements>
- MUST implement `agh config` subcommands for get, set, list, path, validate, and redacted display where applicable
- MUST implement uninstall and shell completion behavior consistent with existing CLI patterns
- MUST harden install/update lifecycle commands with `AGH_MANAGED` detection
- MUST reuse task_05 redaction rules for auth-sensitive MCP and config output
- MUST add CLI tests for config mutation, validation, redaction, completion, and managed lifecycle behavior
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 8.1 Inventory current CLI setup/config commands and define the missing command matrix
- [x] 8.2 Implement `agh config get/set/list/path/validate` with redacted output and validation errors
- [x] 8.3 Add uninstall, completion, install, and update lifecycle hardening with `AGH_MANAGED` handling
- [x] 8.4 Add tests for config persistence, redaction, shell completion, install/update, and uninstall paths
- [x] 8.5 Update operator docs, web settings contracts, and setup docs to match the final CLI behavior
- [x] 8.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Keep command behavior consistent with Cobra conventions already used by the CLI. Config mutation should go through existing persistence and validation APIs rather than ad hoc TOML string manipulation. Setup lifecycle commands should report state clearly without modifying unrelated user files.

### Relevant Files
- `internal/cli/root.go` - command tree and completion wiring
- `internal/cli/install.go` - install/update/uninstall lifecycle behavior
- `internal/cli/extension.go` - managed extension command interactions
- `internal/config/persistence.go` - config read/write behavior
- `internal/config/bootstrap.go` - setup bootstrap defaults
- `internal/config/config.go` - validation and config model
- `internal/api/contract/settings.go` - settings payloads if web config surfaces change
- `internal/api/core/settings.go` - settings handlers if web config surfaces change

### Dependent Files
- `internal/cli/*config*_test.go` - config command behavior tests
- `internal/cli/*install*_test.go` - install/update/uninstall lifecycle tests
- `internal/config/*_test.go` - config validation and persistence tests
- `web/src/routes/_app/settings/` - settings forms or typed config clients if impacted
- `packages/site/` - CLI setup, config, and install docs
- `.compozy/tasks/hermes/task_09.md` - release hardening depends on setup lifecycle consistency

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - includes setup and CLI lifecycle hardening
- [ADR-003: MCP OAuth Auth Subsystem](adrs/adr-003-mcp-oauth-auth-subsystem.md) - redaction rules from MCP auth must be reused by config output

## Deliverables
- Completed `agh config` command set
- Hardened install, update, uninstall, and completion behavior
- `AGH_MANAGED` detection and user-facing lifecycle diagnostics
- Redacted config output aligned with MCP auth security rules
- Tests for CLI lifecycle and config behavior
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Config get/set/list/path/validate commands read and mutate through config APIs
  - [x] Redacted display hides all sensitive MCP and auth values
  - [x] Completion command emits valid shell completion output
  - [x] Managed install/update/uninstall paths are idempotent
- Integration tests:
  - [x] CLI config commands operate against isolated temp homes and workspaces
  - [x] Install/update lifecycle reports managed and unmanaged state correctly
  - [x] Settings web or docs updates match the actual command behavior
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Operators can inspect, validate, and modify AGH config safely from the CLI
- Setup lifecycle commands are idempotent and managed-state aware
- Secret redaction is consistent across config and MCP auth output
- Affected CLI, config, web, and docs tests pass
