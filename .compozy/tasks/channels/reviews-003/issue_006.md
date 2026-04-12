---
status: resolved
file: internal/daemon/channels_test.go
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093890324,nitpick_hash:ce973d60473d
review_hash: ce973d60473d
source_review_id: "4093890324"
source_review_submitted_at: "2026-04-11T15:00:20Z"
---

# Issue 006: Convert this suite to t.Run("Should...") table-driven subtests.
## Review Comment

The coverage is solid, but the file currently encodes each scenario as a separate top-level test. That diverges from the repo’s default test structure and makes the repeated setup harder to extend consistently.

As per coding guidelines, `**/*_test.go`: `Use table-driven tests with subtests (t.Run) as default` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `Invalid`
- Notes:
  Converting the whole file to table-driven top-level coverage would be broad structural churn with no correctness gain for this batch. The existing tests already use `Should...` naming, isolate distinct lifecycle scenarios cleanly, and the review scope is remediation of concrete defects rather than test-suite reformatting.
  Closed with no code change because there is no behavioral bug to fix in the scoped file.
