---
provider: coderabbit
pr: "105"
round: 6
round_created_at: 2026-05-06T03:03:04.040959Z
status: resolved
file: internal/api/core/network_details.go
line: 1164
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_1nZG,comment:PRRC_kwDOR5y4QM6-TX8T
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Treat `surface=direct` / `direct_id` as directed traffic in the visibility filters.**

These payload changes carry privacy/routing on `Surface` and `DirectID`, but `isDirectedChannelMessage()` still keys entirely off `PeerTo`. A direct-room `say` message with `surface="direct"` and no `peer_to` — like the new assertion case in `internal/daemon/network_e2e_assertions_test.go` lines 174-203 — will be treated as a public channel message and can also disappear from directed timelines. Please fold `SurfaceDirect` / `DirectID` into the directed-message check.
 

<details>
<summary>Suggested fix</summary>

```diff
 func isDirectedChannelMessage(message store.NetworkMessageEntry) bool {
-	return strings.TrimSpace(message.PeerTo) != ""
+	if strings.TrimSpace(message.PeerTo) != "" {
+		return true
+	}
+	if strings.TrimSpace(message.DirectID) != "" {
+		return true
+	}
+	return strings.TrimSpace(message.Surface) == string(network.SurfaceDirect)
 }
```
</details>


Also applies to: 1459-1468

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/network_details.go` around lines 1154 - 1164, The
visibility logic currently only checks PeerTo inside isDirectedChannelMessage(),
so messages with Surface == "direct" or a non-empty DirectID are misclassified;
update isDirectedChannelMessage() (and any similar directed-message checks used
elsewhere) to also treat entries as directed when entry.Surface equals the
SurfaceDirect value (or literal "direct") or when entry.DirectID is
non-empty/trimmed, and ensure the same change is applied to the parallel check
used around the other mapping path noted (the second occurrence that mirrors
isDirectedChannelMessage()). Trim whitespace when reading Surface/DirectID to
match the struct population.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `internal/api/core/network_details.go` currently classifies directed channel traffic exclusively through `PeerTo`. That misclassifies direct-room messages that carry routing via `surface=direct` and `direct_id` but intentionally omit `peer_to`, which can leak them into public-channel visibility and hide them from directed timelines. I will update the shared directed-message predicate to treat non-empty `direct_id` or `surface=direct` as directed traffic, preserving whitespace trimming semantics used elsewhere in the file.
  Resolved by extending `isDirectedChannelMessage()` to recognize trimmed `direct_id` and `surface=direct`, then re-verifying the batch with `make verify`.
