---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/situation/service_test.go
line: 417
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233729248,nitpick_hash:1659f1d03dff
review_hash: 1659f1d03dff
source_review_id: "4233729248"
source_review_submitted_at: "2026-05-06T06:27:47Z"
---

# Issue 008: Wrap these new scenarios in t.Run("Should ...") blocks.
## Review Comment

These added cases are the only new tests in this chunk that skip the repo’s required subtest structure, which makes later expansion and table conversion harder.

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures".

Also applies to: 452-539

## Triage

- Decision: `valid`
- Root cause analysis: the new top-level tests added in this area encode single scenarios directly in the test body instead of wrapping them in `t.Run("Should ...")` blocks.
- Why this is valid: AGH test conventions require subtest structure even for one-case additions so later expansion stays consistent and reviewable.
- Fix approach: wrap the affected `internal/situation/service_test.go` scenarios in `t.Run("Should ...")` blocks while keeping `t.Parallel()` on the independent case bodies.

## Resolution

- Wrapped the affected `internal/situation/service_test.go` scenarios in `t.Run("Should ...")` blocks and kept the independent bodies parallel.

## Verification

- Focused regression: `go test ./internal/situation -run 'TestTaskStoreStubListRunReviewsSortsBeforeApplyingLimit|TestTaskRunPromptOverlayByIDRejectsMismatchedRunTaskPair|TestContextForSessionIncludesReviewerTaskBundleWithoutActiveLease' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
