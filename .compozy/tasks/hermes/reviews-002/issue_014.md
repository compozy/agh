---
status: resolved
file: internal/memory/store_test.go
line: 1130
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4175824057,nitpick_hash:7a0c4542a83c
review_hash: 7a0c4542a83c
source_review_id: "4175824057"
source_review_submitted_at: "2026-04-25T16:07:33Z"
---

# Issue 014: Add a cross-workspace history subtest here.
## Review Comment

This scenario only ever exercises one workspace in the shared catalog, so it won't fail if `History` loses the bound workspace default or if `scope=""` events bleed across workspaces. A second `t.Run("Should isolate history by workspace")` case would cover the real multi-workspace path these APIs are built around.

As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases` and `Focus on critical paths: workflow execution, state management, error handling`.

## Triage

- Decision: `valid`
- Root cause: the existing operation-history test only exercises one workspace, so it would not catch empty-scope event leakage or loss of the workspace-bound default in `History`.
- Fix approach: add a `Should...` cross-workspace subtest that writes/searches through two stores sharing one catalog and asserts each workspace sees only its own workspace-bound search history.
