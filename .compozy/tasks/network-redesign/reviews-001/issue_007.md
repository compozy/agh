---
status: resolved
file: internal/api/core/network_details.go
line: 558
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeI,comment:PRRC_kwDOR5y4QM66CAkr
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Last message preview can disappear when metadata wins the timestamp race.**

`applyNetworkChannelMetadata()` can move `aggregate.lastActivityAt` past the newest message, and `aggregateMessageIsLatest()` then rejects every message preview. A just-created channel can hit this if the metadata row is written after the first envelope, leaving the room list with messages but no preview. Track the latest message timestamp separately from overall activity when deciding `lastMessagePreview`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 541 - 558, The issue is
that applyNetworkChannelMetadata can advance aggregate.lastActivityAt past the
newest message timestamp, causing aggregateMessageIsLatest to reject message
previews; fix by tracking the latest message timestamp separately (e.g., add a
field like latestMessageAt or lastMessageTimestamp to networkChannelAggregate),
update ensureNetworkChannelAggregate to initialize that field, update the loop
where messages are processed to set/advance aggregate.latestMessageAt based on
message.Timestamp (independent of lastActivityAt), and change
aggregateMessageIsLatest (or the preview check) to compare message.Timestamp
against aggregate.latestMessageAt instead of aggregate.lastActivityAt so
metadata updates no longer suppress previews.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `applyNetworkChannelMetadata()` advances `aggregate.lastActivityAt`, and `aggregateMessageIsLatest()` compares message timestamps against that same field. If metadata updates after the newest persisted message, preview selection can reject every message even though the channel has traffic.
- Fix plan: track latest message time independently from overall activity, base preview selection on that message-only timestamp, and add coverage proving metadata updates do not erase the preview.
- Resolution: split latest-message tracking from overall activity tracking, based preview selection on the newest message timestamp, and added regression coverage showing newer metadata does not erase message previews.
- Verification: `go test ./internal/api/core` and `make verify`
