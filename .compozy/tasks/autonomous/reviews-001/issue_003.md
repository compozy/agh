---
status: resolved
file: internal/api/core/agent_channels.go
line: 74
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:72fabaf89645
review_hash: 72fabaf89645
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 003: AgentChannelRecv: Double filtering of envelopes.
## Review Comment

Line 91-101 calls `agentChannelInbox` which internally calls `filterAgentChannelEnvelopes`, then line 102 calls `agentChannelMessagesFromEnvelopes` which also calls `filterAgentChannelEnvelopes` (line 671). This results in filtering the envelopes twice with the same channel.

---

## Triage

- Decision: `VALID`
- Notes: `AgentChannelRecv` calls `agentChannelInbox`, which already filters by channel, and then passes the result into `agentChannelMessagesFromEnvelopes`, which filters the same channel again. The second filter is the only one needed before response mapping. Fix by making `agentChannelInbox` only choose `Inbox` versus `WaitInbox` and return service envelopes unchanged.
