---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 75
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:e1c26b3f5b69
review_hash: e1c26b3f5b69
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 012: Monolithic test function covers too many unrelated concerns.
## Review Comment

This 150-line test function verifies at least 10 distinct behaviors (nil projectors, nil catalogs, nil plans, codec comparison helpers, nil publisher, syncer creation) without subtests. Per coding guidelines, tests should use `t.Run("Should...")` pattern for organization.

Splitting into focused subtests improves:
- Failure diagnostics (know exactly which scenario failed)
- Parallel execution within the test
- Readability and maintenance

---

## Triage

- Decision: `VALID`
- Root cause: `TestToolMCPComparisonAndNilHelpers` currently mixes nil-receiver behavior, codec comparison helpers, publisher behavior, and constructor behavior in one long flow, which weakens failure diagnostics.
- Fix plan: split the unrelated concerns into focused subtests and keep them parallel where independence is clear.
- Resolution: split the monolithic test into focused parallel subtests for nil helpers, codec comparison, publisher behavior, and nil-logger syncer construction.
- Verification: `go test ./internal/daemon` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.
