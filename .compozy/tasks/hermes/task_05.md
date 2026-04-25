---
status: completed
title: MCP Auth and Skill Security
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 05: MCP Auth and Skill Security

## Overview

Build the MCP remote authentication subsystem and close selected skill filesystem escape risks. This task adds OAuth 2.1 with PKCE, durable token storage, refresh and redaction behavior, CLI auth commands, remote server metadata handling, and symlink escape hardening for skill and managed extension paths.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-003, and task_01 outputs before changing MCP configuration or token storage
- NEVER log access tokens, refresh tokens, client secrets, authorization codes, or generated PKCE verifiers
- DO NOT implement OAuth as static token config; this task requires full auth lifecycle support
- DO NOT follow symlinks outside approved roots when loading skills or managed extension content
- Token persistence must have explicit permissions and tests for redaction boundaries
</critical>

<requirements>
- MUST add MCP remote auth configuration with safe redaction in config, API, CLI, and logs
- MUST implement OAuth 2.1 authorization code with PKCE for supported remote MCP servers
- MUST persist, refresh, inspect, and revoke or delete MCP auth tokens through a durable subsystem
- MUST add `agh mcp auth` commands for login, status, logout, and failure diagnostics
- MUST harden skill and managed extension file traversal against symlink escape
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 5.1 Add typed MCP auth config, metadata discovery, and redacted contract output
- [x] 5.2 Implement OAuth 2.1 + PKCE login, token exchange, refresh, status, and logout flows
- [x] 5.3 Add durable token storage with permission checks, redaction tests, and migration wiring
- [x] 5.4 Implement `agh mcp auth` CLI commands and operator-facing diagnostics
- [x] 5.5 Harden skill and managed extension path loading against symlink escape
- [x] 5.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Treat MCP OAuth as a real subsystem, not as config decoration. Keep token material behind narrow APIs, return redacted views through contracts, and make auth status useful for both CLI and future web settings surfaces. Skill security fixes should use canonical path checks and tests with temporary symlinks.

### Relevant Files
- `internal/config/provider.go` - provider and remote auth configuration
- `internal/config/mcp_resource.go` - MCP resource definitions and validation
- `internal/config/mcpjson.go` - MCP JSON import/export behavior
- `internal/api/contract/settings.go` - redacted settings and auth status payloads
- `internal/api/core/settings.go` - settings handlers and conversions
- `internal/cli/` - `agh mcp auth` command surface
- `internal/mcp/auth/` - new auth subsystem package destination
- `internal/skills/loader.go` - skill path loading hardening
- `internal/skills/resource.go` - skill resource traversal checks
- `internal/extension/install_managed.go` - managed extension path hardening

### Dependent Files
- `internal/config/*_test.go` - config validation and redaction tests
- `internal/mcp/auth/*_test.go` - OAuth, token refresh, persistence, and failure tests
- `internal/cli/*mcp*_test.go` - CLI command tests for login/status/logout behavior
- `internal/skills/*_test.go` - symlink escape rejection tests
- `web/src/routes/_app/settings/` - MCP settings or typed auth status updates if surfaced
- `packages/site/` - MCP remote auth and skill security documentation
- `.compozy/tasks/hermes/task_08.md` - setup lifecycle depends on redaction and auth config behavior

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - identifies MCP auth and skill security as selected hardening work
- [ADR-003: MCP OAuth Auth Subsystem](adrs/adr-003-mcp-oauth-auth-subsystem.md) - defines the OAuth 2.1 + PKCE and token storage decision

## Deliverables
- MCP OAuth 2.1 + PKCE auth subsystem with durable token storage
- `agh mcp auth login/status/logout` command set
- Redacted config/API/CLI/log output for MCP credentials and tokens
- Symlink escape hardening for skill and managed extension file loading
- Tests for auth lifecycle, refresh failure, redaction, and path escape rejection
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] OAuth state, PKCE verifier, and token exchange validation reject malformed or mismatched responses
  - [x] Token refresh updates durable storage without logging sensitive values
  - [x] Redacted config/API/CLI views never expose secret material
  - [x] Skill and managed extension loaders reject symlink escapes outside allowed roots
- Integration tests:
  - [x] Local mock OAuth server exercises login, refresh, status, and logout paths
  - [x] MCP remote auth status survives daemon restart through durable token storage
  - [x] Settings API and CLI report consistent redacted auth state
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- MCP remote servers can authenticate through OAuth 2.1 + PKCE
- Token state is durable, refreshable, and redacted across all surfaces
- Skill and managed extension loaders cannot escape approved roots through symlinks
- Affected backend, CLI, web, and docs tests pass
