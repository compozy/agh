---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/hooks/dispatch_events_test.go
line: 130
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_033x,comment:PRRC_kwDOR5y4QM6-SXva
---

# Issue 007: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Align these table-driven cases with the repo test conventions.**

These cases still use terse names like `"input"` / `"network peer"`, and the subtests at Lines 273 and 408 run serially. Please switch the case names to `t.Run("Should ...")` and add `t.Parallel()` to the subtests that do not intentionally opt out.

 

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures" and "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)."


Also applies to: 146-277, 283-414

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/hooks/dispatch_events_test.go` around lines 64 - 130, Rename the
subtest names in the table-driven cases (the cases slice used by the Test that
calls TurnIDFromPayload) from terse labels like "input" to descriptive "Should
..." phrases (e.g., "Should return turn ID for input payload") and ensure each
subtest goroutine calls t.Parallel() at the top of the t.Run body unless the
subtest intentionally opts out; update the cases variable entries' name fields
and add t.Parallel() inside the anonymous t.Run closures that invoke
TurnIDFromPayload; apply the same change pattern to the other table-driven
subtests in this file that the review references (the other cases blocks around
the file) so all follow the "Should ..." naming and default to t.Parallel().
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The table-driven cases in `dispatch_events_test.go` still use terse names like `input`, `prompt`, and `network peer`, and the loops at lines 273 and 408 do not call `t.Parallel()` even though they do not mutate env or shared process-wide state.
  - The repo's AGH test conventions require `Should ...` names and parallel subtests by default for independent cases.
  - Fix plan: rename the table rows to descriptive `Should ...` phrases and add `t.Parallel()` inside the eligible `t.Run` bodies.

## Resolution

- Renamed the touched table-driven cases in `internal/hooks/dispatch_events_test.go` to descriptive `Should ...` phrases.
- Added `t.Parallel()` to the independent `SessionContextFromPayload` and `CorrelationFromPayload` subtests.
- Verified with fresh full `make verify` (passed).
