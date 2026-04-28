---
status: pending
title: Hosted AGH MCP Session Exposure and Approval Bridge
type: backend
complexity: critical
dependencies:
  - task_05
  - task_09
---

# Task 10: Hosted AGH MCP Session Exposure and Approval Bridge

## Overview

Expose AGH registry tools to ACP sessions through a session-bound hosted MCP proxy. This task adds the local MCP bind lifecycle, ACP stdio injection, UDS peer and AGH binary validation, and approval bridge behavior while ensuring every hosted MCP call re-enters the registry dispatch pipeline.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-005, and ADR-010 before implementing hosted MCP
- DO NOT treat the bind nonce as bearer auth; it is a correlation value plus UDS peer/binary validation
- DO NOT accept client-supplied approval tokens over hosted MCP
- TESTS REQUIRED: bind failures, approval timeout/cancel/unreachable, and ACP `mcpServers` injection must be deterministic
</critical>

<requirements>
1. MUST add `agh tool mcp --session <id> --bind-nonce <nonce>` as the hosted MCP stdio entrypoint used by ACP.
2. MUST mint session-bound bind nonces with TTL, single-use behavior, and redacted diagnostics.
3. MUST validate UDS peer credentials and expected AGH binary before accepting hosted MCP binds; unsupported validation must fail closed.
4. MUST ensure hosted MCP `tools/list` equals the effective session-callable projection.
5. MUST route hosted MCP `tools/call` through `Registry.Call` and existing ACP permission/approval paths.
6. MUST return deterministic `approval_unreachable`, `approval_timed_out`, and `approval_canceled` errors.
</requirements>

## Subtasks
- [ ] 10.1 Add hosted MCP launch record, bind nonce lifecycle, and redacted diagnostics
- [ ] 10.2 Implement `agh tool mcp --session --bind-nonce` proxy entrypoint
- [ ] 10.3 Inject only the AGH-hosted stdio MCP entry into ACP session start/load payloads
- [ ] 10.4 Validate UDS peer credentials and expected AGH binary, failing closed when unavailable
- [ ] 10.5 Bridge hosted MCP approval-required calls to existing ACP/session permission flow
- [ ] 10.6 Add acpmock/runtime tests for hosted MCP list/call, bind failure, approval timeout, cancellation, and disconnect

## Implementation Details

Use TechSpec "Session Tool Exposure", "Hosted MCP Bind Contract", "Approval Bridge", and ADR-002. Hosted MCP is an exposure transport for AGH registry tools, not the external MCP backend implemented in task_09.

### Relevant Files
- `internal/acp/client.go` - ACP `mcpServers` conversion/injection boundary
- `internal/acp/permission.go` - ACP permission mode and approval ceiling
- `internal/session/manager_start.go` - session start wiring
- `internal/session/manager_prompt.go` - existing approval route
- `internal/mcp/**` - hosted MCP proxy implementation
- `internal/cli/**` - `agh tool mcp` command entrypoint
- `internal/testutil/acpmock/**` - fixture support for ACP `mcpServers` and approval assertions

### Dependent Files
- `internal/api/contract/tools.go` - task_11 exposes session projection and invoke semantics
- `internal/cli/tool*.go` - task_12 exposes user-facing command shape
- `web/src/systems/session/**` - task_13 may display hosted tool call/projection state
- `packages/site/content/runtime/core/sessions/permissions.mdx` - task_14 documents approval bridge behavior

### Related ADRs
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - defines AGH-hosted MCP as session exposure
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - defines approval bridge and timeout behavior
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - separates remote MCP backend from hosted AGH MCP

### Web/Docs Impact
- `web/`: task_13 must reflect session-callable tools and approval-required/unavailable states only when backed by API contracts.
- `packages/site`: task_14 must document hosted MCP threat model, bind lifecycle, approval bridge, ACP injection, and failure modes.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: exposes all allowed registry tools to ACP-compatible agents through a hosted MCP transport.
- Agent manageability: adds `agh tool mcp` internal command path and session projection behavior consumed by agents.
- Config lifecycle: consumes `[tools.hosted_mcp]` and `[tools.policy].approval_timeout_seconds` from task_02.

## Deliverables
- Hosted AGH MCP proxy and bind lifecycle
- ACP session MCP injection that does not misrepresent remote MCP servers
- Approval bridge and deterministic timeout/cancel/unreachable errors
- Unit tests with 80%+ coverage **(REQUIRED)**
- Runtime/acpmock integration tests **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Bind nonce is session-bound, single-use, expires deterministically, and is redacted from logs/errors
  - [ ] UDS peer credential or expected binary validation failure rejects the bind
  - [ ] `toSDKMCPServers` or replacement injection never converts remote HTTP/SSE MCP servers into blank stdio entries
  - [ ] Approval timeout, cancellation, and unreachable approval channel return stable reason codes
- Integration tests:
  - [ ] acpmock observes the AGH-hosted MCP entry during session start/load
  - [ ] Hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools` once task_11 lands, or the equivalent internal projection before routes exist
  - [ ] Hosted MCP safe built-in call succeeds and mutating call routes to ACP permission request
  - [ ] Proxy disconnect cancels in-flight tool calls without leaving stale bind records
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- ACP sessions receive AGH registry tools through hosted MCP only
- Hosted MCP calls cannot bypass registry dispatch, policy, approval, hooks, or redaction
