---
status: resolved
file: internal/api/core/network_details.go
line: 377
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg3x,comment:PRRC_kwDOR5y4QM63ZMHa
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**History-only channels never make it into the list.**

This only increments message counters for channels that were already seeded from live sessions or peers. A channel with persisted messages but no current runtime presence is skipped entirely, so it disappears from the channels page even though `NetworkChannelMessages` can still serve its history.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 365 - 377, The loop over
messages skips channels not already present in aggregates so history-only
channels are never included; modify the block inside ListNetworkMessages
handling so that when aggregates[strings.TrimSpace(message.Channel)] is not
found you create and insert a new aggregate entry (populate channel name,
initialize messageCount to 0 and lastMessageAt nil), then increment
aggregate.messageCount and set aggregate.lastMessageAt =
laterTimePtr(aggregate.lastMessageAt, message.Timestamp); reference the
aggregates map, message.Channel/message.Timestamp, aggregate.messageCount,
aggregate.lastMessageAt, and laterTimePtr when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: channel aggregation is seeded only from live sessions and runtime peers, then persisted messages are counted only for channels already present in that map, so history-only channels disappear from the list.
- Fix approach: create an aggregate entry when persisted messages reference a previously unseen channel and then apply the message count / last-message timestamp.
