---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 918
severity: medium
author: claude-reviewer
---

# Issue 155: OpenCode SSE event stream silently swallows ParseHookEvent errors



## Review Comment

In `handleStreamEvent`, when `ParseHookEvent` fails, the error is silently discarded:

```go
if _, err := d.ParseHookEvent(payload); err != nil {
    return nil
}
```

This means that if the SSE stream delivers events that cannot be parsed (due to schema changes, new event types, or data corruption), the driver silently drops them with no logging or error reporting. This makes debugging SSE integration issues extremely difficult in production.

The same method also silently returns `nil` for empty payloads and non-matching sessions, which is reasonable. But parsing failures represent a potential protocol mismatch that operators need to know about.

**Suggested fix**: Add structured logging (using `slog`) for parse failures so operators can diagnose SSE integration issues. The error should be logged at debug or warn level, not returned (since returning it would kill the entire stream):

```go
if _, err := d.ParseHookEvent(payload); err != nil {
    slog.Debug("opencode: skipping unparseable SSE event", "error", err)
    return nil
}
```

This would require adding a logger field to the Driver struct or accepting a logger parameter.

## Triage

- Decision: `valid`
- Notes:
  - `handleStreamEvent` intentionally treats parse failures as non-fatal, but dropping them silently removes the only operator-visible signal that the SSE protocol or payload schema no longer matches expectations.
  - This is an observability defect in production behavior: malformed or drifted events disappear without any trace, which makes diagnosing broken hook ingestion unnecessarily hard.
  - The correct fix is to keep the stream alive while emitting structured debug/warn logging for parse skips.
  - Resolution: OpenCode now emits structured debug logs for skipped unparseable SSE events and has targeted coverage in `internal/drivers/opencode/opencode_test.go`.
