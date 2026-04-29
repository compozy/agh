---
status: completed
title: Daemon-Owned MCP Call-Through and Auth Diagnostics
type: backend
complexity: critical
dependencies:
  - task_03
  - task_04
---

# Task 09: Daemon-Owned MCP Call-Through and Auth Diagnostics

## Overview

Add executable `mcp` backend tools by letting the daemon discover and call configured MCP servers through `internal/mcp`. This task reuses existing MCP config and auth storage, preserves remote server metadata, normalizes upstream MCP descriptors including `output_schema`, maps redacted auth status to registry availability reasons, and prevents token material from crossing package boundaries.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-005, ADR-010, and ADR-011 before editing MCP behavior
- DO NOT create a second MCP auth store or leak `TokenRecord`, bearer headers, OAuth codes, PKCE verifiers, refresh tokens, or client secrets
- DO NOT use `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, `transport.NewOAuthHandler`, `MemoryTokenStore`, or any library-managed login/cache/refresh path as the authority for remote MCP credentials
- DO NOT convert remote HTTP MCP servers into blank ACP stdio entries
- DO NOT silently reinterpret `mcp_server.transport = "sse"` as `http`, or erase `Transport`/`URL`/`Auth` during config/resource projection
- DO NOT introduce hand-rolled MCP JSON-RPC framing, transport, or auth flow outside the `mark3labs/mcp-go` wrapper
- TESTS REQUIRED: auth redaction, config preservation, and real/fake MCP call-through must be covered
</critical>

<requirements>
1. MUST implement `MCPCallExecutor` in `internal/mcp`, not in `internal/tools`.
2. MUST preserve configured MCP `Transport`, `URL`, `Auth`, command, args, and env through daemon resource/config projections.
3. MUST map `internal/mcp/auth` status into registry reason codes such as auth unconfigured, required, expired, invalid, and refresh failed.
4. MUST normalize external MCP tools into canonical `mcp__<server>__<tool>` IDs while preserving raw names in `SourceRef` and carrying both `input_schema` and `output_schema` through the daemon-internal `MCPToolDescriptor`.
5. MUST inject bearer/header material only inside `internal/mcp` and return only redacted diagnostics/results.
6. MUST cover stdio, HTTP, SSE, timeout, cancellation, collision, and auth-required behavior.
</requirements>

## Subtasks
- [x] 9.1 Fix MCP config/resource cloning so remote metadata and auth config are preserved
- [x] 9.2 Add redacted registry-facing MCP auth status adapter
- [x] 9.3 Implement `MCPCallExecutor` list/call behavior inside `internal/mcp` using `client.NewStdioMCPClient` for stdio, `client.NewStreamableHttpClient` for remote HTTP, and `client.NewSSEMCPClient` for remote SSE
- [x] 9.4 Normalize MCP descriptors, `output_schema` preservation, and collision handling into registry providers
- [x] 9.5 Add token/redaction guards across errors, logs, events, CLI/API payloads, and test fixtures
- [x] 9.6 Add MCP fake-server integration tests for discovery, call-through, auth, timeout, cancellation, and cache-invalidation notifications

## Implementation Details

Use TechSpec "MCP Backend Contract", "MCP Auth/Hosted MCP Existing Surface Alignment", "MCP Library Adoption", and ADR-010/ADR-011. External MCP backend call-through is distinct from hosted AGH MCP exposure in task_10.

### Relevant Files
- `internal/mcp/auth/types.go` - redacted auth status values
- `internal/mcp/auth/service.go` - existing token lifecycle to reuse without exposing secrets
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client` - client/session/transport entry points
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/transport` - streamable HTTP, SSE, header injection, and optional OAuth helper surfaces
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/oauth.go` - helper constructors that must not become AGH auth authority
- `internal/daemon/tool_mcp_resources.go` - existing clone path that must preserve transport, URL, and auth
- `internal/config/provider.go` - MCP server config model
- `internal/config/mcp_resource.go` - MCP resource validation
- `internal/settings/service.go` - existing settings/status surface to keep aligned
- `internal/tools/tool.go` - `MCPToolDescriptor` contract that must carry normalized MCP `output_schema`
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
- [ADR-011: Use `mark3labs/mcp-go` For MCP Protocol And Transport](adrs/adr-011-mark3labs-mcp-go.md) - requires library client transports instead of hand-rolled MCP plumbing

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
- `MCPToolDescriptor` contract update for normalized `output_schema`
- Redacted MCP auth status integration with registry availability
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests against fake stdio, HTTP, and SSE MCP servers **(REQUIRED)**

## Tests
- Unit tests:
  - [x] MCP config clone preserves `Transport`, `URL`, `Auth`, command, args, and env with deep-copy semantics
  - [x] Auth statuses map exactly to registry availability reason codes without exposing token records
  - [x] Canonical MCP IDs fail closed on sanitized-name collisions and over-length IDs
  - [x] Shared `Canonicalize(rawServer, rawTool)` fixtures prove byte-stable MCP ID normalization and reject unsanitizable upstream names
  - [x] `MCPToolDescriptor` preserves normalized `output_schema` for external discovery, using raw schema bytes when surfaced directly by the library and one canonical JSON encoding otherwise
  - [x] Authorization headers are never visible to `internal/tools`, API DTOs, logs, or events
  - [x] `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, `transport.NewOAuthHandler`, and `MemoryTokenStore` are never instantiated as AGH auth authority in the executor path
  - [x] Remote HTTP and SSE auth injection stays inside `internal/mcp`, attempts at most one `internal/mcp/auth.Service.Refresh`, never starts a new login flow, and returns deterministic redacted reason codes on failure
  - [x] Remote HTTP transport uses the explicit streamable HTTP client path and remote SSE uses the explicit SSE client path; neither is silently rewritten into the other
- Integration tests:
  - [x] Fake stdio MCP server supports `tools/list` and `tools/call` through `Registry.Call`
  - [x] Fake HTTP MCP server preserves transport-specific config and auth behavior
  - [x] Fake SSE MCP server preserves transport-specific config and auth behavior
  - [x] Discovered MCP tools synthesize registry descriptors whose `input_schema` and `output_schema` match the normalized daemon-internal descriptor contract
  - [x] Timeout, cancellation, auth-required, expired, invalid, and refresh-failed paths return deterministic redacted errors
  - [x] Remote MCP descriptor refresh remains on-demand, treats upstream `tools/list_changed` only as cache invalidation, and does not rely on standalone notification subscriptions in MVP even if a later transport session enables listening
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- MCP backend tools are executable through daemon-owned call-through
- Existing MCP auth/config surfaces remain the source of truth and do not leak secrets
