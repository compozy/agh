---
status: resolved
file: internal/bundles/service_test.go
line: 261
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:13731e089596
review_hash: 13731e089596
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 008: Consider converting these lifecycle scenarios to table-driven t.Run("Should...") subtests.

## Review Comment

The cases are independent and already follow the same arrange/act/assert shape. A table-driven harness would cut the duplicated fixture setup and make it easier to add new bundle/profile/scope permutations without growing the file linearly.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default, and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `invalid`
- Reasoning: this comment is a structural refactor suggestion, not a correctness defect. The existing lifecycle tests already isolate distinct behaviors with focused fixtures and assertions, and converting them into a shared table harness would widen the change surface without improving runtime behavior or regression detection for this batch.
- Resolution: no code change required for this review item.
