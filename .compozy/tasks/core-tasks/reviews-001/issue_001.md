---
status: resolved
file: internal/api/contract/contract_test.go
line: 452
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:1e1d2da2ea92
review_hash: 1e1d2da2ea92
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 001: Use t.Run("Should...") consistently for the new task contract tests.
## Review Comment

This block introduces bare task JSON-shape tests and short case names like `"empty"` / `"title"`, which drifts from the repo’s required subtest convention.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the new task contract coverage was added as flat tests, and the `UpdateTaskRequest.HasChanges` table still uses terse case labels like `"empty"` and `"title"` instead of the required `Should...` convention.
- Fix approach: wrap the new task contract assertions in `t.Run("Should...")` subtests and rename the task change-detection cases to descriptive `Should...` names.

## Resolution

- Wrapped the new task contract JSON-shape assertions in `Should...` subtests and renamed the `UpdateTaskRequest.HasChanges` cases to descriptive `Should...` labels.
- Verified in the final `make verify` run.
