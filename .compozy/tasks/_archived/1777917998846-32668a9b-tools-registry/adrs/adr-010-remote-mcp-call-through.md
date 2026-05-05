# ADR-010: Remote MCP Call-Through In MVP

## Status

Accepted

## Date

2026-04-28

## Context

AGH already models MCP server configuration and remote auth. The previous Tool Registry spec treated MCP-backed tools as descriptors with availability diagnostics only. The revised MVP must make MCP tools executable through the same registry dispatch path as built-in and extension-host tools.

Remote MCP call-through must not duplicate AGH's MCP auth model or leak OAuth tokens. ACP currently converts MCP servers to stdio-only session entries, so remote MCP execution should happen inside the daemon-owned registry adapter rather than by passing remote MCP config directly through ACP.

## Decision

Remote/local MCP tools are executable in the Tool Registry MVP.

The daemon owns MCP client adapters that discover/list/call MCP tools from validated MCP configuration and resource sources. The adapters consume existing MCP config, transport, auth metadata, and redacted auth status from `internal/mcp/auth`. Token material remains owned by `internal/mcp/auth` and its `TokenStore`; registry descriptors and results never copy tokens. In this workstream, remote MCP transport support preserves `stdio`, streamable `http`, and declarative `sse`, because `mark3labs/mcp-go` supports all three directly.

The call-through contract is an `MCPCallExecutor` implemented inside `internal/mcp`. `internal/tools` may depend on that interface, but it must not import `internal/mcp/auth`, open the token store, receive raw bearer strings, or construct Authorization headers. The executor resolves bearer material internally for each outbound request and returns only normalized results plus redacted errors.

Hosted MCP remains AGH's session exposure transport. When an agent calls `mcp__...` through the hosted MCP server, the call re-enters `internal/tools.Registry.Call`; the registry then invokes the daemon-owned MCP client adapter after policy, auth, availability, hook, schema, and result-budget checks.

## Alternatives Considered

### Descriptor-only MCP tools

- **Description**: Show MCP tool descriptors and auth diagnostics, but do not call them in MVP.
- **Pros**: Smaller security and transport surface.
- **Cons**: Leaves MCP as a second-class source and fails to prove external tool execution.
- **Why rejected**: The accepted MVP scope includes remote MCP call-through.

### Pass remote MCP servers directly to ACP sessions

- **Description**: Let providers connect directly to remote MCP servers.
- **Pros**: Less daemon adapter work.
- **Cons**: ACP conversion is currently stdio-only, policy/audit is provider-dependent, and AGH cannot centrally enforce result redaction or source grants.
- **Why rejected**: AGH needs one daemon-owned dispatch path.

### Duplicate MCP auth in the registry

- **Description**: Store MCP tokens or OAuth state with tool descriptors.
- **Pros**: Simple adapter lookup.
- **Cons**: Duplicates credential ownership and increases leak risk.
- **Why rejected**: `internal/mcp/auth` remains the sole credential owner.

## Consequences

### Positive

- MCP tools become agent-callable through the same policy, visibility, hook, telemetry, and hosted MCP surfaces as built-ins and extension-host tools.
- Existing MCP auth and settings diagnostics remain authoritative.
- AGH can enforce a consistent `ToolID` and collision policy for MCP sources.
- AGH no longer needs to hard-cut remote `sse` purely because of the protocol library choice.

### Negative

- MVP must implement daemon-side MCP discovery/call clients, transport handling, timeout behavior, auth refresh/error mapping, and redaction tests.
- MCP adapters add more failure states to availability and session projection.

### Risks

- OAuth tokens could leak through registry output. Mitigation: registry consumes only redacted status and uses narrow `internal/mcp/auth` execution interfaces for bearer material.
- Remote MCP call latency could block hosted MCP responses. Mitigation: explicit timeouts, cancellation, and structured backend failure errors.
- External MCP tool names could collide after sanitization. Mitigation: fail-closed conflict handling and operator-visible diagnostics.

## Implementation Notes

- Reuse `aghconfig.MCPServer`, `internal/config/mcpjson.go`, `internal/config/mcp_resource.go`, skill MCP resolution, extension MCP resources, and `internal/mcp/auth`.
- Add `MCPCallExecutor` tests proving bearer headers are injected only inside `internal/mcp` and never cross `internal/tools` logs, errors, events, or results.
- Fix resource cloning paths that currently drop `Transport`, `URL`, or `Auth` before relying on remote MCP diagnostics/calls.
- Add MCP adapter tests for stdio, HTTP, SSE, auth-required, expired/invalid auth, collision, timeout, cancellation, redaction, and transport-specific config preservation.
- Use `client.NewStdioMCPClient`, `client.NewStreamableHttpClient`, and `client.NewSSEMCPClient` from `mark3labs/mcp-go`.
- Keep upstream `notifications/tools/list_changed` as cache invalidation hints only in MVP; do not let remote notifications mutate registry state directly.
- Hosted MCP never receives remote OAuth tokens; it receives only AGH-hosted session projection entries.

## References

- `internal/config/provider.go`
- `internal/mcp/auth`
- `internal/store/globaldb/global_db_mcp_auth.go`
- `internal/acp/client.go`
- `.compozy/tasks/tools-registry/analysis/analysis_acp_tool_registry_compatibility.md`
