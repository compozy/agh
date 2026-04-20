---
status: resolved
file: internal/core/prompt/review.go
line: 68
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4058705969,nitpick_hash:67f1363db896
review_hash: 67f1363db896
source_review_id: "4058705969"
source_review_submitted_at: "2026-04-04T17:43:33Z"
---

# Issue 003: Clarify that this restriction is about issue markdown files, not source edits.
## Review Comment

`Update only the issue files that belong to this batch` can read like code changes are off-limits, which conflicts with the later instruction to implement production fixes. Rewording this to “Do not edit issue files outside this batch” would remove that ambiguity.

## Triage

- Decision: `valid`
- Root cause: `internal/core/prompt/review.go` emits `Update only the issue files that belong to this batch.` inside the critical scope block, so the sentence can be read as forbidding source edits rather than limiting which review markdown files may be changed.
- Evidence: the same generated prompt later instructs the agent to `Implement complete production fixes`, and the `<batch_scope>` section separately lists `Code files in scope`, so the ambiguity comes from this one restriction line rather than the overall batch contract.
- Fix approach: reword the restriction to `Do not edit issue files outside this batch.` and add prompt coverage so the clarified wording remains stable.

## Resolution

- Updated `internal/core/prompt/review.go` to say `Do not edit issue files outside this batch.`, which makes the restriction explicitly about out-of-batch review markdown rather than source edits.
- Updated `internal/core/prompt/prompt_test.go` so the review prompt test now requires the clarified wording and rejects the previous ambiguous sentence.
- Verification:
  - `go test ./internal/core/prompt -run 'TestBuildCodeReviewPrompt(UsesInstalledSkillsAndAvoidsLegacyDependencies|RespectsManualCommitMode)$' -count=1`
  - `make verify` (`0 issues`; `DONE 2416 tests, 1 skipped in 41.112s`; build succeeded with `All verification checks passed`)
