---
status: resolved
file: internal/store/session_liveness_test.go
line: 10
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:bb3e0b3ca7aa
review_hash: bb3e0b3ca7aa
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 017: Refactor test cases to t.Run("Should...") style consistently.
## Review Comment

Coverage is solid, but structure should be normalized to scenario-based `t.Run("Should...")` cases (including `TestCloneSessionLivenessMeta` and `TestHookRunQueryValidate`) for consistency and diagnostics.

As per coding guidelines, "MUST use t.Run(\"Should...\") pattern for ALL test cases".

Also applies to: 72-116, 118-145

## Triage

- Decision: `valid`
- Root cause: the touched session-liveness tests already use subtests in places, but their case names are not normalized to the required `Should ...` scenario style and two new functions still use straight-line bodies.
- Fix plan: rename/refactor the affected cases into consistent `t.Run("Should ...")` subtests while preserving the same validation coverage.
