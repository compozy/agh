---
status: resolved
file: internal/api/core/agent_channels.go
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6ta,comment:PRRC_kwDOR5y4QM67Yhp-
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate the `channel` path before the non-wait inbox path.**

`AgentChannelRecv` only rejects an empty string. With `wait=false`, `agentChannelInbox` reads the caller's whole inbox and filters locally, so an invalid channel identifier can come back as `200` with an empty message list instead of a validation error. That makes malformed channel IDs indistinguishable from real empty channels.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/agent_channels.go` around lines 85 - 103, AgentChannelRecv
currently only checks for empty channel and calls agentChannelInbox even when
wait=false, allowing malformed channel IDs to return 200 with an empty list;
validate the channel path/format (not just non-empty) before calling
agentChannelInbox by adding a proper channel format validation on the trimmed
channel variable and returning a 400 (using h.respondError) for invalid
identifiers, so agentChannelInbox, agentChannelMessagesFromEnvelopes,
parseBoolQuery and parsePositiveIntQuery are only invoked when the channel has
passed validation.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `AgentChannelRecv` only rejects an empty channel path. In non-wait mode, `agentChannelInbox` fetches the whole caller inbox and filters by channel locally, so malformed channel identifiers such as `bad.channel` can return `200` with an empty message list. The fix is to validate the trimmed path with the shared network channel grammar before reading the inbox.
- Resolution: Added `network.ValidateChannel` validation before inbox reads and covered invalid channel rejection in UDS receive tests; verified by focused tests and full `make verify`.
