---
status: resolved
file: internal/codegen/openapits/generate_test.go
line: 47
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:fe906e96012a
review_hash: fe906e96012a
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 009: Rename subtests to the required Should... convention.
## Review Comment

Subtests in this range (for example, `accepts matching generated output`, `rejects stale generated output`) do not follow the required `t.Run("Should...")` pattern.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases."

## Triage

- Decision: `valid`
- Root cause: the new `TestCheck` subtests use descriptive names, but they do not follow the workspace-mandated `Should...` convention.
- Fix plan: rename the affected subtests to `Should...` names while preserving the current assertions and parallelism.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
