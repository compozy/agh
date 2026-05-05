---
status: resolved
file: internal/api/httpapi/session-clear_test.go
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:76819fb97898
review_hash: 76819fb97898
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 005: Use the repo’s standard t.Run("Should...") subtest pattern here.
## Review Comment

These are new handler cases, and they look independent enough to be grouped as subtests and run in parallel. That would align this file with the rest of the Go test conventions the repo asks for.

As per coding guidelines, `Use table-driven tests with subtests (\`t.Run\`) as default pattern for Go tests`, `Add t.Parallel() for independent subtests in Go`, and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `valid`
- Root cause: the new session-clear handler tests are independent cases but were added as separate top-level tests instead of the repo's default subtest/table pattern.
- Fix plan: group the cases under a parent test with `Should...` subtests and mark the independent subtests parallel.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
