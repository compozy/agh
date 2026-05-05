---
status: resolved
file: internal/codegen/openapits/generate_test.go
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:8a0e8267f8c4
review_hash: 8a0e8267f8c4
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 008: Use t.Run("Should...") for TestGenerate to match default test structure.
## Review Comment

`TestGenerate` is a single-case top-level test; wrapping it in a `Should...` subtest keeps this file aligned with the repository’s required test style.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests."

## Triage

- Decision: `valid`
- Root cause: `TestGenerate` is the only top-level case in the file that skips the required `Should...` subtest wrapper.
- Fix plan: wrap the existing assertions in a `Should...` subtest and keep the current behavioral coverage intact.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
