## TC-UI-002: Web Settings MCP And Extension Redaction

**Priority:** P1 (High)
**Type:** UI
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that web settings surfaces show MCP auth status and extension environment diagnostics without exposing token or environment values.

### Traceability

- Tasks: task_05 MCP Auth and Skill Security; task_09 Environment, Extension, and Release Hardening.
- TechSpec: issues 27, 57, and 59.
- ADR: ADR-003 MCP OAuth auth subsystem.
- Surfaces: `web/src/routes/_app/settings`, `web/src/hooks/routes/use-settings-mcp-servers-page.ts`, `use-settings-hooks-extensions-page.ts`, `web/src/systems/settings`, generated OpenAPI types, settings fixtures/tests.

### Preconditions

- Settings fixtures include a remote MCP server with redacted `auth_status` and no token material.
- Extension fixture includes `requires_env` and `missing_env` arrays with variable names only.
- Sentinel token/env values exist in test data only to confirm they do not render.

### Test Steps

1. Run focused web settings MCP and hooks/extensions tests.
   - **Expected:** Tests pass and assert redacted auth status plus missing environment diagnostics.

2. Run settings API adapter tests.
   - **Expected:** Adapter preserves `auth_status`, `requires_env`, and `missing_env` fields without inventing secret values.

3. Typecheck generated settings contracts.
   - **Expected:** TypeScript types include the new fields and reject unmodeled token fields.

4. Render settings MCP and hooks/extensions pages with fixtures.
   - **Expected:** UI shows actionable status and missing env names only; no token, refresh token, authorization code, PKCE verifier, client secret, or env value appears in DOM text.

5. Check responsive layout if browser validation is part of task_11.
   - **Expected:** Missing env badges/messages fit at 375px, 768px, and 1280px.

### Evidence To Capture

- `qa/logs/TC-UI-002/web-settings-vitest.log`
- `qa/logs/TC-UI-002/settings-redaction-grep.log`
- `qa/screenshots/TC-UI-002/settings-mcp-desktop.png`
- `qa/screenshots/TC-UI-002/settings-extensions-mobile.png`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Remote auth unauthenticated | `auth_status.status=missing` | UI prompts auth-managed status, not token edit |
| Multiple missing env names | 3+ names | Names wrap without overflow |
| Token sentinel in fixture | `access_token=secret` accidentally present | Test or grep fails |
| Local stdio MCP server | No auth status | Existing local edit flow still works |

### Related Test Cases

- TC-SEC-001: Backend MCP auth redaction.
- TC-FUNC-003: Backend extension env diagnostics.
