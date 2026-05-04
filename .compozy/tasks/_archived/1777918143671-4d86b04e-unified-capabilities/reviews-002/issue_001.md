---
status: resolved
file: cmd/agh-codegen/main_test.go
line: 461
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:2d93b4c3dbb4
review_hash: 2d93b4c3dbb4
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 001: Wrap TestMarshalOpenAPI assertions in a t.Run("Should...") subtest.
## Review Comment

This is the only new test case in the file not following the required `Should...` subtest pattern.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases."

## Triage

- Decision: `valid`
- Root cause: `TestMarshalOpenAPI` is the one newly added top-level case in this file that bypasses the repo's required `t.Run("Should...")` structure.
- Fix plan: wrap the assertions in a `Should...` subtest and keep the existing checks unchanged.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
