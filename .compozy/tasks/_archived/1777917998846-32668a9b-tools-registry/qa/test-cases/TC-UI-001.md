# TC-UI-001 — Tools diagnostics list renders canonical IDs and reason codes

- **Priority:** P1
- **Type:** UI / web operator surface
- **Trace:** Task 13, ADR-006, ADR-007

## Objective

Prove `web/src/systems/tools/**` renders operator-visible tools with canonical `tool_id`, `backend.kind`, `source.kind`, `availability` reason codes, and risk indicators sourced from `web/src/generated/agh-openapi.d.ts`.

## Preconditions

- Daemon backed by MSW fixtures or live UDS for native, extension-host, MCP, denied, conflicted, unavailable, and auth-required tool states.

## Test Steps

1. Visit operator tools page at viewports 375, 768, 1280.
   - **Expected:** Each tool entry shows canonical `tool_id`, `backend.kind`, `source.kind`, and current state.
2. Filter by `backend.kind = mcp` → only MCP-backed tools shown.
3. Click a conflicted tool → detail view shows `conflicted_id` and full provenance for both providers.
4. Auth-required MCP tool → tooltip/badge shows redacted `auth_status`; never the access token.
5. No invented controls (no inline OAuth login button, no inline approval prompt).

## Edge Cases

- Long canonical IDs do not break layout (overflow with ellipsis or wrap rules from `DESIGN.md`).
- Empty registry state shows informative empty state component.

## Automation

- **Target:** E2E
- **Status:** Existing for unit/component; Missing for cross-viewport visual coverage
- **Command/Spec:** `make bun-test web/src/systems/tools`; `make test-e2e-web`; for Task 16 highest-risk flow drive via `browser-use:browser`.
