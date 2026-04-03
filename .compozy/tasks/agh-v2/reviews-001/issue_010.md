---
status: resolved
file: internal/acp/client.go
line: 463
severity: medium
author: claude-code
provider_ref:
---

# Issue 010: waitForPromptQuiescence has no upper-bound timeout

## Review Comment

`waitForPromptQuiescence` (line 463) resets its timer on every activity signal. If the activity channel keeps receiving signals (e.g., from a stream of rapid session updates), this function will never return. There is no maximum total wait time.

```go
func (d *Driver) waitForPromptQuiescence(active *activePromptState) {
    timer := time.NewTimer(d.promptDrainWait)
    for {
        select {
        case <-timer.C:
            return
        case _, ok := <-active.activity:
            if !ok { return }
            timer.Reset(d.promptDrainWait)
        }
    }
}
```

**Suggested fix:** Add a maximum total wait time using `time.After` as a secondary select case, e.g. `2 * d.promptDrainWait` or a configurable cap.

## Triage

- Decision: `valid`
- Notes: `waitForPromptQuiescence()` only waits for a quiet period and resets the timer on every activity signal. If activity never stops, the function never returns, so prompt completion can hang forever. A total upper bound is needed.
