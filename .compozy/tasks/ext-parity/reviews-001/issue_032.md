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

# Issue 032: Guard slice access before asserting MCP server fields
## Review Comment

Line 115 assumes at least one MCP server and can panic if normalization/filtering behavior changes, which makes failures less diagnosable.

## Triage

- Decision: `VALID`
- Notes: `TestAgentResourceCodecCanonicalizesTypedRecordSpec` still indexes `got.MCPServers[0]` without first proving that the slice is non-empty. If normalization or validation behavior changes, the test would panic instead of failing with a useful message. The fix is to assert the expected slice length before checking individual fields.
