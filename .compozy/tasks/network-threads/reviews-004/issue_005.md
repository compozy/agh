---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/core/network_details.go
line: 1384
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0Drj,comment:PRRC_kwDOR5y4QM6-RRYc
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Preserve conversation routing fields in cloned timeline entries.**

`summarizeNetworkMessageHistory()` rebuilds the response timeline from `cloneNetworkMessageEntry()`, and `NetworkChannelMessagePayloadFromView()` later reads `Surface`, `ThreadID`, and `DirectID` from that cloned value. Because those fields are dropped here, thread/direct messages come back with empty routing metadata even when the store row has it.
 
<details>
<summary>Suggested fix</summary>

```diff
 func cloneNetworkMessageEntry(entry store.NetworkMessageEntry) store.NetworkMessageEntry {
 	return store.NetworkMessageEntry{
 		MessageID:   strings.TrimSpace(entry.MessageID),
 		SessionID:   strings.TrimSpace(entry.SessionID),
 		Channel:     strings.TrimSpace(entry.Channel),
+		Surface:     strings.TrimSpace(entry.Surface),
+		ThreadID:    strings.TrimSpace(entry.ThreadID),
+		DirectID:    strings.TrimSpace(entry.DirectID),
 		Direction:   strings.TrimSpace(entry.Direction),
 		PeerFrom:    strings.TrimSpace(entry.PeerFrom),
 		PeerTo:      strings.TrimSpace(entry.PeerTo),
 		Kind:        strings.TrimSpace(entry.Kind),
 		WorkID:      strings.TrimSpace(entry.WorkID),
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/network_details.go` around lines 1366 - 1384,
cloneNetworkMessageEntry is dropping routing fields (Surface, ThreadID,
DirectID) causing summarizeNetworkMessageHistory ->
NetworkChannelMessagePayloadFromView to see empty routing metadata; update
cloneNetworkMessageEntry to include those fields from the input entry (e.g.,
Surface, ThreadID, DirectID) in the returned store.NetworkMessageEntry, applying
the same strings.TrimSpace where appropriate so the cloned timeline preserves
conversation routing for summarizeNetworkMessageHistory and
NetworkChannelMessagePayloadFromView.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `cloneNetworkMessageEntry` currently drops `Surface`, `ThreadID`, and `DirectID`. `summarizeNetworkMessageHistory` rebuilds timeline entries from that clone, so later payload conversion can lose conversation routing metadata. Preserve the routing fields in the clone and cover the regression in tests.
