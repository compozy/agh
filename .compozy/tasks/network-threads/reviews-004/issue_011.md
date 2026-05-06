---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/bridges/types.go
line: 621
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:9d20b934cce9
review_hash: 9d20b934cce9
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 011: Validate work_id with the same canonical rules as the other AGH conversation IDs.
## Review Comment

`thread_id` and `direct_id` are format-checked here, but `work_id` only gets generic length/control-character validation. That means malformed values like `"foo"` pass `InboundMessageEnvelope.Validate()` and only fail later in downstream network code.

Also applies to: 999-1017

## Triage

- Decision: `invalid`
- Notes: The current bridge validation path already routes `conversation.work_id` through the same canonical work-id rules used elsewhere in the network layer. `NetworkConversationRef.Validate` calls `validateBridgeNetworkConversationID(normalized.WorkID, "work_id")`, and the downstream canonical validator `network.ValidateWorkID` uses the same generic `ValidateConversationID(..., "work_id")` contract. There is no missing validation bug to fix here.
