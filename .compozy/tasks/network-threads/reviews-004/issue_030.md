---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/router.go
line: 1084
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0DsT,comment:PRRC_kwDOR5y4QM6-RRZc
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don’t skip direct-room identity checks for directed messages.**

The `envelope.IsDirected()` short-circuit means any `surface=direct` envelope is delivered to the addressed local peer even if `direct_id` was derived from a different peer pair. For non-work direct chat there is no other guard, so a sender can inject messages into the wrong direct-room timeline just by choosing `to`.

<details>
<summary>💡 Suggested fix</summary>

```diff
 func localPeerMatchesConversation(peer LocalPeer, envelope Envelope) bool {
-	if envelope.Surface == nil || *envelope.Surface != SurfaceDirect || envelope.IsDirected() {
+	if envelope.Surface == nil || *envelope.Surface != SurfaceDirect {
 		return true
 	}
 	if envelope.DirectID == nil {
 		return false
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/network/router.go` around lines 1072 - 1084, The current
localPeerMatchesConversation short-circuits when envelope.IsDirected() is true,
which allows directed (surface=direct) messages to bypass direct-room identity
validation; update localPeerMatchesConversation so that for SurfaceDirect
envelopes you always compute DirectRoomIdentity (call
DirectRoomIdentity(envelope.Channel, envelope.From, peer.PeerID)) and compare
the resulting directID to strings.TrimSpace(*envelope.DirectID) before returning
true, even when envelope.IsDirected() is true; keep the existing nil checks
(envelope.Surface, envelope.DirectID) and still return false on
DirectRoomIdentity errors, but do not return true solely because IsDirected() is
set.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `localPeerMatchesConversation` short-circuits on `envelope.IsDirected()`, so directed direct-room messages skip the direct-room identity check entirely. That lets `surface=direct` traffic target a local peer by `to` even when `direct_id` belongs to a different peer pair.
- Fix approach: always validate `direct_id` for `surface=direct` envelopes, including directed ones, and add a regression test proving mismatched direct-room identity is not delivered.
- Verification: fixed in scoped code and validated with fresh `make verify`.
