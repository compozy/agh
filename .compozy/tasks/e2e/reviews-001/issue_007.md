---
status: resolved
file: internal/daemon/daemon_mock_agents_integration_test.go
line: 21
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:17d82b2675e8
review_hash: 17d82b2675e8
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 007: Add t.Parallel() to integration test functions.
## Review Comment

The daemon E2E test functions are missing `t.Parallel()` calls. Each test uses an isolated runtime harness, so parallel execution should be safe.

## Triage

- Decision: `VALID`
- Root cause: these daemon E2E tests each spin up their own isolated harness and do not share mutable state, so they can run in parallel safely.
- Fix plan: add `t.Parallel()` to each top-level test after the Node precondition guard.
- Resolution: added `t.Parallel()` to the top-level fixture-backed daemon mock-agent integration tests.
- Verification: `go test ./internal/daemon` passed. `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon` was rerun but is blocked before these tests execute because the branch is missing `internal/testutil/acpmock/driver/dist/index.js`. `make verify` hits the same unrelated blocker in `internal/testutil/acpmock` and `internal/testutil/e2e`.
