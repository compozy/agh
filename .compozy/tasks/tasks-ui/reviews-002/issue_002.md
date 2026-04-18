---
status: resolved
file: internal/api/contract/tasks_test.go
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133463506,nitpick_hash:5003f47331f2
review_hash: 5003f47331f2
source_review_id: "4133463506"
source_review_submitted_at: "2026-04-18T03:54:22Z"
---

# Issue 002: Split these contract checks into named Should... subtests.
## Review Comment

This file packs several independent payload/request cases into a few large tests, which makes failures harder to localize and drifts from the repo’s test structure standard.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

Also applies to: 175-415, 417-438

## Triage

- Decision: `VALID`
- Reasoning: `internal/api/contract/tasks_test.go` currently packs multiple independent assertions into broad top-level tests, which makes failures harder to localize and does not follow the repo’s required `t.Run("Should ...")` subtest pattern.
- Root cause analysis: The new contract coverage was added as large monolithic tests rather than structured subtests.
- Intended fix: Split the broad cases into named `Should ...` subtests while preserving the existing payload coverage and assertions.
- Resolution: Split the contract coverage into focused `Should ...` subtests for read-model marshalling and `UpdateTaskRequest.HasChanges()`.
- Verification:
  - `go test ./internal/api/contract ./internal/api/core ./internal/daemon`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
