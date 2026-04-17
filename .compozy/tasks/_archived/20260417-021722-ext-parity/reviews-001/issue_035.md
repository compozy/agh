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

# Issue 035: Assert the specific validation failure, not only non-nil error
## Review Comment

This test currently passes on any decode/validate error, including unrelated regressions.

As per coding guidelines: `MUST have specific error assertions (ErrorContains, ErrorAs)`.

## Triage

- Decision: `VALID`
- Notes: `TestMCPServerResourceCodecRejectsInvalidSpec` still treats any non-nil error as success, which would allow unrelated decode or scope regressions to satisfy the test. The fix is to assert the specific missing-command failure so the test proves the intended validation branch.
