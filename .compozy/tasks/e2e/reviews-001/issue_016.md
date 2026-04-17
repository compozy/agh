---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 566
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:d2698ed6aef6
review_hash: d2698ed6aef6
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 016: Consider using errors.New for sentinel errors.
## Review Comment

The custom `errorString` type works but adds complexity. Standard sentinel errors using `errors.New` work better with `errors.Is()`:

## Triage

- Decision: `VALID`
- Root cause: the custom `errorString` helper exists only to synthesize sentinel-like test errors, which is unnecessary now that the surrounding assertions should use `errors.Is`.
- Fix plan: replace the helper with standard `errors.New` sentinel vars and remove the custom error type.
- Resolution: replaced the custom `errorString` helper with standard `errors.New` sentinel values and removed the bespoke error type.
- Verification: `go test ./internal/daemon` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.
