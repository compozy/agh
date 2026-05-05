---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 1386
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:6b4b466cb597
review_hash: 6b4b466cb597
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 022: Exercise re-approval with an explicit idempotency key.
## Review Comment

Both approve calls send an empty body, so this never verifies the new `TaskExecutionRequest.idempotency_key` path. The same run ID could still be returned by an "already approved" fast path, which leaves the actual execution dedupe behavior untested.

As per coding guidelines, `Focus on critical paths: workflow execution, state management, error handling` and `Ensure tests verify behavior outcomes, not just function calls`.

## Triage

- Decision: `VALID`
- Notes: The integration test approves the same task twice with nil bodies. That only covers the already-approved path, not explicit `TaskExecutionRequest.idempotency_key` decoding and run dedupe through the UDS handler.
- Fix: Send a JSON body with a stable idempotency key for both approve requests and keep asserting the second response returns the first run ID.
