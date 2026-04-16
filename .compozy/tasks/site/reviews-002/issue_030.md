---
status: resolved
file: internal/config/mcp_resource_test.go
line: 27
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:9c1643846f52
review_hash: 9c1643846f52
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 030: Assert the specific validation failure, not only non-nil error
## Review Comment

This test currently passes on any decode/validate error, including unrelated regressions.

As per coding guidelines: `MUST have specific error assertions (ErrorContains, ErrorAs)`.

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed test file has moved; the current parser coverage is in `internal/config/mcpjson_test.go`.
  - Root cause: `TestParseMCPServersJSONRejectsInvalidEntries` only checks that parsing fails, so it does not pin the missing-command validation branch.
  - Intended fix: assert the specific field-path/message emitted by the invalid entry validation.
  - Result: the current MCP JSON parser test now asserts the missing-command validation context explicitly; verified with `go test ./internal/config` and `make verify`.
