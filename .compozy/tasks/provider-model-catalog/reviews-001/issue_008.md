---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/api/spec/spec_test.go
line: 108
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:79cad50e0528
review_hash: 79cad50e0528
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 008: Assert error response schemas for declared non-2xx statuses
## Review Comment

This case currently verifies that 403/503 responses exist, but not their JSON schema shape. Add schema assertions for those statuses (for example, required error envelope fields) so spec regressions are caught earlier.

As per coding guidelines "Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".

## Triage

- Decision: `valid`
- Notes:
  - The OpenAPI spec test currently confirms that `403` and `503` responses exist for `/api/openai/v1/models`, but it does not validate their JSON schema.
  - That leaves the declared error-envelope contract under-tested and allows schema regressions behind stable status codes.
  - Fix: assert the non-2xx response schemas expose the expected error-envelope fields.
