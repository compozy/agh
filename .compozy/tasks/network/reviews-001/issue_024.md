---
status: resolved
file: internal/network/router.go
line: 595
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZm,comment:PRRC_kwDOR5y4QM623eZ3
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Clamp replay entries to the configured replay window.**

If a sender sets a far-future `expires_at`, this keeps the ID in `r.seen` until that time instead of the configured `maxReplayAge`. That makes duplicate suppression last longer than configured and lets a peer pin replay-cache entries for arbitrarily long periods.

<details>
<summary>🐛 Suggested fix</summary>

```diff
 func replayDeadline(envelope Envelope, now time.Time, maxReplayAge time.Duration) time.Time {
+	ceiling := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
 	if envelope.ExpiresAt != nil {
-		return time.Unix(*envelope.ExpiresAt, 0).UTC()
+		deadline := time.Unix(*envelope.ExpiresAt, 0).UTC()
+		if deadline.After(ceiling) {
+			deadline = ceiling
+		}
+		if deadline.Before(now) {
+			return now.Add(maxReplayAge).UTC()
+		}
+		return deadline
 	}
-	deadline := time.Unix(envelope.TS, 0).Add(maxReplayAge).UTC()
+	deadline := ceiling
 	if deadline.Before(now) {
 		return now.Add(maxReplayAge).UTC()
 	}
 	return deadline
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/router.go` around lines 587 - 595, The replayDeadline
function currently uses Envelope.ExpiresAt verbatim, letting peers set
far-future expiries; instead clamp any provided expires_at to the configured
replay window. Change replayDeadline: compute tsMax :=
time.Unix(envelope.TS,0).Add(maxReplayAge).UTC(), and if Envelope.ExpiresAt !=
nil compute expires := time.Unix(*envelope.ExpiresAt,0).UTC() and set deadline =
min(expires, tsMax); otherwise set deadline = tsMax; keep the existing safety
that if deadline.Before(now) you return now.Add(maxReplayAge).UTC(). This
ensures r.seen entries (replayDeadline) never outlive maxReplayAge.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `replayDeadline` currently trusts a peer-supplied `expires_at` verbatim. That lets a sender keep an entry in `r.seen` beyond the configured replay window, which defeats the local `maxReplayAge` bound and allows remote peers to pin replay-cache state arbitrarily long. The fix is to clamp any explicit expiry to `envelope.TS + maxReplayAge` while keeping the existing stale-deadline fallback.
  Resolved by clamping replay deadlines in `internal/network/router.go`. Because the existing regression lived in `internal/network/router_test.go`, I made a minimal out-of-scope test update there and added an in-scope focused assertion in `internal/network/manager_test.go`. Verified with package tests and a clean `make verify`.
