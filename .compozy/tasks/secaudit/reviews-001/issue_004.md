---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: internal/api/core/tools.go
line: 428
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215828964,nitpick_hash:5d82c73d823b
review_hash: 5d82c73d823b
source_review_id: "4215828964"
source_review_submitted_at: "2026-05-03T03:31:19Z"
---

# Issue 004: Centralize this approval-scope normalization helper.
## Review Comment

`approvalScopeField` now duplicates the same security rule already implemented in `internal/tools/approval_token.go`. Keeping both copies in sync is easy to miss later, and any drift would make mint-time and store-time validation disagree. Consider moving this helper into `internal/tools` and reusing it from both call sites.

## Triage

- Decision: `invalid`
- Reasoning: this is a maintainability refactor, not a correctness or security defect in the current batch. `internal/api/core/tools.go` and `internal/tools/approval_token.go` currently enforce the same scope-mismatch rule and there is no evidence of behavioral drift in the active code or tests.
- Scope note: accepting the suggestion would require touching additional non-scoped production files only to deduplicate code, which is outside this remediation batch’s intent.
