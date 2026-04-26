---
status: resolved
file: internal/automation/dispatch.go
line: 529
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQg,comment:PRRC_kwDOR5y4QM67VX6-
---

# Issue 002: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Don't let fire-limit rejections count against the next fire-limit window.**

`reserveExistingRun()` now persists scheduled fire-limit hits as `RunCancelled`, but `evaluateFireLimit()` counts every run returned by `ListRuns()` regardless of status. That means the rejection itself consumes a slot, so a deferred scheduler can keep deferring forever once the window is saturated. The fire-limit query needs to exclude these cancellations or otherwise count only runs that actually entered execution.



Also applies to: 539-597

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 521 - 529, evaluateFireLimit()
is currently counting runs returned by ListRuns() regardless of status, so when
reserveExistingRun() persists a fire-limit hit as RunCancelled it still consumes
a slot; update evaluateFireLimit() (and any other fire-limit counting logic) to
filter out cancelled runs (e.g., exclude status RunCancelled or any
cancel-by-fire-limit marker) or only count runs that have entered execution
(started/running/completed states that should count toward the window) when
computing the fire-limit; ensure ListRuns() is called with the adjusted status
filter and that logic in reserveExistingRun()/finishRun()/fireLimitRunStatus()
remains consistent with the new exclusion so cancellations no longer decrement
available slots.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `evaluateFireLimit()` currently counts every run returned by `ListRuns()` and does not exclude fire-limit cancellations recorded as `RunCancelled` for deferred scheduled fires.
  - Root cause: scheduled fire-limit rejections are persisted through `finishRun(..., RunCancelled, ...)`, but the fire-limit window logic uses raw run count instead of filtering to statuses that should consume the limit.
  - Fix plan: exclude canceled fire-limit reservation runs from the window count and retry-at calculation, then add a dispatch regression showing that a canceled reserved run does not block the next window.
