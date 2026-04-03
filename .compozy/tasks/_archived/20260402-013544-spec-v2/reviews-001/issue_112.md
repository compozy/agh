---
status: resolved
file: internal/cli/state.go
line: 80
severity: low
author: claude-reviewer
---

# Issue 112: state read command silently allows negative --limit values



## Review Comment

The `newStateReadCommand` (line 36) and `newEventsCommand` (line 246) both accept a `--limit` flag as an `int` but do no client-side validation that the value is positive. The `--limit` flag is declared at line 80 for `newStateReadCommand` and line 293 for `newEventsCommand`.

While the `ReadBlackboard` client method only adds the limit to the query if `options.Limit > 0`, a negative value will simply be ignored (treated as no limit), which may be confusing to users.

Similarly, the `events` command has the same pattern. The kernel-side `BlackboardReadOptions.Validate()` rejects negative limits, but the CLI never calls Validate -- it sends the request and lets the server reject it. This means the error message will come from the server rather than the CLI, which provides a worse user experience.

Consider adding a check:

```go
if limit < 0 {
    return userCommandError(errors.New("--limit must be a positive integer"))
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in both `newStateReadCommand` and `newEventsCommand`: the CLI forwards negative `--limit` values without local validation. The server-side validators reject them, but the CLI can catch this earlier and return a clearer user-facing error without a round trip.
