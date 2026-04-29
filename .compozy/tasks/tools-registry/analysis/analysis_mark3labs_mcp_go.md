# Analysis: `mark3labs/mcp-go` Adoption

## Scope

This analysis evaluates whether the Tool Registry foundation should implement MCP protocol/server/client behavior manually inside AGH, adopt the official Go SDK, or adopt `github.com/mark3labs/mcp-go` while keeping AGH-specific policy/auth/manageability logic above the protocol layer.

The research uses primary sources only:

- repository and releases: `github.com/mark3labs/mcp-go`
- locally downloaded module source resolved by `go list -m -json github.com/mark3labs/mcp-go@latest`
- official Go SDK repository used only for comparison: `github.com/modelcontextprotocol/go-sdk`
- current AGH MCP code under `internal/mcp`, `internal/config`, `internal/acp`, and daemon MCP resource publication paths

## Executive Conclusion

AGH should adopt `github.com/mark3labs/mcp-go`, specifically module version `v0.49.0` released on April 21, 2026, for MCP protocol/session/transport mechanics instead of planning a hand-rolled MCP implementation in the Tool Registry workstream.

Compared with the official Go SDK, `mcp-go` fits AGH's hosted-proxy and daemon-owned call-through needs better in three places:

- it has first-class `stdio`, `streamable HTTP`, and `SSE` client surfaces;
- it has direct hosted-server ergonomics for session-scoped tool mutation and `tools/list_changed`;
- it exposes `RawInputSchema` and `RawOutputSchema` on `mcp.Tool`, which is a better match for AGH's descriptor-authored hosted schemas.

This does **not** mean handing MCP ownership to the SDK wholesale. AGH must keep these boundaries:

- `internal/tools` remains the registry/policy/provenance/decision owner.
- `internal/mcp/auth` remains the durable OAuth + redacted-status owner.
- `internal/daemon` remains the composition root and hosted-MCP session binder.
- AGH-specific `ToolID`, approval bridge, UDS peer validation, reason codes, redaction rules, and manageability surfaces remain AGH-owned.

The library should replace **manual MCP protocol plumbing**, not AGH policy or AGH auth/state lifecycles.

## What `mark3labs/mcp-go` Already Covers

### 1. Server and client lifecycle

`mcp-go` already exposes the main shapes AGH needs for hosted MCP and remote call-through:

- hosted server creation through `server.NewMCPServer(...)`;
- concrete client lifecycle through `client.NewClient(...)`, `Initialize(...)`, request methods, and `Close()`;
- built-in session-aware server features such as session-specific tools and list-change signaling.

This is a better fit than a custom JSON-RPC layer because the library already maps MCP lifecycle behavior into those abstractions.

### 2. Stdio, streamable HTTP, and SSE transports

The library already supports:

- `server.ServeStdio(...)` and `client.NewStdioMCPClient(...)`;
- `server.NewStreamableHTTPServer(...)` and `client.NewStreamableHttpClient(...)`;
- `server.NewSSEServer(...)` and `client.NewSSEMCPClient(...)`.

That maps directly to AGH's two main MCP paths:

- hosted AGH MCP proxy over stdio for ACP session injection;
- daemon-owned MCP clients for configured stdio, remote HTTP, and remote SSE MCP servers.

Important scope implication for AGH: the library has a distinct SSE client path. That means the prior hard-cut of `mcp_server.transport = "sse"` was an artifact of the official SDK choice, not a product requirement. If AGH wants to preserve its current MCP config surface, `mcp-go` supports that direction directly.

### 3. Tool registration, schema validation, and `tools/list_changed`

The library already supports server-side tool registration and client-side tool calling:

- `server.AddTool` / `AddTools` / `SetTools` / `DeleteTools`
- `client.ListTools`
- `client.CallTool`
- automatic `notifications/tools/list_changed`
- session-specific tools through `AddSessionTool` / `AddSessionTools`

The library also exposes raw schema fields on `mcp.Tool`:

- `RawInputSchema`
- `RawOutputSchema`

That removes a large amount of otherwise error-prone manual glue in AGH's hosted MCP proxy, where descriptor-owned schema bytes should remain authoritative.

### 4. Optional OAuth helpers, but not auth ownership

The library ships client-side OAuth helpers:

- `client.NewOAuthStreamableHttpClient(...)`
- `client.NewOAuthSSEClient(...)`
- `transport.OAuthConfig`
- `transport.TokenStore`
- `transport.OAuthHandler`

However, `transport.NewOAuthHandler` defaults to `MemoryTokenStore` if no store is supplied. That is not compatible with AGH's current auth ownership model.

## What AGH Must Keep Owning

### 1. Durable MCP auth lifecycle

AGH already has real MCP auth product surfaces:

- `internal/mcp/auth`
- `internal/store/globaldb` token persistence
- `agh mcp auth login/status/logout`
- settings/API redacted `auth_status`

The library's OAuth helpers are useful protocol helpers, but they do not replace AGH's persistent token store, operator workflows, or redacted diagnostics. The right design is:

