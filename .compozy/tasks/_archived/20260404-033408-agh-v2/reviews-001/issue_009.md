---
status: resolved
file: internal/acp/types.go
line: 327
severity: medium
author: claude-code
provider_ref:
---

# Issue 009: Subprocess stderr buffer grows without limit (OOM risk)

## Review Comment

The `lockedBuffer` used for `cmd.Stderr` (line 188, `Write` at line 327) appends all stderr output without any size limit:

```go
func (b *lockedBuffer) Write(p []byte) (int, error) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.b = append(b.b, p...)
    return len(p), nil
}
```

A misbehaving agent subprocess could flood stderr with gigabytes of data, causing OOM. Compare with `managedTerminal.appendOutput` which correctly caps at `defaultTerminalOutputLimit` (64KB).

**Suggested fix:** Apply a similar ring-buffer or truncation strategy, capping at e.g. 64KB or 256KB, consistent with the terminal output limit pattern.

## Triage

- Decision: `valid`
- Notes: `lockedBuffer.Write()` appends stderr bytes without any upper bound. A noisy or malicious subprocess can therefore grow memory usage indefinitely. The terminal path already uses bounded output retention; stderr capture should do the same.
