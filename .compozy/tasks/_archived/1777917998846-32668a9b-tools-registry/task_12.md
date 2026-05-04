---
status: completed
title: CLI Operator Commands
type: backend
complexity: high
dependencies:
  - task_11
---

# Task 12: CLI Operator Commands

## Overview

Add agent-operable CLI surfaces for registry inspection and invocation. This task implements structured `agh tool` and `agh toolsets` commands backed by the same UDS/HTTP contracts as task_11 and preserves existing `agh mcp auth` as the MCP authentication management path.

<critical>
- ALWAYS READ `_techspec.md`, ADR-005, ADR-006, ADR-007, and CLI docs rules before adding commands
- DO NOT duplicate MCP auth login/logout/status under `agh tool`; existing `agh mcp auth` remains authoritative
- DO NOT print secrets, raw bind nonces, approval tokens, OAuth material, or unredacted sensitive inputs
- TESTS REQUIRED: every command needs JSON output tests and deterministic error-body assertions
</critical>

<requirements>
1. MUST add `agh tool list`, `agh tool search`, `agh tool info`, and `agh tool invoke`.
2. MUST add `agh toolsets list` and `agh toolsets info`.
3. MUST support structured output modes needed by agents, including JSON where applicable.
4. MUST validate input JSON files/stdin for invoke commands before sending requests.
5. MUST return deterministic non-zero exits and structured errors for denied, unavailable, conflicted, auth-required, approval, and schema-validation failures.
6. MUST regenerate CLI docs in task_14 and include tests that detect missing generated docs.
</requirements>

## Subtasks
- [x] 12.1 Add CLI command registration for `tool` and `toolsets`
- [x] 12.2 Add UDS/HTTP client methods for list/search/info/invoke/toolsets
- [x] 12.3 Add JSON/text output rendering with redaction and deterministic errors
- [x] 12.4 Add input JSON validation and stdin/file support for invocation
- [x] 12.5 Add command tests, snapshots, and error-path coverage
- [x] 12.6 Mark CLI docs regeneration requirements for task_14

## Implementation Details

Use TechSpec "Agent Manageability" and "Implementation Steps" 14 and 16. CLI commands should be thin clients over the contracts from task_11, not a second registry implementation.

### Relevant Files
- `internal/cli/root.go` - command registration
- `internal/cli/client.go` - CLI client methods
- `internal/cli/tool*.go` - new tool and toolset commands
- `internal/cli/mcp_auth.go` - existing MCP auth command to preserve
- `internal/api/contract/tools.go` - DTOs consumed by CLI rendering
- `packages/site/content/runtime/cli-reference/**` - generated docs target for task_14

### Dependent Files
- `internal/cli/*_test.go` - command and output tests
- `packages/site/content/runtime/core/tools.mdx` - task_14 references CLI examples
- `web/src/systems/tools/**` - task_13 may mirror CLI-visible states in UI

### Related ADRs
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - CLI invoke cannot bypass approval/policy
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - CLI operator surfaces can show diagnostic states
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - CLI accepts canonical ToolID only

### Web/Docs Impact
- `web/`: no direct code impact - checked systems; web consumes API contracts from task_11 in task_13.
- `packages/site`: task_14 must run `make cli-docs` and document `agh tool`, `agh toolsets`, and the relationship to `agh mcp auth`.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: CLI exposes extension-host and MCP tool diagnostics using the same descriptor/reason model as native tools.
- Agent manageability: adds primary local agent-operable commands for list, search, info, invoke, and toolset inspection with structured output.
- Config lifecycle: no new keys; CLI reflects config and policy from earlier tasks.

## Deliverables
- `agh tool` and `agh toolsets` command families
- Structured JSON/text output and deterministic error handling
- CLI tests and snapshots
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests against daemon UDS/HTTP contracts **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `agh tool list -o json` renders canonical IDs, backend kind, availability, and redacted diagnostics
  - [x] `agh tool info <tool_id> -o json` rejects invalid IDs and unavailable tools with structured errors
  - [x] `agh tool invoke` validates JSON input and redacts sensitive result/error fields
  - [x] `agh toolsets list/info` renders expanded and unavailable members deterministically
- Integration tests:
  - [x] CLI output matches HTTP/UDS payloads for the same daemon state
  - [x] Existing `agh mcp auth status --refresh -o json` remains the auth-management path and agrees with registry diagnostics
  - [x] Generated CLI docs include new command pages after task_14 runs `make cli-docs`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agents can manage and invoke registry tools through structured CLI commands
- CLI does not duplicate auth stores or bypass registry policy/approval
