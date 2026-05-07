---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/udsapi/model_catalog_test.go
line: 64
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:5c9861865aef
review_hash: 5c9861865aef
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 009: Assert 404 response body, not only status code.
## Review Comment

Line 64-Line 67 validates only the code; add a body assertion to make the route contract explicit.

As per coding guidelines: "`**/*_test.go`: Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient."

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/udsapi/model_catalog_test.go` checks only the `404` status for `/api/openai/v1/models`.
  - AGH test rules require response-body evidence alongside status assertions, even for negative route contracts.
  - Fix plan: assert the returned body contains the router’s deterministic not-found payload.
  - Fixed in `internal/api/udsapi/model_catalog_test.go` and verified with focused package tests plus `make verify`.
