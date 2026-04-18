---
status: resolved
file: internal/api/contract/tasks_test.go
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:5bc0ca046365
review_hash: 5bc0ca046365
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 002: Break this suite into t.Run("Should...") cases.
## Review Comment

These tests cover good scenarios, but they pack multiple independent assertions into a few top-level functions. Converting them to table-driven `Should...` subtests would make failures easier to localize and align the new coverage with the repo’s test shape.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `invalid`
- Notes: This is a test-structure preference, not a correctness or coverage defect. The current contract serialization tests are deterministic and already isolate the task contract surfaces being exercised. Rewriting them into `Should...` subtests would be a broad style refactor without changing behavior in this remediation batch.
