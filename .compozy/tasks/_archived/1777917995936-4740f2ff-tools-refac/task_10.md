---
status: completed
title: MCP Auth Status and Hosted MCP Projection Parity
type: backend
complexity: critical
dependencies:
    - task_01
    - task_03
    - task_04
    - task_05
    - task_06
    - task_07
    - task_08
    - task_09
---

# Task 10: MCP Auth Status and Hosted MCP Projection Parity

## Overview

Finish the canonical hosted-MCP and auth-diagnostics story by adding an agent-callable auth-status tool and aligning projection/approval behavior with the broadened tool surface. The goal is to reuse the branch's shipped hosted MCP plumbing and redacted auth status model, not to invent a new management or OAuth subsystem.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, and ADR-004 before changing MCP auth or hosted MCP exposure
- REFERENCE TECHSPEC sections "Hosted MCP", "Existing MCP Config And Auth", and "Technical Dependencies"
- FOCUS ON WHAT: expose status diagnostics and keep hosted MCP projections/approval behavior aligned with the expanded registry surface
- MINIMIZE CODE — reuse existing hosted MCP, approval-bridge, and redacted auth status primitives
- TESTS REQUIRED — status, denial, projection, and approval semantics must match CLI/HTTP/UDS/hosted MCP behavior
</critical>

<requirements>
1. MUST add an agent-callable MCP auth status tool using the shipped redacted auth status model.
2. MUST keep MCP auth login/logout on management surfaces only.
3. MUST align hosted MCP session exposure, approval bridging, and projection semantics with the expanded built-in tool surface.
4. MUST preserve strict redaction for OAuth codes, tokens, PKCE material, and callback secrets.
</requirements>

## Subtasks
- [x] 10.1 Add the MCP auth status built-in over the current redacted auth status provider
- [x] 10.2 Align hosted MCP projection logic with the expanded callable tool surface from tasks 03-09
- [x] 10.3 Preserve approval bridge semantics for hosted MCP tool calls and denied operations
- [x] 10.4 Keep CLI/HTTP/UDS login/logout as the repair path surfaced by diagnostics rather than as normal tools
- [x] 10.5 Add unit and integration coverage for status, projection parity, approval, and redaction

## Implementation Details

See TechSpec sections "Hosted MCP", "Existing MCP Config And Auth Lifecycle", and "Implementation Steps". This task is parity and projection work over the hosted MCP foundation already present on this branch.

### Relevant Files
- `internal/tools/mcp.go` — current MCP diagnostics and auth-status reason mapping
- `internal/mcp/auth/service.go` — redacted auth status provider and auth management behavior
- `internal/daemon/hosted_mcp.go` — hosted MCP exposure inside the daemon
- `internal/mcp/hosted.go` — hosted MCP session projection plumbing
- `internal/daemon/tool_approval_bridge.go` — approval bridge semantics that hosted MCP must inherit

### Dependent Files
- `internal/cli/mcp_auth.go` — management-surface repair path that diagnostics should continue to reference
- `internal/api/core/conversions.go` — settings/status payload conversion that must stay aligned with the auth status model

### Related ADRs
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)
- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md)

### Web/Docs Impact
- `web/`: `web/src/generated/agh-openapi.d.ts`, `web/src/systems/settings/adapters/settings-api.ts`, and `web/src/systems/settings/types.ts` if auth-status payloads or diagnostics visible through settings surfaces change.
- `packages/site`: `packages/site/content/runtime/cli-reference/mcp/auth/*`, `packages/site/content/runtime/core/configuration/mcp-json.mdx`, and any hosted-MCP/runtime tool docs added or updated in task_11.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: affects hosted MCP session projection, tool-call approval bridging, and MCP-sidecar diagnostics.
- Agent manageability: gives agents structured auth diagnostics while keeping human-interactive login/logout on management paths.
- Config lifecycle: preserves current MCP server and auth lifecycle keys; docs and examples must explain the status-tool-to-management handoff clearly.

## Deliverables
- MCP auth status built-in using the shipped redacted status model
- Hosted MCP projection and approval parity with the expanded tool surface
- Preserved login/logout management surfaces referenced by diagnostics
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for hosted MCP/auth parity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] auth status tool reuses the current redacted status model and never exposes token material
  - [x] hosted MCP projections include only callable tools while operator views preserve denial diagnostics
  - [x] approval bridge semantics stay consistent for hosted MCP calls that require confirmation
- Integration tests:
  - [x] auth status, hosted MCP projection, and approval behavior agree across tool, CLI, HTTP, UDS, and hosted MCP surfaces
  - [x] login/logout remain management-only and are referenced as repair paths rather than exposed as tool calls
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agents can inspect MCP auth health through tools without pulling browser-driven login/logout into the normal tool loop
- Hosted MCP mirrors the canonical tool surface and approval semantics already established on this branch

## References
- `.compozy/tasks/tools-refac/analysis/competitor-tool-surface-notes.md`
- `.resources/claude-code/commands/mcp/mcp.tsx`
- `.resources/claude-code/services/mcp/auth.ts`
- `.resources/hermes/website/docs/reference/mcp-config-reference.md`
- `.resources/openclaw/docs/cli/mcp.md`
