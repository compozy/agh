---
status: resolved
file: internal/workspace/resolver_crud.go
line: 163
severity: medium
author: claude-code
provider_ref:
---

# Issue 013: Workspace registration retry loop has no iteration bound

## Review Comment

`createWorkspaceRegistration` has a `for {}` loop that retries `InsertWorkspace` when `ErrWorkspaceNameTaken` occurs and no explicit name was provided. Each retry generates a new name via `nextWorkspaceName`, but if name generation collides repeatedly (small namespace or deterministic generation), this loop spins indefinitely. The only exit is context cancellation (checked at line 164).

**Fix:** Add a maximum retry count (e.g., 10 attempts) to prevent unbounded looping:

```go
const maxNameRetries = 10
for attempt := 0; ; attempt++ {
    if attempt >= maxNameRetries {
        return store.WorkspaceRow{}, fmt.Errorf("workspace: exceeded %d name retries", maxNameRetries)
    }
    // ...existing logic...
}
```

## Triage

- Decision: `invalid`
- Analysis: `nextWorkspaceName` computes a deterministic unique name from the current taken set and increments suffixes until it finds a gap. On retry, `createWorkspaceRegistration` recomputes against a refreshed taken set, so it makes forward progress under ordinary collisions.
- Analysis: An infinite loop would require perpetual concurrent collisions on every newly generated candidate, which is not a concrete bug in the current implementation and would remain possible even with an arbitrary retry cap under enough contention.
- Conclusion: The loop is intentionally open-ended to preserve correctness under transient concurrent inserts, and there is no bounded failure mode to fix here without introducing spurious registration failures.