- keep AGH auth state authoritative;
- prefer AGH-owned header injection or a narrow AGH-owned adapter inside `internal/mcp`;
- never let `MemoryTokenStore`, library-owned login flows, or library-owned refresh persistence become the authority.

### 2. Registry policy, provenance, and reason codes

The library is intentionally protocol-level. It does not implement AGH-specific:

- canonical `ToolID` policy;
- operator/session projection split;
- source trust rules;
- lineage subset enforcement;
- approval bridge semantics;
- AGH reason-code taxonomy;
- result redaction and cross-surface consistency requirements.

Those remain in AGH's registry layer.

### 3. Hosted MCP bind/authentication model

AGH's hosted MCP server is not just an MCP server. It is a daemon-scoped exposure transport bound to:

- a session,
- a workspace,
- a UDS peer,
- an expected AGH binary,
- a daemon-minted bind nonce,
- AGH approval semantics.

The library can run the MCP protocol over stdio, but AGH still owns the pre-bind/bind rules and the UDS hop into `Registry.Call`.

## Relevant Library Limits / Rough Edges

### 1. Remote schema fidelity is not byte-exact by default

Hosted MCP can preserve exact descriptor bytes because AGH can populate `RawInputSchema` and `RawOutputSchema` directly.

External MCP discovery is more nuanced. The client path decodes `tools/list` into typed `mcp.Tool` values. When upstream schemas arrive only through structured fields, AGH should treat a canonical JSON encoding of the decoded schema as authoritative rather than promising byte-for-byte preservation of the original upstream payload.

This is not a blocker, but it should be stated accurately in the TechSpec.

### 2. Library OAuth is not the AGH operator surface

The library can perform OAuth-oriented transport setup, but AGH needs:

- durable storage,
- redacted diagnostics,
- CLI/settings/API manageability,
- explicit ownership boundaries.

Therefore the library OAuth package should be considered optional or adapter-facing, not the new source of truth.

### 3. Streamable HTTP resumability is not implemented

The streamable HTTP implementation explicitly does not support stream resumability on client or server.

That matters for any future AGH long-lived remote listening path, but it is not a blocker for the MVP hosted stdio path or on-demand remote discovery/call path.

### 4. Continuous listening should remain optional in MVP

The streamable HTTP client can enable standalone GET listening through `WithContinuousListening()`. AGH does not need that as a hard dependency for MVP. If enabled later, external `notifications/tools/list_changed` should still be treated only as cache invalidation hints, not as authoritative registry mutations.

## Recommendation

Adopt `mcp-go` in these exact places:

1. Hosted AGH MCP stdio proxy:
   - use `server.NewMCPServer`
   - register session-callable registry tools as `mcp.Tool` values with `RawInputSchema` / `RawOutputSchema`
   - serve over `server.ServeStdio`

2. Daemon-owned MCP clients for configured servers:
   - stdio MCP servers via `client.NewStdioMCPClient`
   - remote HTTP MCP servers via `client.NewStreamableHttpClient`
   - remote SSE MCP servers via `client.NewSSEMCPClient`

3. Keep AGH auth as the durable owner:
   - `internal/mcp/auth` stays authoritative
   - outbound requests use AGH-owned auth state, with header injection or a narrow internal adapter
   - avoid `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, default `transport.NewOAuthHandler`, and `MemoryTokenStore` as product authority

4. Keep AGH registry semantics outside the SDK:
   - projection, policy, provenance, approval, hook dispatch, telemetry, redaction, and reason codes stay AGH-owned

## TechSpec Implications

The TechSpec should stop implying a hand-rolled MCP protocol server/client stack. It should instead state:

- AGH uses `github.com/mark3labs/mcp-go` for MCP protocol lifecycle, tools, and stdio/HTTP/SSE transports.
- AGH wraps the library inside `internal/mcp` and `agh tool mcp`; the library is not exposed as the policy authority.
- AGH keeps durable auth in `internal/mcp/auth`.
- AGH canonical `ToolID` remains the hosted MCP `Tool.name`.
- Hosted schemas are exact descriptor bytes; external discovered schemas are preserved as raw bytes when available or canonicalized decoded schema otherwise.
- Remote `sse` remains a valid MCP transport in config and resources when the source actually declares SSE.

## Evidence

External primary sources:

- `https://github.com/mark3labs/mcp-go`
- `https://github.com/mark3labs/mcp-go/releases`
- `https://github.com/modelcontextprotocol/go-sdk`

Local module source:

- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/README.md`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/client.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/http.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/sse.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/stdio.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/transport/oauth.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/transport/streamable_http.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/mcp/tools.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/server.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/session.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/streamable_http.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/sse.go`

Local AGH evidence:

- `internal/mcp/auth`
- `internal/config/provider.go`
- `internal/acp/client.go`
- `internal/daemon/tool_mcp_resources.go`
- `.compozy/tasks/tools-registry/_techspec.md`
