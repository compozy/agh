---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: internal/network/router.go
line: 947
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGV,comment:PRRC_kwDOR5y4QM69ZeEx
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep replay dedupe entries until the envelope is actually stale.**

This clamp reopens the replay window for accepted future-skewed envelopes. If `TS` is slightly ahead of `now`, validation still accepts the envelope until `TS + maxReplayAge`, but `markSeen` now evicts it at `now + maxReplayAge`, so the same message can be replayed and accepted again before it ages out.

 

<details>
<summary>Suggested fix</summary>

```diff
 func replayDeadline(envelope Envelope, now time.Time, maxReplayAge time.Duration) time.Time {
 	deadline := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
-	maxDeadline := now.Add(maxReplayAge).UTC()
-	if deadline.After(maxDeadline) {
-		deadline = maxDeadline
-	}
 	if envelope.ExpiresAt != nil {
 		expiresAt := time.Unix(*envelope.ExpiresAt, 0).UTC()
 		if expiresAt.Before(deadline) {
 			deadline = expiresAt
 		}
 	}
 	if deadline.Before(now) {
 		return now.UTC()
 	}
 	return deadline
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/router.go` around lines 935 - 947, Clamp must use the
envelope's original timestamp (TS) when computing maxDeadline so dedupe entries
aren't evicted based on current now; change the logic that sets maxDeadline
(currently maxDeadline := now.Add(maxReplayAge).UTC()) to compute the ceiling as
TS.Add(maxReplayAge).UTC() when the envelope timestamp is present (fall back to
now.Add(maxReplayAge) only if TS is missing), then continue the existing
expiresAt and deadline comparisons (using deadline, maxReplayAge,
envelope.ExpiresAt) and keep the final now check as-is.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `replayDeadline` clamps dedupe retention to `now + maxReplayAge`, which can evict future-skewed envelopes before the accepted replay window actually closes. That allows replay acceptance after the dedupe record is dropped.
- Fix plan: retain replay markers until the envelope’s real acceptance deadline and update the router/manager tests that currently encode the premature clamp.
