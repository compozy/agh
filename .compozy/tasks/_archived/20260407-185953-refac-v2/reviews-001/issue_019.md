---
status: resolved
file: internal/api/core/handlers.go
line: 185
severity: medium
author: claude-code
provider_ref:
---

# Issue 019: StopSession violates the refac-v2 204 delete contract

## Review Comment

The accepted TechSpec for `refac-v2` defines `DELETE /api/sessions/:id` as a `204` endpoint with no response body, but `BaseHandlers.StopSession()` still returns `200 OK` plus `{"status":"stopped"}`:

```go
c.JSON(http.StatusOK, gin.H{"status": "stopped"})
```

This is not just a documentation mismatch. The API subtree refactor was supposed to make `api/core` the canonical shared behavior for both transports, so encoding the old status code here keeps the implementation and the accepted contract out of sync. The transport and integration tests currently reinforce the wrong behavior, which makes the drift easy to miss.

**Fix:** Return `c.Status(http.StatusNoContent)` from `StopSession()` and update the affected transport/core tests to assert `204` with an empty body. If the team intentionally wants to keep `200`, then `_techspec.md` must be corrected so the public contract and implementation stop disagreeing.

## Triage

- Decision: `valid`
- Root cause: `BaseHandlers.StopSession` still encodes the pre-refactor `200 {"status":"stopped"}` response even though the accepted refac-v2 tech spec defines a `204 No Content` contract for `DELETE /api/sessions/:id`.
- Fix approach: Return `204` with an empty body from the shared core handler and update the core, HTTP, UDS, and CLI expectations that depend on the delete response.
- Resolution: Implemented across the shared handler and transport tests; full repository verification passed.
