---
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 1053
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4149203771,nitpick_hash:c0ef08834446
review_hash: c0ef08834446
source_review_id: "4149203771"
source_review_submitted_at: "2026-04-21T16:10:13Z"
---

# Issue 002: Use t.Run("Should...") for the new UDS handler cases.
## Review Comment

Both added tests are standalone bodies, but new Go test coverage in this repo is expected to use named `Should...` subtests.

As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

## Triage

- Decision: `valid`
- Root cause: the two newly added prompt handler tests are standalone top-level test bodies instead of repository-standard `t.Run("Should...")` subtests.
- Fix plan: wrap each scenario in a named `Should...` subtest without changing the assertions or coverage.
- Resolution: wrapped both new UDS prompt handler scenarios in named `Should...` subtests while preserving the existing request/response assertions.
- Verification: `go test ./internal/api/udsapi` and `make verify` passed after the change.
