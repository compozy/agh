---
status: resolved
file: internal/api/core/tasks_surface_test.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:53a11c725d04
review_hash: 53a11c725d04
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 009: Reshape this file around Should... subtests.
## Review Comment

The coverage is good, but the new file still relies on large top-level tests and subtests that don’t use the repo’s `Should...` pattern. Splitting the route/payload/error-path assertions into table-style subtests would make failures much easier to isolate.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `invalid`
- Notes: This is a formatting/style request, not a concrete behavior defect. The existing payload-builder tests already exercise distinct task conversion surfaces and failure output is still local enough to debug. Reformatting the whole file into `Should...` subtests is outside the value of this batch.
