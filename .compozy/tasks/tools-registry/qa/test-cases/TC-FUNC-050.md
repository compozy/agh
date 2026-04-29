# TC-FUNC-050 — HTTP routes for `/api/tools[...]` return canonical contracts

- **Priority:** P1
- **Type:** Functional / HTTP API
- **Trace:** Task 11

## Test Steps

1. `GET /api/tools` returns 200 with operator projection (canonical `tool_id`, `backend`, `source`, `availability`, etc.).
2. `POST /api/tools/search` accepts `{query, scope}` body and returns matching tools.
3. `GET /api/tools/{id}` returns 200 for known id, 404 for unknown, 400 for invalid id grammar, 409 for conflicted id.
4. `POST /api/tools/{id}/approvals` issues approval token (TC-SEC-011 covers lifecycle).
5. `POST /api/tools/{id}/invoke` returns invoke envelope.
6. `GET /api/sessions/{id}/tools` returns session-callable subset.
7. `POST /api/sessions/{id}/tools/search` searches only within callable subset.
8. `GET /api/toolsets` and `GET /api/toolsets/{id}` return toolset expansions.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/api/httpapi -run TestToolsRoutes`
