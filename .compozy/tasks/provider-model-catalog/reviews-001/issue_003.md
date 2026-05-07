---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/acp/types_test.go
line: 47
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:771287c95f18
review_hash: 771287c95f18
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 003: TestAgentProcessCapsSnapshotClonesConfigOptions should use t.Run("Should ...") subtests.
## Review Comment

The test covers two distinct behaviors — snapshot isolation from mutation, and `setConfigOptions` update correctness — but they are merged into a flat function body. Per the coding guidelines, each scenario should be a named subtest with its own `t.Parallel()`.

As per coding guidelines: `Use t.Run("Should ...") subtests with t.Parallel as default`.

## Triage

- Decision: `valid`
- Notes:
  - `TestAgentProcessCapsSnapshotClonesConfigOptions` currently merges two behaviors into one flat body.
  - The repo test conventions require named `t.Run("Should ...")` subtests with parallelization where allowed.
  - Fix: split snapshot-isolation and `setConfigOptions` update coverage into separate subtests.
