---
status: resolved
file: internal/api/core/tasks_surface_integration_test.go
line: 278
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:f5983c72be42
review_hash: f5983c72be42
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 012: Capture and assert the forwarded ExecutionRequest.
## Review Comment

These three stubs discard the new `taskpkg.ExecutionRequest`, so this test still passes if the handlers stop binding `idempotency_key` or `network_channel` and forward a zero-value request. Please record the request for at least one publish/start/approve call and assert the handler passed the expected payload through.

As per coding guidelines, `Focus on critical paths: workflow execution, state management, error handling` and `Ensure tests verify behavior outcomes, not just function calls`.

## Triage

- Decision: `VALID`
- Notes: The integration stubs for publish/start/approve discard `taskpkg.ExecutionRequest`, so the test only proves the handler calls the service and misses regressions in request binding for `idempotency_key`, `network_channel`, or metadata. Fix by sending request bodies for the mutation routes and recording/asserting the forwarded execution request for publish/start/approve.
