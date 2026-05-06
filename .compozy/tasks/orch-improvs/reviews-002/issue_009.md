---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/session/query_test.go
line: 623
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-Z,comment:PRRC_kwDOR5y4QM6-VcC2
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**The pre-finalization check can pass without proving `openQueryRecorder` actually blocked.**

`started` is closed before the goroutine calls `openQueryRecorder`, so the immediate non-blocking `select` only proves the goroutine was scheduled. A regression that returns early can still slip through if the goroutine has not written to `resultCh` yet. Please add a synchronization point reached after the call enters its wait path, or give the goroutine a short bounded window to return before asserting it is blocked.
 
As per coding guidelines, "Ensure tests can fail when business logic changes".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/session/query_test.go` around lines 595 - 623, The current test only
closes started before launching the goroutine so the immediate select can pass
even if openQueryRecorder never began waiting; modify the goroutine around
h.manager.openQueryRecorder(session.ID) to signal a new synchronization channel
(e.g., enteredWait or waitingStarted) after openQueryRecorder has entered its
wait path (or right after calling openQueryRecorder but before blocking) and
have the main test block on that channel before performing the non-blocking
select; alternatively give the main test a short bounded window (using a
time.After deadline) to allow the goroutine to return before asserting it is
blocked—ensure you reference the existing started/resultCh/openQueryRecorder
variables and still send the final result on resultCh and call cleanup as
before.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the test closes `started` before the goroutine can block in `openQueryRecorder`, so the immediate `select` does not prove the call actually stayed blocked.
- Fix approach: Add a bounded wait window that can catch premature returns before finalization completes, keeping the test deterministic and aligned with the real wait path.
