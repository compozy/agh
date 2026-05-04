# TC-FUNC-007 — Empty config loads safe defaults for `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]`

- **Priority:** P1
- **Type:** Functional / config
- **Trace:** Task 02, TechSpec Config Lifecycle

## Objective

Prove a minimal `config.toml` (or empty config) yields TechSpec-defined defaults: `enabled = true`, `hosted_mcp_enabled = true`, `default_max_result_bytes = 262144`, `external_default = "disabled"`, `approval_timeout_seconds = 120`, `bind_nonce_ttl_seconds = 30`, `trusted_sources = []`.

## Test Steps

1. Boot daemon with config containing only `[mcp_servers]` placeholder.
2. Inspect resolved tool config via debug API or test-only inspection.
   - **Expected:** Defaults match.
3. Confirm overlays preserve defaults when only overriding individual keys.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/config -run TestToolsConfigDefaults`
