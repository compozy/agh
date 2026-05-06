---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/hooks/hooks_test.go
line: 1666
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232600854,nitpick_hash:e3359e1672b9
review_hash: e3359e1672b9
source_review_id: "4232600854"
source_review_submitted_at: "2026-05-06T01:29:19Z"
---

# Issue 008: Split the unmatched compaction path into its own subtest.
## Review Comment

This is a second behavior case inside `TestDispatchPermissionAndContextHooksApplyPatches`; moving it into `t.Run("Should leave compaction untouched when no hook matches", ...)` will keep failures isolated and match the repo’s Go test conventions.

As per coding guidelines, `**/*_test.go`: Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures.

## Triage

- Decision: `valid`
- Notes:
  - `TestDispatchPermissionAndContextHooksApplyPatches` currently verifies both the matched compaction patch path and the unmatched pass-through path in one flat assertion block.
  - A failure in the unmatched path is harder to localize because it is bundled with the earlier permission/context assertions.
  - Fix plan: keep the shared hook setup in the parent test and split the unmatched compaction behavior into its own named subtest, with the matched compaction assertions isolated as a separate behavior case too.

## Resolution

- Split the matched and unmatched compaction assertions into separate named subtests while keeping the shared hook setup in the parent test.
- Verified with fresh full `make verify` (passed).
