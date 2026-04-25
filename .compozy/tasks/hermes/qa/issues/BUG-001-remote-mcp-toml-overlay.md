# BUG-001: Remote MCP TOML overlays reject documented fields

## Status

Fixed in Task 11.

## Severity

P0 security/config regression. Remote OAuth-enabled MCP servers documented under `[[mcp_servers]]` could not be loaded from `config.toml`, blocking the real MCP auth/redaction flow.

## Reproduction

1. Create an isolated `AGH_HOME` with:

```toml
[[mcp_servers]]
name = "linear"
transport = "sse"
url = "https://mcp.example/sse"

[mcp_servers.auth]
type = "oauth2_pkce"
authorization_url = "https://auth.example/authorize"
token_url = "https://auth.example/token"
client_id = "client-id"
client_secret_env = "HERMES_MCP_SECRET"
scopes = ["read"]
```

2. Run `agh config validate -o json`.

## Observed

The CLI failed with unknown key errors for `mcp_servers.transport`, `mcp_servers.url`, and every `mcp_servers.auth.*` field.

## Expected

The documented remote MCP config shape loads, validates, and exposes only redacted OAuth status and environment variable names through CLI/API/settings surfaces.

## Root Cause

`internal/config/merge.go` decoded TOML overlays through a narrower `mcpServerOverlay` type that only listed legacy stdio fields: `name`, `command`, `args`, and `env`. The canonical `MCPServer` model and validators already supported remote fields, but the overlay decoder rejected them before validation.

## Fix

Added `transport`, `url`, and structured `auth` overlay fields with merge support in `internal/config/merge.go`, plus regression coverage in `internal/config/config_test.go`.

## Verification Evidence

- Failing live flow: initial TC-SEC-001 command output during QA execution.
- Regression test: `.compozy/tasks/hermes/qa/logs/TC-SEC-001/regression-config-remote-mcp.log`
- Build after fix: `.compozy/tasks/hermes/qa/logs/TC-SEC-001/make-build-after-config-fix.log`
- Post-fix CLI validation: `.compozy/tasks/hermes/qa/logs/TC-SEC-001/config-validate-postfix.json`
- Redacted CLI/API evidence:
  - `.compozy/tasks/hermes/qa/logs/TC-SEC-001/config-show-redacted.json`
  - `.compozy/tasks/hermes/qa/logs/TC-SEC-001/mcp-auth-status.json`
  - `.compozy/tasks/hermes/qa/logs/TC-SEC-001/settings-mcp-servers.json`
  - `.compozy/tasks/hermes/qa/logs/TC-SEC-001/redaction-check.log`
