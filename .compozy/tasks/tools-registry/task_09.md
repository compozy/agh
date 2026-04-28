---
status: pending
title: Daemon-Owned MCP Call-Through and Auth Diagnostics
type: backend
complexity: critical
dependencies:
  - task_03
  - task_04
---

# Task 09: Daemon-Owned MCP Call-Through and Auth Diagnostics

## Overview

Add executable `mcp` backend tools by letting the daemon discover and call configured MCP servers through `internal/mcp`. This task reuses existing MCP config and auth storage, preserves remote server metadata, maps redacted auth status to registry availability reasons, and prevents token material from crossing package boundaries.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-005, and ADR-010 before editing MCP behavior
- DO NOT create a second MCP auth store or leak `TokenRecord`, bearer headers, OAuth codes, PKCE verifiers, refresh tokens, or client secrets
- DO NOT convert remote HTTP/SSE MCP servers into blank ACP stdio entries
- TESTS REQUIRED: auth redaction, config preservation, and real/fake MCP call-through must be covered
</critical>

<requirements>
1. MUST implement `MCPCallExecutor` in `internal/mcp`, not in `internal/tools`.
2. MUST preserve configured MCP `Transport`, `URL`, `Auth`, command, args, and env through daemon resource/config projections.
3. MUST map `internal/mcp/auth` status into registry reason codes such as auth unconfigured, required, expired, invalid, and refresh failed.
4. MUST normalize external MCP tools into canonical `mcp__<server>__<tool>` IDs while preserving raw names in `SourceRef`.
5. MUST inject bearer/header material only inside `internal/mcp` and return only redacted diagnostics/results.
6. MUST cover stdio, HTTP, SSE, timeout, cancellation, collision, and auth-required behavior.
</requirements>

## Subtasks
- [ ] 9.1 Fix MCP config/resource cloning so remote metadata and auth config are preserved
- [ ] 9.2 Add redacted registry-facing MCP auth status adapter
- [ ] 9.3 Implement `MCPCallExecutor` list/call behavior inside `internal/mcp`
- [ ] 9.4 Normalize MCP descriptors and collision handling into registry providers
- [ ] 9.5 Add token/redaction guards across errors, logs, events, CLI/API payloads, and test fixtures
- [ ] 9.6 Add MCP fake-server integration tests for discovery, call-through, auth, timeout, and cancellation

## Implementation Details

Use TechSpec "MCP Backend Contract", "MCP Auth/Hosted MCP Existing Surface Alignment", and ADR-010. External MCP backend call-through is distinct from hosted AGH MCP exposure in task_10.

### Relevant Files
- `internal/mcp/auth/types.go` - redacted auth status values
- `internal/mcp/auth/service.go` - existing token lifecycle to reuse without exposing secrets
- `internal/daemon/tool_mcp_resources.go` - existing clone path that must preserve transport, URL, and auth
- `internal/config/provider.go` - MCP server config model
- `internal/config/mcp_resource.go` - MCP resource validation
- `internal/settings/service.go` - existing settings/status surface to keep aligned
- `internal/tools/registry*.go` - MCP provider adapter registration

### Dependent Files
- `internal/api/contract/settings.go` - status parity with existing settings MCP surfaces
- `internal/api/contract/tools.go` - task_11 exposes MCP tool descriptors and auth reasons
- `internal/cli/mcp_auth.go` - existing auth CLI must remain the management path
- `web/src/hooks/routes/use-settings-mcp-servers-page.ts` - task_13 keeps settings diagnostics truthful

### Related ADRs
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - distinguishes remote MCP backend from hosted AGH MCP exposure
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - constrains external tool policy
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - defines daemon-owned MCP backend execution

### Web/Docs Impact
- `web/`: task_13 must align tool diagnostics with existing MCP settings auth status and avoid invented remote login controls.
- `packages/site`: task_14 must update MCP config/auth docs and explain remote MCP call-through versus hosted AGH MCP.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: adds executable MCP backend provider support and source metadata for external tools.
- Agent manageability: existing `agh mcp auth` remains auth management; task_11/task_12 expose registry info/invoke paths.
- Config lifecycle: consumes existing MCP server config and `tools.policy` source grants; fixes preservation of `Transport`, `URL`, and `Auth`.

## Deliverables
- `MCPCallExecutor` inside `internal/mcp`
- MCP descriptor discovery and call-through provider adapter
- Redacted MCP auth status integration with registry availability
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests against fake stdio, HTTP, and SSE MCP servers **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] MCP config clone preserves `Transport`, `URL`, `Auth`, command, args, and env with deep-copy semantics
  - [ ] Auth statuses map exactly to registry availability reason codes without exposing token records
  - [ ] Canonical MCP IDs fail closed on sanitized-name collisions and over-length IDs
  - [ ] Authorization headers are never visible to `internal/tools`, API DTOs, logs, or events
- Integration tests:
  - [ ] Fake stdio MCP server supports `tools/list` and `tools/call` through `Registry.Call`
  - [ ] Fake HTTP/SSE MCP servers preserve transport-specific config and auth behavior
  - [ ] Timeout, cancellation, auth-required, expired, invalid, and refresh-failed paths return deterministic redacted errors
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- MCP backend tools are executable through daemon-owned call-through
- Existing MCP auth/config surfaces remain the source of truth and do not leak secrets
