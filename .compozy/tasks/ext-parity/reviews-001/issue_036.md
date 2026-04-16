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

# Issue 036: Protect Args[0] assertion with a length check
## Review Comment

Line 90 can panic if canonicalization or validation starts rejecting/emptying args in future changes.

## Triage

- Decision: `VALID`
- Notes: `TestMCPServerResourceStoreRoundTripReturnsTypedRecords` still indexes `record.Spec.Args[0]` directly. If canonicalization ever drops or rejects args, the test will panic instead of explaining what changed. The fix is to assert the expected args length before checking the first element.
