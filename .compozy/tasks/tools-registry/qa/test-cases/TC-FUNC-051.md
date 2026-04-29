# TC-FUNC-051 — UDS routes have parity with HTTP routes

- **Priority:** P1
- **Type:** Functional / UDS API
- **Trace:** Task 11

## Test Steps

1. For each HTTP route, the matching UDS route returns behaviorally equivalent payloads for the same persisted state.
2. Status codes and body shapes match.
3. UDS error payloads use the same `code` / `reason_codes` schema as HTTP.
4. Both transports use the same `internal/api/core` handlers (no duplicated logic).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/api/udsapi -run TestToolsUDSParity`
