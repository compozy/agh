---
status: resolved
file: internal/observe/health.go
line: 59
severity: high
author: claude-code
provider_ref:
---

# Issue 018: Observe health always reports wrong active_agents count

## Review Comment

`activeCounts()` returns the same `count` for both `active_sessions` and `active_agents` in both code paths:

```go
return count, count, nil
```

That makes `/api/observe/health` inaccurate whenever multiple active sessions share the same agent name, or when active sessions and active agents diverge for any other reason. The API advertises both fields separately, so callers will assume they are independently computed metrics.

The current tests only cover cases where the two numbers happen to be equal, so this bug is not exercised.

**Fix:** Compute `active_agents` from the distinct non-empty agent names among active sessions instead of reusing the session count. Add a test with two active sessions using the same agent and assert `active_sessions == 2` while `active_agents == 1`.

## Triage

- Decision: `valid`
- Root cause: `activeCounts` returns the active-session count for both metrics and never computes distinct active agents. That makes `/api/observe/health` incorrect whenever multiple active sessions share an agent name.
- Fix approach: Count distinct non-empty agent names separately in both the in-memory and registry-backed code paths, and add a regression test where sessions and agents intentionally diverge.
- Resolution: Implemented with distinct-agent counting and regression coverage; full repository verification passed.
