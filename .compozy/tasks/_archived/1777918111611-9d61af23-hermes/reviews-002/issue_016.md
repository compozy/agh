---
status: resolved
file: internal/observe/observer_test.go
line: 93
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4175824057,nitpick_hash:46c9524c5b4f
review_hash: 46c9524c5b4f
source_review_id: "4175824057"
source_review_submitted_at: "2026-04-25T16:07:33Z"
---

# Issue 016: Prefer one table-driven retention test here.
## Review Comment

These two cases differ only by config and expected outcomes, so a single subtest table would remove duplication and make future retention branches easier to extend.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

## Triage

- Decision: `valid`
- Root cause: enabled and disabled retention sweep tests duplicate setup instead of using a table-driven structure for the behavior variants.
- Fix approach: refactor the retention sweep coverage into table-driven `Should...` subtests while preserving the enabled cutoff and disabled keep-history assertions.
