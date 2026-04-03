---
status: resolved
file: internal/dashboard/api.go
line: 271
severity: low
author: claude-reviewer
---

# Issue 128: Redundant strings.ToLower call in statusCodeForReadError



## Review Comment

In `statusCodeForReadError`, line 267 computes `message := strings.ToLower(err.Error())`, but line 271 calls `strings.ToLower(err.Error())` again instead of reusing the `message` variable:

```go
func statusCodeForReadError(err error) int {
    if err == nil {
        return http.StatusOK
    }
    message := strings.ToLower(err.Error())    // line 267
    if strings.Contains(message, "not running") {
        return http.StatusConflict
    }
    if strings.Contains(strings.ToLower(err.Error()), "not found") {  // line 271: redundant ToLower
        return http.StatusNotFound
    }
    return http.StatusBadRequest
}
```

Line 271 should use `message` instead of `strings.ToLower(err.Error())`.

**Suggested fix:**
```go
if strings.Contains(message, "not found") {
```

## Triage

- Decision: `invalid`
- Notes: The redundant `strings.ToLower` call is real but it is not the substantive bug in this area. The actual defect is the broader string-based error classification captured by issue `121`; fixing that removes this redundancy entirely. Treating this as a separate actionable defect would duplicate the same root-cause remediation.
