---
status: resolved
file: internal/scheduler/scheduler.go
line: 534
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vQ,comment:PRRC_kwDOR5y4QM67Z0NL
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Channel-bound runs can still match unscoped sessions.**

`coordinationChannelMatches` returns `true` when `candidate.Channel` is empty, so a run with `CoordinationChannelID` can still be routed to sessions outside that channel.



<details>
<summary>Proposed fix</summary>

```diff
 func coordinationChannelMatches(work *RunSnapshot, candidate SessionSnapshot) bool {
     if work == nil {
         return false
     }
+    runChannel := strings.TrimSpace(work.Run.CoordinationChannelID)
     sessionChannel := strings.TrimSpace(candidate.Channel)
-    if sessionChannel == "" {
+    if runChannel == "" {
         return true
     }
-    return strings.TrimSpace(work.Run.CoordinationChannelID) == sessionChannel
+    return sessionChannel == runChannel
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    if !coordinationChannelMatches(work, candidate) {
        return false
    }
    return capabilitiesCover(candidate.Capabilities, work.Run.RequiredCapabilities)
}

func coordinationChannelMatches(work *RunSnapshot, candidate SessionSnapshot) bool {
    if work == nil {
        return false
    }
    runChannel := strings.TrimSpace(work.Run.CoordinationChannelID)
    sessionChannel := strings.TrimSpace(candidate.Channel)
    if runChannel == "" {
        return true
    }
    return sessionChannel == runChannel
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/scheduler/scheduler.go` around lines 519 - 534,
coordinationChannelMatches currently treats an empty candidate.Channel as a
wildcard and returns true, which allows a run with a non-empty
Run.CoordinationChannelID to match an unscoped session; update
coordinationChannelMatches (and its use of SessionSnapshot.Channel and
RunSnapshot.Run.CoordinationChannelID) so that an empty session channel only
matches when the run's CoordinationChannelID is also empty (i.e., trim both
values and return true only if both are empty or both are equal), keeping the
existing nil guard for work.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `coordinationChannelMatches` treats an empty candidate session channel as a wildcard even when the run is bound to a coordination channel. That can wake an unscoped session for channel-bound work. Fix by trimming both values and requiring equality when the run channel is non-empty; empty session channels only match unbound runs.
