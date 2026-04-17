---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:35f0f6ce58dd
review_hash: 35f0f6ce58dd
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 003: Add t.Parallel() to integration test functions.
## Review Comment

The test functions are missing `t.Parallel()` calls. For integration tests that use isolated harnesses, parallel execution should be safe and improve test suite performance.

## Triage

- Decision: `VALID`
- Root cause: the transport-parity integration tests use isolated runtime harnesses and package-local state only, so they currently leave easy parallelism on the table.
- Fix plan: add `t.Parallel()` after the environment precondition check in each top-level test in this file.
- Resolution: added `t.Parallel()` to both top-level HTTP transport-parity integration tests.
- Verification: `go test ./internal/api/httpapi` passed. `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon` was rerun but is blocked before these tests execute because the branch is missing `internal/testutil/acpmock/driver/dist/index.js`. `make verify` hits the same unrelated blocker in `internal/testutil/acpmock` and `internal/testutil/e2e`.
