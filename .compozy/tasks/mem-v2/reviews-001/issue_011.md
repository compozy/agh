---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/cli/memory.go
line: 1000
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isw,comment:PRRC_kwDOR5y4QM6-UFV7
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Unused variable `fromDLQ` assigned but never used**

Similar to the previous issue, `fromDLQ` flag is captured on line 986 but assigned to blank identifier on line 1000 (`_ = fromDLQ`). This indicates incomplete implementation.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/memory.go` at line 1000, The variable fromDLQ is being captured
but then suppressed with a blank identifier assignment (`_ = fromDLQ`), leaving
an incomplete implementation; remove the dummy assignment and either use the
fromDLQ flag in the memory command flow (e.g., pass it into the function that
lists/filters memories or DLQ handling logic where flags like fromDLQ were
intended) or, if the flag is not needed, remove its capture/definition entirely
(search for the fromDLQ variable and the surrounding flag parsing/handler code
to apply the fix).
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `newMemoryExtractorReplayCommand` still registers `--from-dlq`, stores it in `fromDLQ`, and discards it with `_ = fromDLQ`.
  - The current request surface only supports `failure_id` / `session_id`, so the flag cannot influence daemon behavior and is dead CLI surface.
  - Fix approach: remove the unused flag and dead local variable. This may require a minimal test update outside the scoped file list if CLI command-shape tests currently mention the flag.
