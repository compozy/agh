---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/udsapi/agent_channels_test.go
line: 365
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:17a20cddef49
review_hash: 17a20cddef49
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 009: Assert the resolved direct room ID, not just that a pointer exists.
## Review Comment

The new routing contract is surface-based, so this test should fail if reply resolution produces the wrong direct room or an empty string. Seed `source.DirectID` and check `*seen.DirectID` exactly; `seen.DirectID != nil` alone won't catch that regression.

## Triage

- Decision: `valid`
- Notes: `TestAgentChannelReplyResolvesSourceMessageMetadata` proves only that `seen.DirectID` is non-nil. Because the reply contract is surface-specific, the test should assert the exact direct-room identifier copied from source metadata so it fails if reply routing resolves the wrong room.
