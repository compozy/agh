---
status: resolved
file: internal/acp/types.go
line: 297
severity: high
author: claude-code
provider_ref:
---

# Issue 001: emitPromptEvent can deadlock holding RLock on full channel

## Review Comment

`emitPromptEvent` acquires `promptMu.RLock()`, then sends on `active.events` (line 304) while still holding the lock. If the consumer of the channel is slow and the buffered channel is full, this send blocks indefinitely while holding the read lock. This prevents `endPrompt` (which needs the write lock via `promptMu.Lock()`) from ever executing, creating a deadlock.

```go
func (p *AgentProcess) emitPromptEvent(event AgentEvent) {
    p.promptMu.RLock()
    active := p.activePrompt
    // ...
    active.events <- event  // blocks while holding RLock
    // ...
    p.promptMu.RUnlock()
}
```

**Suggested fix:** Capture `active` under the lock, release it, then send:

```go
p.promptMu.RLock()
active := p.activePrompt
p.promptMu.RUnlock()
if active == nil {
    return
}
active.events <- event
```

## Triage

- Decision: `valid`
- Notes: `emitPromptEvent` holds `promptMu.RLock()` across a potentially blocking send to `active.events`. `endPrompt` must take the write lock before it can clear and close the prompt, so a full channel can stall prompt teardown indefinitely. The fix needs to preserve close/send safety without holding `promptMu` during the blocking send.
