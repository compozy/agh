---
status: resolved
file: internal/memory/consolidation/runtime.go
line: 403
severity: medium
author: claude-code
provider_ref:
---

# Issue 012: Dream session prompt errors silently discarded

## Review Comment

`spawnSession` drains all events with `for range events {}` but never inspects them. If the dream agent encounters an error during consolidation (e.g., permission denied, tool failure), the error is silently discarded and the function returns `nil`. The caller (`resolveWorkspaces` loop) proceeds thinking consolidation succeeded.

```go
events, err := sessions.Prompt(ctx, dreamSession.ID, prompt)
if err != nil {
    return fmt.Errorf("daemon: prompt dream session %q: %w", dreamSession.ID, err)
}

for range events {} // all events including errors silently dropped
return nil
```

**Fix:** Check events for error types and return an aggregated error, or at minimum log error-type events:

```go
for ev := range events {
    if ev.Type == "error" {
        slog.Warn("dream consolidation error", "session", dreamSession.ID, "error", ev.Error)
    }
}
```

## Triage

- Decision: `valid`
- Root cause: `spawnSession` drains the prompt event stream but never inspects it for error events, so dream-consolidation failures can be swallowed and reported as success.
- Fix approach: Inspect streamed events, capture any error payloads, and return an aggregated failure instead of silently discarding them.
- Resolution: Implemented with error aggregation and regression coverage; full repository verification passed.
