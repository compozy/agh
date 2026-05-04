# TC-FUNC-032 — Cancellation propagates from registry through provider handle and hooks

- **Priority:** P1
- **Type:** Functional / lifecycle
- **Trace:** Task 04, Safety Invariant 11

## Test Steps

1. Invoke a slow provider; cancel context after 100ms.
   - **Expected:** Provider handle observes cancellation; returns deterministic cancellation error; `tool.post_error` fires; telemetry includes `decision = "canceled"`.
2. Hook running pre-call observes context cancellation.
   - **Expected:** Hook execution stops; result returned to caller is `tool_canceled`.
3. Hosted MCP proxy disconnect mid-call cancels the derived approval/dispatch context (interaction with TC-FUNC-049).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestDispatchCancellation`
