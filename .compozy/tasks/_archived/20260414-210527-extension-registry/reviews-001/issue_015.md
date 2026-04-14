---
status: resolved
file: internal/registry/version_test.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106850065,nitpick_hash:74441d728e9b
review_hash: 74441d728e9b
source_review_id: "4106850065"
source_review_submitted_at: "2026-04-14T14:43:27Z"
---

# Issue 015: Add t.Parallel() to subtests for independent execution.
## Review Comment

Per coding guidelines, independent subtests should call `t.Parallel()` to enable concurrent execution.

As per coding guidelines: "Use t.Parallel() for independent subtests in Go tests".

## Triage

- Decision: `valid`
- Notes: The subtests in `TestVersionIsNewer()` are independent and currently do not call `t.Parallel()`. I will add `t.Parallel()` to those subtests in `internal/registry/version_test.go` so the file matches the repository’s concurrency guidance for independent table cases.
- Resolution: Added `t.Parallel()` to the independent `TestVersionIsNewer()` subtests and captured the loop variable before running them in parallel.
- Verification: `go test ./internal/registry/...`; `make verify`
