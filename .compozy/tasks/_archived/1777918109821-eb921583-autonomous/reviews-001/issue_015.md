---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1567
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:f573f870b2b4
review_hash: f573f870b2b4
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 015: Re-use an explicit idempotency_key on the second approve call.
## Review Comment

Right now the repeated approve request has no body, so this only proves the endpoint is re-entrant after approval. It does not verify the new `TaskExecutionRequest` idempotency contract, and a handler that ignored idempotency keys entirely would still pass.

## Triage

- Decision: `VALID`
- Notes: The repeated approval integration request uses an empty body, so it proves re-entrancy but not the `TaskExecutionRequest.idempotency_key` contract. Fix by sending the same explicit `idempotency_key` on both approval calls and asserting the second call returns the same run.
