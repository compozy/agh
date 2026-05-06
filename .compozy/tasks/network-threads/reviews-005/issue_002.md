---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/api/core/network_details.go
line: 1160
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_033w,comment:PRRC_kwDOR5y4QM6-SXvZ
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep presence episodes scoped by surface/thread/direct/work.**

Now that Lines 1150-1160 carry routing metadata, `summarizeNetworkMessageHistory()` can no longer merge greet messages using only channel + peer pair. Two presence pings for the same peers in different threads/direct rooms/work items will collapse into one episode, and the merged row will inherit the later routing fields, which mislabels the earlier conversation.

 

<details>
<summary>Suggested fix</summary>

```diff
 type networkPresenceEpisodeKey struct {
 	direction string
 	channel   string
+	surface   string
+	threadID  string
+	directID  string
+	workID    string
 	peerFrom  string
 	peerTo    string
 }
@@
 	if strings.TrimSpace(current.entry.Channel) != strings.TrimSpace(next.Channel) {
 		return false
 	}
+	if strings.TrimSpace(current.entry.Surface) != strings.TrimSpace(next.Surface) {
+		return false
+	}
+	if strings.TrimSpace(current.entry.ThreadID) != strings.TrimSpace(next.ThreadID) {
+		return false
+	}
+	if strings.TrimSpace(current.entry.DirectID) != strings.TrimSpace(next.DirectID) {
+		return false
+	}
+	if strings.TrimSpace(current.entry.WorkID) != strings.TrimSpace(next.WorkID) {
+		return false
+	}
 	if strings.TrimSpace(current.entry.PeerFrom) != strings.TrimSpace(next.PeerFrom) {
 		return false
 	}
@@
 	return networkPresenceEpisodeKey{
 		direction: strings.TrimSpace(message.Direction),
 		channel:   strings.TrimSpace(message.Channel),
+		surface:   strings.TrimSpace(message.Surface),
+		threadID:  strings.TrimSpace(message.ThreadID),
+		directID:  strings.TrimSpace(message.DirectID),
+		workID:    strings.TrimSpace(message.WorkID),
 		peerFrom:  strings.TrimSpace(message.PeerFrom),
 		peerTo:    strings.TrimSpace(message.PeerTo),
 	}
 }
```
</details>


Also applies to: 1175-1218, 1319-1351

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/network_details.go` around lines 1150 - 1160,
summarizeNetworkMessageHistory() currently merges presence episodes using only
channel + peer pair, which collapses pings across different routing contexts;
change the episode grouping key to include routing metadata Surface, ThreadID,
DirectID and WorkID (and keep PeerFrom/PeerTo as before) so episodes are scoped
by surface/thread/direct/work; when merging choose and preserve the routing
fields from the earliest message in the episode (not the later one) so earlier
rows aren't relabeled; apply the same fix to the other merge/episode-creation
sites in this file where similar logic appears (the nearby presence-merge blocks
that handle DisplayName/SessionID/Local/WorkID).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `summarizeNetworkMessageHistory()` groups presence episodes by `direction + channel + peerFrom + peerTo` only, and `canExtendPresenceEpisode()` enforces the same narrow match.
  - That lets greet/presence rows from different routing containers merge into one episode, after which `extendPresenceEpisode()` replaces the stored entry with the later row and exposes the wrong `surface/thread/direct/work` metadata for the earlier activity.
  - Fix plan: scope presence episode identity and extension checks by `Surface`, `ThreadID`, `DirectID`, and `WorkID`, then add regression coverage for peer timeline payloads that mix routing contexts.

## Resolution

- Scoped presence episode keys and merge checks by `surface`, `thread_id`, `direct_id`, and `work_id` in `internal/api/core/network_details.go`.
- Added regression coverage in `internal/api/httpapi/network_test.go` to prove thread and direct greet episodes from the same peer stay separate in the API payload.
- Verified with fresh full `make verify` (passed).
