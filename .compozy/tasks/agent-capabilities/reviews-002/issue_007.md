---
status: pending
file: internal/network/router_integration_test.go
line: 196
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135966430,nitpick_hash:36aff77d8442
review_hash: 36aff77d8442
source_review_id: "4135966430"
source_review_submitted_at: "2026-04-19T12:48:57Z"
---

# Issue 007: Add t.Parallel() for independent integration tests.
## Review Comment

The new integration tests are self-contained and can run concurrently. Per coding guidelines, independent tests should use `t.Parallel()`.

## Triage

- Decision: `UNREVIEWED`
- Notes:
