# TC-FUNC-034 — Deterministic error mapping across HTTP/UDS/CLI/hosted MCP

- **Priority:** P1
- **Type:** Functional / error mapping
- **Trace:** Task 04, Task 11, TechSpec API Endpoints

## Objective

Prove registry errors map deterministically to public surfaces:

| Error | HTTP | CLI exit | Hosted MCP |
|-------|------|----------|------------|
| `ErrToolNotFound` | 404 | non-zero | tools/call → tool not found error |
| `ErrToolConflict` | 409 | non-zero | tool not registered |
| `ErrToolUnavailable` | 422 | non-zero | tools/call returns availability reason |
| `ErrToolDenied` | 403 | non-zero | denial reason |
| `ErrToolApprovalRequired` | 202 (CLI/HTTP) or 403 if no approval channel | non-zero with approval reason | approval bridge wait |
| `ErrToolInvalidInput` | 400 | non-zero | schema_invalid |
| `ErrToolResultTooLarge` | result with `truncated=true` | success with truncation note | tools/call returns truncated content |
| `ErrToolBackendFailed` | 502 | non-zero | backend error code |

## Test Steps

For each error: trigger via fixture, observe HTTP status code, body shape (`code`, `message`, `tool_id`, `reason_codes`, redacted details), CLI exit code + JSON body, hosted MCP `tools/call` response.

## Automation

- **Target:** Integration
- **Status:** Existing partial
- **Command/Spec:** `go test ./internal/api/core -run TestToolErrorMapping`
