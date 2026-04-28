---
status: pending
title: Tools Config Lifecycle and Agent Grammar
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Tools Config Lifecycle and Agent Grammar

## Overview

Add the configuration and agent-definition grammar required by the registry before policy and dispatch consume it. This task introduces final `config.toml` keys, defaults, validation, merge behavior, and agent `tools`/`toolsets`/`deny_tools` semantics without compatibility bridges.

<critical>
- ALWAYS READ `_techspec.md`, ADR-005, ADR-006, and ADR-007 before editing config behavior
- DO NOT add deprecated aliases, fallback key names, or old-state migrations
- DO NOT hand-edit `go.mod`; use `go get` only if a dependency is truly required
- TESTS REQUIRED: config defaults, overlays, validation, and examples must move together
</critical>

<requirements>
1. MUST add `[tools]`, `[tools.policy]`, and `[tools.hosted_mcp]` config sections with TechSpec defaults.
2. MUST add agent `toolsets` and `deny_tools` while validating existing `tools` as canonical ToolID atoms or explicit patterns allowed by the policy grammar.
3. MUST model approval timeout, hosted MCP bind nonce TTL, result byte limits, source defaults, and `trusted_sources`.
4. MUST validate unknown, contradictory, or unsafe config values at load time with deterministic errors.
5. MUST update merge/overlay behavior, examples, config docs references, and config tests in the same change.
6. MUST avoid compatibility shims for any rejected or renamed key.
</requirements>

## Subtasks
- [ ] 2.1 Add tools config structs, defaults, and validation
- [ ] 2.2 Extend agent config with `toolsets` and `deny_tools`
- [ ] 2.3 Validate ToolID atoms, source grants, trusted sources, and hosted MCP values
- [ ] 2.4 Update merge, workspace overlay, and example config behavior
- [ ] 2.5 Add config tests for defaults, invalid values, overlays, and agent grammar
- [ ] 2.6 Document downstream docs and generated examples required by task_14

## Implementation Details

Use TechSpec "Config Lifecycle" and "Agent Manageability" sections. Keep this task focused on loading and validating configuration; do not implement registry policy decisions here beyond parse-time validation needed by task_03.

### Relevant Files
- `internal/config/config.go` - root config struct and defaults
- `internal/config/agent.go` - agent tool grammar
- `internal/config/provider.go` - MCP server config values consumed by later MCP tasks
- `internal/config/*_test.go` - config validation and merge tests
- `packages/site/content/runtime/core/configuration/config-toml.mdx` - docs target for task_14

### Dependent Files
- `internal/tools/policy*.go` - task_03 consumes parsed config
- `internal/acp/permission.go` - task_03 maps ACP ceiling into policy decisions
- `internal/session/manager_start.go` - task_10 consumes hosted MCP config
- `packages/site/content/runtime/core/sessions/permissions.mdx` - task_14 documents permission behavior

### Related ADRs
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - constrains policy and approval config
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - distinguishes operator and session visibility
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - constrains config atoms

### Web/Docs Impact
- `web/`: generated settings or diagnostics types may change later through task_11/task_13; no direct web code in this task.
- `packages/site`: task_14 must update `packages/site/content/runtime/core/configuration/config-toml.mdx`, `packages/site/content/runtime/core/sessions/permissions.mdx`, and any config examples covering agents or MCP.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: config now gates extension-host and MCP source trust, toolsets, and risk classes used by later executable providers.
- Agent manageability: establishes agent-readable config semantics but no CLI/HTTP/UDS management endpoints yet.
- Config lifecycle: adds `tools.*`, `tools.policy.*`, `tools.hosted_mcp.*`, agent `toolsets`, and agent `deny_tools`; defaults, validation, examples, docs, and tests must ship together.

## Deliverables
- Config structs/defaults/validation for tool registry policy and hosted MCP
- Agent definition grammar for `tools`, `toolsets`, and `deny_tools`
- Updated config examples and validation tests
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for config load/merge behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Empty config loads safe defaults for `[tools]`, policy, hosted MCP, and result limits
  - [ ] Invalid ToolID atoms, invalid toolsets, negative timeouts, and unsafe trusted-source entries fail validation
  - [ ] `deny_tools` overrides parsed allow atoms without requiring policy evaluation in config code
  - [ ] Workspace overlays preserve deterministic precedence for tools config
- Integration tests:
  - [ ] A realistic `config.toml` with tools policy and agent toolsets loads through the same path used by the daemon
  - [ ] Existing config fixtures still reject unknown keys and malformed MCP server config
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Tool registry config is validated before runtime policy consumes it
- No compatibility aliases or deprecated config paths are introduced
