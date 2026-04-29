# ADR-011: Use `mark3labs/mcp-go` For MCP Protocol And Transport

## Status

Accepted

## Date

2026-04-28

## Context

The Tool Registry TechSpec currently describes hosted MCP exposure and daemon-owned MCP call-through in a way that leaves room for AGH to implement MCP protocol/session/transport behavior manually.

That would duplicate a large amount of MCP protocol code that already exists in `github.com/mark3labs/mcp-go`:

- client/server lifecycle
- stdio transport
- streamable HTTP transport
- SSE transport
- tool registration
- `tools/list_changed`
- session-scoped tools/filtering
- raw input/output schema fields

At the same time, AGH already owns product-specific MCP boundaries that the library should not replace:

- `internal/mcp/auth` durable token storage and redacted status
- CLI/settings/API operator surfaces
- registry projection, policy, approval, lineage, provenance, and redaction
- hosted MCP UDS bind rules and session scoping

## Decision

AGH will use `github.com/mark3labs/mcp-go` as the MCP protocol and transport implementation for the Tool Registry MVP.

Adoption boundary:

- Hosted AGH MCP proxy uses `server.NewMCPServer`, `mcp.Tool`, and `server.ServeStdio` rather than manual MCP message handling.
- Daemon-owned MCP call-through uses the library client/transport APIs for stdio, streamable HTTP, and SSE MCP servers.
- AGH keeps `internal/mcp/auth` as the durable auth owner. Library OAuth helpers are not the product authority.
- AGH keeps registry semantics, policy, approval, projection, redaction, provenance, diagnostics, and canonical `ToolID` semantics outside the library.

AGH canonical `ToolID` remains the public identity and is passed to `mcp-go` as MCP `Tool.name`.

## Alternatives Considered

### Continue with hand-rolled MCP implementation

- **Pros**: maximum internal control and no external library dependency.
- **Cons**: duplicates already-solved protocol work, increases bug surface for lifecycle/transports/auth wiring, and delays registry delivery.
- **Why rejected**: `mcp-go` already covers the protocol layer AGH needs.

### Replace AGH auth ownership with library OAuth ownership

- **Pros**: more reuse around OAuth.
- **Cons**: loses AGH's current durable token store, redacted status model, and operator manageability surfaces.
- **Why rejected**: AGH auth is a product boundary, not just a transport detail.

### Use the official MCP Go SDK instead of `mcp-go`

- **Pros**: tighter governance alignment with the MCP spec source of truth.
- **Cons**: for this workstream it forced an unnecessary hard-cut of remote `sse`, gave AGH a less natural auth integration surface, and fit hosted session-scoped tool projection less directly.
- **Why rejected**: `mcp-go` matches AGH's practical hosted-proxy and call-through needs better while still preserving AGH-owned policy/auth boundaries.

## Consequences

### Positive

- AGH reuses an existing MCP protocol/session/transport implementation instead of rebuilding it.
- Hosted MCP and remote MCP call-through gain first-class stdio, streamable HTTP, and SSE support.
- The library's per-session tools and `tools/list_changed` behavior fit AGH's hosted projection model well.
- Raw input/output schema fields give AGH a direct way to preserve descriptor-authored hosted schemas.
- The Tool Registry TechSpec can focus on AGH-specific semantics instead of rebuilding MCP itself.

### Negative

- AGH now depends on an evolving external library and must track version compatibility explicitly.
- Some AGH requirements still require wrapper code because the library is protocol-level, not product-level.
- The library ships OAuth helpers that default to in-memory token storage, which conflicts with AGH's auth ownership if used naively.

### Risks

- Library behavior changes could leak into AGH MCP behavior.
  - Mitigation: pin the module version, wrap usage behind `internal/mcp`, and test hosted/stdio/HTTP/SSE behavior explicitly.
- AGH developers may overreach and replace auth/product boundaries with library OAuth defaults.
  - Mitigation: keep `internal/mcp/auth` authoritative in the TechSpec and tests, and explicitly forbid `MemoryTokenStore` or library-owned login flows as product authority.
- Remote schema fidelity could be overstated if AGH assumes byte-for-byte preservation of schemas discovered from upstream servers.
  - Mitigation: keep hosted schemas exact through raw fields, and define external schema preservation in terms of raw bytes when available or canonicalized decoded schema otherwise.

## Implementation Notes

- Add `github.com/mark3labs/mcp-go` through `go get`; do not edit `go.mod` by hand.
- Hosted MCP stdio proxy should use `server.NewMCPServer` plus `server.ServeStdio`, with explicit `mcp.Tool` values carrying descriptor-owned `RawInputSchema` and `RawOutputSchema` bytes.
- MCP call-through should use `client.NewStdioMCPClient` for stdio servers, `client.NewStreamableHttpClient` for remote HTTP MCP servers, and `client.NewSSEMCPClient` for remote SSE MCP servers.
- AGH keeps the existing `mcp_server.transport = "stdio" | "http" | "sse"` surface. `http` maps to streamable HTTP; `sse` maps to the library's explicit SSE client path.
- MVP auth integration should use AGH-owned token state plus header injection or a narrowly-scoped AGH-owned adapter inside `internal/mcp`; it must not rely on `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, default `transport.NewOAuthHandler`, or `MemoryTokenStore` as the product authority.
- Any refresh behavior triggered during outbound MCP calls may attempt at most one `internal/mcp/auth.Service.Refresh` and must never start a new login flow outside `agh mcp auth login`.
- Upstream `notifications/tools/list_changed` may be used as cache invalidation hints only in MVP; AGH does not let external notifications mutate registry structure directly.

## References

- `.compozy/tasks/tools-registry/analysis/analysis_mark3labs_mcp_go.md`
- `https://github.com/mark3labs/mcp-go`
- `https://github.com/mark3labs/mcp-go/releases`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/README.md`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/mcp/tools.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/server.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/server/streamable_http.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/stdio.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/http.go`
- `/Users/pedronauck/go/pkg/mod/github.com/mark3labs/mcp-go@v0.49.0/client/sse.go`
