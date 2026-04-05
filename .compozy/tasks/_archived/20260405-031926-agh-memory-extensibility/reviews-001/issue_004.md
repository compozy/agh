---
status: resolved
file: internal/memory/staleness.go
line: 9
severity: medium
author: claude-code
provider_ref:
---

# Issue 004: AgeDays() uses non-injectable time.Now()

## Review Comment

`AgeDays()` (line 9) calls `time.Now()` directly, while the rest of the memory package (dream service, consolidation lock) and the broader codebase (session manager, daemon) consistently use injectable `now func() time.Time` clocks.

This creates two problems:

1. **Testing fragility**: The existing test (`TestStalenessHelpers`) works by creating times relative to `time.Now()`, which makes it dependent on wall-clock time. Tests that run at midnight boundaries could produce flaky results due to the calendar-day calculation in `calendarDayNumber()`.

2. **Inconsistency**: `FreshnessWarning()` is used in prompt assembly context where deterministic behavior matters. When assembler tests verify prompt output, staleness warnings may vary.

**Suggested fix**: Accept a `time.Time` parameter for the "now" reference point:

```go
func AgeDays(modTime time.Time, now time.Time) int {
    days := calendarDayNumber(now.In(modTime.Location())) - calendarDayNumber(modTime)
    if days < 0 {
        return 0
    }
    return days
}
```

Callers pass their injectable clock value. This aligns with the established pattern throughout the codebase.

## Triage

- Decision: `valid`
- Root cause: `AgeDays()` hard-codes `time.Now()`, which makes the staleness helpers wall-clock dependent even though the rest of the memory subsystem is designed around injectable clocks.
- Evidence: `internal/memory/staleness.go` uses `time.Now()` directly, and the current test builds expectations from the current date instead of a fixed reference point.
- Fix approach: Make the staleness helpers accept an explicit `now` reference and update the tests to use fixed timestamps so the calendar-day behavior is deterministic.
- Resolution: The staleness helpers now accept an explicit reference time and the tests use fixed timestamps, removing the wall-clock dependency.
