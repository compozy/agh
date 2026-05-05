# ADR-003: Implement MCP OAuth as a First-Class Auth Subsystem

## Status

Accepted

## Date

2026-04-24

## Context

The current MCP server configuration models local subprocess servers with command, args, and environment. Selected Hermes issue 27 requires OAuth 2.1 and PKCE support for authenticated MCP integrations.

Manual token environment variables would be smaller, but they would not deliver a complete OAuth flow, token refresh, or safe operator ergonomics.

## Decision

Add a first-class MCP auth subsystem, with a package such as `internal/mcp/auth`, that supports OAuth 2.1 + PKCE, authorization callback handling, refresh token lifecycle, provider/server metadata, durable token storage, secret redaction, and CLI login/status/logout workflows.

The config model must distinguish local subprocess MCP servers from authenticated remote MCP servers. Auth state must be stored outside plain config overlays and must avoid leaking access tokens through logs, settings responses, or CLI output.

## Alternatives Considered

- Minimal token resolver using `token_env`, `auth_header`, or a refresh command. This is simpler but leaves OAuth and token lifecycle outside AGH.
- Model-only interfaces with no login flow. This reduces initial scope but leaves issue 27 partially unresolved.

## Consequences

- New token storage and redaction tests are required.
- MCP resolution must become auth-aware without exposing secrets through existing settings and resource surfaces.
- CLI commands need integration tests with a local OAuth test server.

## Implementation Notes

- Prefer small interfaces for token store, provider metadata discovery, and OAuth browser/callback coordination.
- Store refreshable credentials under AGH home with restrictive file permissions or in the global DB using redacted API DTOs.
- Keep local subprocess MCP server behavior unchanged except for shared config validation.

## References

- `.compozy/tasks/hermes/analysis/analysis_tools_security.md`
- Issue: 27
