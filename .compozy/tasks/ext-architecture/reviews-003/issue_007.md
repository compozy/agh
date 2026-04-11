---
status: resolved
file: internal/api/spec/spec_test.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092884033,nitpick_hash:105b987d88ae
review_hash: 105b987d88ae
source_review_id: "4092884033"
source_review_submitted_at: "2026-04-10T23:25:44Z"
---

# Issue 007: Prefer a table-driven loop for these OpenAPI contract cases.
## Review Comment

These subtests all follow the same arrange/assert pattern, so adding more endpoint/schema coverage will keep duplicating setup and make omissions easier. A small table of `{name, path, method, check}` would keep this file much easier to extend.

As per coding guidelines, "Use table-driven tests with subtests (t.Run) as default in Go tests".

## Triage

- Decision: `valid`
- Notes:
  - The OpenAPI contract assertions follow the same arrange/assert shape repeatedly, so the file is a good fit for the repo’s table-driven test default.
  - Root cause: the test grew by accreting individual subtests instead of extending a shared table.
  - Fix plan: convert the repeated subtests into a compact table-driven loop that keeps one shared document setup and one `t.Run("Should...")` case per contract assertion.
  - Implemented: converted the repeated OpenAPI contract assertions into a single table-driven loop that preserves the existing `Should...` subtest names and shared document setup.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.
