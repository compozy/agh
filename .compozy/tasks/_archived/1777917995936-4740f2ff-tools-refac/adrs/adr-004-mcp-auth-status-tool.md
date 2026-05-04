# ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces

## Status

Accepted

## Date

2026-04-29

## Context

Agents need enough visibility to understand why an MCP-backed tool is
unavailable, but OAuth-based MCP login/logout flows are not normal runtime tool
calls. They involve browser redirects, callback handling, secure token
persistence, re-authorization, and explicit operator trust.

The initial `tools-registry` foundation treated `agh mcp auth login/status/logout`
as the existing management path. Follow-up discussion asked whether the final
surface should convert those flows into agent-callable tools. Competitor review
showed a consistent separation:

This branch already ships the operator-management CLI path and the redacted MCP
auth status plumbing used by registry diagnostics. The follow-up work is to add
an AGH-owned built-in status wrapper, not to invent a new auth subsystem.

- Claude Code routes re-auth to `/mcp` management UI/commands.
- Hermes handles MCP OAuth through config plus first-connect browser auth and
  `/reload-mcp`.
- OpenClaw keeps MCP management on `openclaw mcp` command surfaces.

The user chose the middle path: agents should get structured status visibility,
but login/logout remain operator management flows.

## Decision

AGH exposes MCP auth status to agents but keeps login/logout on management
surfaces:

1. Add an agent-callable MCP auth status tool.
2. Keep login and logout on CLI/HTTP/UDS operator-management surfaces.
3. Tool diagnostics may point agents or operators to the appropriate management
   command or endpoint, but the tool surface does not execute the OAuth browser
   flow directly.
4. Redaction rules remain strict: no token material, codes, PKCE verifiers, or
   secret callback state crosses operator or agent surfaces.

## Alternatives Considered

### Alternative 1: Full status/login/logout tool family

- **Description**: Expose MCP auth status, login, and logout as agent-callable
  tools.
- **Pros**: Maximizes symmetry with the rest of the tool registry.
- **Cons**: Pulls interactive browser and callback lifecycle into the normal
  agent tool path; diverges from the management separation used by the
  references; complicates approval and UX semantics.
- **Why rejected**: The final design needs diagnostics in the tool layer, not a
  browser-driven auth flow in the normal tool-call loop.

### Alternative 2: Keep all MCP auth operator-only

- **Description**: Do not expose MCP auth to agents at all.
- **Pros**: Simplest security boundary.
- **Cons**: Agents cannot distinguish auth failures from broader source-health
  failures in a structured way.
- **Why rejected**: The registry needs structured status visibility to stay
  agent-manageable.

## Consequences

### Positive

- Agents can reason about auth-related availability failures.
- Browser/OAuth lifecycle remains on explicit management paths.
- The design stays closer to the management split used by the references.

### Negative

- Agents cannot fully self-repair a missing OAuth login through tools alone.
- Docs and error contracts must clearly describe the handoff from status tool to
  management surface.

### Risks

- Users may expect full self-healing from the status tool. Mitigation: make
  deterministic error codes and follow-up repair hints part of the contract.

## Implementation Notes

- Reuse existing `internal/mcp/auth` redacted status models.
- Reuse the existing `internal/tools.MCPAuthStatus` adapter shape rather than
  introducing a parallel status model for the built-in tool.
- Ensure CLI, HTTP, and UDS expose the same status semantics referenced by the
  tool diagnostics.
- Treat auth refresh side effects carefully when the status path offers a
  refresh mode.

## References

- `.compozy/tasks/tools-registry/_techspec.md`
- `internal/cli/mcp_auth.go`
- `internal/mcp/auth/service.go`
- `.resources/claude-code/commands/mcp/mcp.tsx`
- `.resources/claude-code/services/mcp/auth.ts`
- `.resources/hermes/website/docs/reference/mcp-config-reference.md`
- `.resources/hermes/website/docs/reference/slash-commands.md`
- `.resources/openclaw/docs/cli/mcp.md`
