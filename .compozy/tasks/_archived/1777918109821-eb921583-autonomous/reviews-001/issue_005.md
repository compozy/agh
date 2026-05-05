---
status: resolved
file: internal/api/core/agent_channels.go
line: 666
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:d238dd557aa5
review_hash: d238dd557aa5
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 005: agentChannelMessagesFromEnvelopes: Similar unreachable nil check.
## Review Comment

Line 706 checks `if messages == nil` but messages is initialized with `make(...)` on line 672. Same issue as in `mergeCoordinationChannels`.

## Triage

- Decision: `VALID`
- Notes: `agentChannelMessagesFromEnvelopes` initializes `messages` with `make`, so the final nil guard is unreachable. The function already preserves a non-nil empty response slice. Fix by removing the dead nil check.
