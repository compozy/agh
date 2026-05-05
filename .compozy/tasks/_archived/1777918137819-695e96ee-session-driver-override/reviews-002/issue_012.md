---
status: resolved
file: internal/store/session_liveness_test.go
line: 22
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:b5271614a9df
review_hash: b5271614a9df
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 012: Consider table-driven subtests to reduce duplication.
## Review Comment

These repeated validation cases are good candidates for table-driven tests; this will keep additions simpler and assertions more uniform.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

Also applies to: 125-160

## Triage

- Decision: `UNREVIEWED`
- Decision: `invalid`
- Notes: This is a readability refactor rather than a correctness or coverage defect. The current tests already use explicit `Should...` subtests, and converting the small number of cases to table-driven form would not change behavior or fix a review regression by itself.
