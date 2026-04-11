---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 316
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:9995e9859a92
review_hash: 9995e9859a92
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 002: Deduplicate StartInstance and RestartInstance transition flow.
## Review Comment

Both methods run the same Starting → Ready sequence. A shared helper would reduce maintenance drift.

---

## Triage

- Decision: `Invalid`
- Notes:
  This is a local integration-test helper refactor, not a correctness issue. `StartInstance` and `RestartInstance` intentionally keep distinct error context, and changing only the UDS helper would not materially reduce maintenance drift because the same pattern exists in the HTTP integration suite outside this batch. Under the scoped review-fix workflow, this would be stylistic churn rather than a bug fix.
  Closed with no code change after inspection confirmed there is no behavioral defect to fix.
