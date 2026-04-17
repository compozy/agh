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

# Issue 029: Wrap validation failures with operation context
## Review Comment

Line 28 and Line 52 currently return raw errors; wrapping here would preserve call-site intent and align with project error rules.

As per coding guidelines: `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

Also applies to: 51-53

## Triage

- Decision: `INVALID`
- Notes:
  - The reviewed file `internal/config/mcp_resource.go` is not present in this checkout.
  - In the live MCP parsing/validation code, the relevant returns already include source or field-path context, for example `config: decode MCP JSON ...` in `internal/config/mcpjson.go` and `providers.<name>.mcp_servers[...]` paths in `internal/config/provider.go`.
  - There is no remaining context-free validation return matching this review comment, so no code change is required.
  - Result: resolved as stale/non-applicable after current-tree inspection; no code change required.
