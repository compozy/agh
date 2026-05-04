# TC-FUNC-014 — Daemon min/max bounds for hosted MCP TTL and approval timeout

- **Priority:** P2
- **Type:** Functional / config validation
- **Trace:** Task 02, TechSpec Validation

## Objective

Prove `[tools.hosted_mcp].bind_nonce_ttl_seconds` must be within `[5, 300]`, and `[tools.policy].approval_timeout_seconds` must be within `[10, 1800]`. Out-of-range values fail config validation.

## Test Steps

1. Set `bind_nonce_ttl_seconds = 4`.
   - **Expected:** Reject.
2. Set `bind_nonce_ttl_seconds = 301`.
   - **Expected:** Reject.
3. Set `approval_timeout_seconds = 9`.
   - **Expected:** Reject.
4. Set `approval_timeout_seconds = 1801`.
   - **Expected:** Reject.
5. Boundary values 5 / 300 / 10 / 1800 accepted.
6. Negative result byte limits and zero rejected.
7. `trusted_sources` entries that do not resolve to known extension/MCP source refs rejected.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/config -run TestToolsConfigBounds`
