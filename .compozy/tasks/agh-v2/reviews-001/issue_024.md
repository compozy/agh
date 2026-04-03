---
status: resolved
file: internal/session/manager.go
line: 708
severity: low
author: claude-code
provider_ref:
---

# Issue 024: Vacuous timestamp format check is dead code

## Review Comment

In `normalizeEvent` at line 708:

```go
if strings.TrimSpace(normalized.Timestamp.Format(time.RFC3339Nano)) == "" ||
    normalized.Timestamp.IsZero() {
```

`time.Time.Format(time.RFC3339Nano)` will never return an empty string for any `time.Time` value (even zero time formats to `"0001-01-01T00:00:00Z"`). The `TrimSpace` of a formatted timestamp will also never be empty. Only the `IsZero()` check is effective; the format+trim check is dead code that adds an unnecessary allocation.

**Suggested fix:** Simplify to `if normalized.Timestamp.IsZero() {`.

## Triage

- Decision: `valid`
- Notes: `strings.TrimSpace(normalized.Timestamp.Format(time.RFC3339Nano)) == \"\"` is vacuous because formatting a `time.Time` never returns an empty string. The only meaningful guard here is `IsZero()`, so the extra format+trim work is dead code.
