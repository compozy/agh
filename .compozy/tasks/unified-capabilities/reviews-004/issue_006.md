---
status: resolved
file: internal/daemon/daemon_test.go
line: 4222
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4149203771,nitpick_hash:60c21cae1595
review_hash: 60c21cae1595
source_review_id: "4149203771"
source_review_submitted_at: "2026-04-21T16:10:13Z"
---

# Issue 006: Wrap this case in a t.Run("Should...") subtest.
## Review Comment

The scenario is useful, but the new coverage doesn't follow the test shape required for new Go test cases in this repo.

As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

## Triage

- Decision: `valid`
- Root cause: `TestFakeSessionManagerClearConversationTreatsMissingSessionAsFreshConversation` was added as a direct top-level body instead of the repository's required `t.Run("Should...")` test shape.
- Fix plan: wrap the existing assertions in a single named subtest and preserve the current behavior checks.
- Resolution: wrapped the fake-session-manager coverage in a single named `Should...` subtest without changing the behavior under test.
- Verification: `go test ./internal/daemon` and `make verify` passed after the change.
