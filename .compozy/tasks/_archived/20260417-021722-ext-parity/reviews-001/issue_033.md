---
status: resolved
file: internal/config/mcp_resource.go
line: 27
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:508f86abb5e4
review_hash: 508f86abb5e4
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 033: Wrap validation failures with operation context
## Review Comment

Line 28 and Line 52 currently return raw errors; wrapping here would preserve call-site intent and align with project error rules.

As per coding guidelines: `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

Also applies to: 51-53

## Triage

- Decision: `VALID`
- Notes: `validateMCPServerSpec` still returns raw scope and spec validation errors, so callers lose operation context when decode-time normalization fails. Wrapping both the scope validation and `normalized.Validate("mcp_server")` failures with MCP-server-specific context aligns the file with the project error-wrapping rules, and the tests should assert that context.
