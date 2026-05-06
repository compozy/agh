---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/acp/client.go
line: 852
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0Dre,comment:PRRC_kwDOR5y4QM6-RRYX
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't swallow unrelated prompt failures just because stop was requested.**

`stopWasRequested()` is a coarse process-wide flag. If `SendRequest` races with shutdown and returns a real transport/protocol failure, this branch drops the error event entirely, so callers just see the prompt stream close with no failure classification. Suppress only the expected cancellation/stop-shaped errors during shutdown and keep emitting/classifying everything else.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/client.go` around lines 849 - 852, The current error branch in
SendRequest/wherever the block uses proc.stopWasRequested() swallows all errors
during shutdown; instead, only suppress errors that are actual cancellation/stop
errors. Change the logic so when err != nil you check if proc.stopWasRequested()
AND the err is a cancellation-type error (e.g., errors.Is(err, context.Canceled)
or context.DeadlineExceeded, or an internal stop sentinel) before returning;
otherwise continue to emit/classify the error as normal. Update the error
handling around SendRequest/err/proc.stopWasRequested() to use
errors.Is/appropriate sentinel checks so unrelated transport/protocol failures
are not dropped.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `Driver.Prompt` currently drops every prompt error whenever `proc.stopWasRequested()` is true. The branch at `internal/acp/client.go:849-852` suppresses unrelated transport/protocol failures during shutdown, so callers can lose the final failure classification. Fix by suppressing only stop-shaped errors and keep emitting classified error events for everything else.
