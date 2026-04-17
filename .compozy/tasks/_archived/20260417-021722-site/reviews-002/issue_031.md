---
status: resolved
file: internal/config/mcp_resource_test.go
line: 90
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:dd7e4cc05a70
review_hash: dd7e4cc05a70
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 031: Protect Args[0] assertion with a length check
## Review Comment

Line 90 can panic if canonicalization or validation starts rejecting/emptying args in future changes.

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed test file has moved; the current trim-collision MCP merge test is in `internal/config/provider_test.go`.
  - Root cause: the test asserts `Args[0]` without proving the slice length first, which can panic and hide a regression.
  - Intended fix: split the length and value assertions so failures stay diagnostic.
  - Result: the current trim-collision provider test now asserts args length before value; verified with `go test ./internal/config` and `make verify`.
