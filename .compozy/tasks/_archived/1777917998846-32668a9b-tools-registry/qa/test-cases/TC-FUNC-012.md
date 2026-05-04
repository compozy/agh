# TC-FUNC-012 — Operator vs session projection differs intentionally

- **Priority:** P0
- **Type:** Functional / projections
- **Trace:** Task 03, ADR-006, Safety Invariant 14

## Objective

Prove the operator projection includes unavailable, unauthorized, and conflicted tools with reason codes, while the session projection exposes only callable tools for the effective session.

## Test Steps

1. Configure 8 tools: 2 native callable, 1 native denied by deny-list, 1 conflicted, 1 extension-host with unhealthy extension, 1 MCP with `mcp_auth_required`, 1 with `read_only` source-untrusted, 1 hidden by ACP ceiling.
2. `GET /api/tools` operator projection.
   - **Expected:** All 8 listed with their reason codes.
3. `GET /api/sessions/{id}/tools` session projection.
   - **Expected:** Only the 2 callable native tools listed; reasons not exposed (only `tools` array).
4. CLI parity: `agh tool list -o json` mirrors `GET /api/tools`; `agh tool list --session-only` mirrors `GET /api/sessions/{id}/tools`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestProjectionDivergence`
