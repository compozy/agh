---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: internal/network/router.go
line: 943
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjm,comment:PRRC_kwDOR5y4QM69Zj0N
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Replay dedupe can be bypassed at the replay-window boundary.**

At **Line 942**, returning `now.UTC()` can make the seen-entry expire immediately (`markSeen` deletes entries when `expiresAt <= now`). Because freshness uses Unix-second checks, a boundary envelope can still be valid for the rest of that second and be replayed repeatedly.

 

<details>
<summary>Suggested fix</summary>

```diff
 func replayDeadline(envelope Envelope, now time.Time, maxReplayAge time.Duration) time.Time {
 	deadline := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
 	if envelope.ExpiresAt != nil {
 		expiresAt := time.Unix(*envelope.ExpiresAt, 0).UTC()
 		if expiresAt.Before(deadline) {
 			deadline = expiresAt
 		}
 	}
-	if deadline.Before(now) {
-		return now.UTC()
+	minDeadline := time.Unix(now.Unix()+1, 0).UTC()
+	if deadline.Before(minDeadline) {
+		return minDeadline
 	}
 	return deadline
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/router.go` around lines 941 - 943, The branch that currently
returns now.UTC() when deadline.Before(now) causes expiresAt to be <= now and
allows replay dedupe to be bypassed; modify the if branch in the deadline
handling (the code using variables deadline and now, related to
markSeen/replay-window) to return now.Add(time.Second).UTC() instead so the
computed expiry is strictly in the future (one second) and cannot be treated as
already expired by markSeen's expiresAt <= now check.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `replayDeadline()` still returns `now.UTC()` once the computed deadline is in the past, while `markSeen()` deletes entries whose expiry is `<= now`. That means a boundary-time envelope can be marked seen and immediately swept, allowing the same envelope ID to be replayed again within the same Unix-second freshness window.
- Fix approach: clamp the replay deadline to a strictly future instant and extend the router test coverage to prove replay retention survives that boundary case.
- Resolution: replay deadlines now clamp to at least the next Unix second, and the router regression test covers the boundary case where an already-past expiry would otherwise be swept immediately.
- Verification: `go test ./extensions/bridges/teams ./internal/network -count=1 -race`, `make verify`.
