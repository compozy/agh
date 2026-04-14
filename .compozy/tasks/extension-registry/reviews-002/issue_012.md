---
status: pending
file: internal/registry/multi_test.go
line: 431
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107316563,nitpick_hash:bc3b33304e40
review_hash: bc3b33304e40
source_review_id: "4107316563"
source_review_submitted_at: "2026-04-14T15:47:27Z"
---

# Issue 012: Consider using "Should..." naming pattern in subtests.
## Review Comment

Per coding guidelines, subtests should use `t.Run("Should...")` pattern. The current naming ("newer version available", "equal version") is descriptive but doesn't follow the prescribed pattern.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

---

## Triage

- Decision: `UNREVIEWED`
- Notes:
