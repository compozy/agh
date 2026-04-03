---
status: resolved
file: internal/kernel/kernel.go
line: 798
severity: medium
author: claude-reviewer
---

# Issue 092: API error routing uses fragile string matching on error messages



## Review Comment

Many API route handlers in `registerKernelRoutes` use `strings.Contains(err.Error(), ...)` to determine HTTP status codes. While the session start/detail/delete/resume routes have been refactored to use proper typed error helpers (`sessionCreateAPIError`, `sessionReadAPIError`, `sessionResumeAPIError`) that use `errors.Is()`, numerous other routes still rely on fragile string matching:

```go
// Workgroup create (line 798)
if strings.Contains(err.Error(), "not found") {
    statusCode = http.StatusNotFound
}

// Workgroup destroy (line 816)
if strings.Contains(err.Error(), "not found") {
    statusCode = http.StatusNotFound
}

// Agent spawn (line 846-848)
case strings.Contains(err.Error(), "session"):
    writeAPIError(c, http.StatusNotFound, err.Error())
case strings.Contains(err.Error(), "not found"):
    writeAPIError(c, http.StatusNotFound, err.Error())

// Agent kill (line 866)
if strings.Contains(err.Error(), "not found") {

// Direct message (line 895), Broadcast (line 923), Escalate (line 951)
if strings.Contains(err.Error(), "not found") {

// Blackboard append (line 979), Context view (line 997),
// Agent status read (line 1015), Agent status update (line 1043),
// Agent done (line 1071), Events read (line 1101)
if strings.Contains(err.Error(), "not found") {
```

This pattern is fragile because:
1. It couples HTTP status code selection to the exact wording of error messages from lower layers.
2. If error messages change (e.g., internationalization, rewording), the wrong HTTP status will be returned.
3. The project's CLAUDE.md explicitly states: "Use errors.Is() and errors.As() for error matching; do not compare error strings."

The session-level routes (`sessionCreateAPIError`, `sessionReadAPIError`, `sessionResumeAPIError` at lines 1202-1241) are already good examples of the correct pattern using `errors.Is()` with sentinel errors like `ErrSessionNotFound` and `ErrSessionExists`. The remaining routes (workgroups, agents, messaging, blackboard, context, status) should adopt the same approach.

## Triage

- Decision: `valid`
- Notes: Confirmed in `registerKernelRoutes`: the non-session API handlers still branch on `strings.Contains(err.Error(), "not found")` and one agent-spawn path also checks for `"session"` in the raw message. The lower layers already centralize most entity resolution through a handful of helpers, so this is an actionable root-cause fix by introducing typed/wrapped not-found errors and routing with `errors.Is`.
