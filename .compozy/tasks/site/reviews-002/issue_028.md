---
status: resolved
file: internal/config/agent_resource_test.go
line: 115
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:8438a0e64aad
review_hash: 8438a0e64aad
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 028: Guard slice access before asserting MCP server fields
## Review Comment

Line 115 assumes at least one MCP server and can panic if normalization/filtering behavior changes, which makes failures less diagnosable.

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed test file has moved; the current MCP merge assertions live in `internal/config/provider_test.go`.
  - Root cause: the tests read `Args[0]` directly without first asserting the slice length, which can turn a regression into a panic and hide the real failure.
  - Intended fix: assert slice length before value in the current provider merge tests.
  - Result: the current provider merge tests now assert slice length before reading `Args[0]`; verified with `go test ./internal/config` and `make verify`.
