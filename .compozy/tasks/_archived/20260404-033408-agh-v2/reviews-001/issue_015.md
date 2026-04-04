---
status: resolved
file: internal/store/session_db.go
line: 300
severity: medium
author: claude-code
provider_ref:
---

# Issue 015: Writer goroutine leaks if Close() is never called

## Review Comment

The `writerLoop` goroutine (started at line 96, defined at line 300) only exits when it receives on `shutdownCh`. If `Close()` is never called (e.g., the caller drops all references to the `SessionDB`), the goroutine leaks forever along with the open `sql.DB` connection.

This violates the project rule: "Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation." The goroutine has no connection to any `context.Context`.

**Suggested fix:** Accept a parent `context.Context` in `OpenSessionDB`, and select on `ctx.Done()` in the writer loop as a fallback shutdown path. This ensures the goroutine exits even if `Close()` is never called.

## Triage

- Decision: `invalid`
- Notes: `SessionDB` is an owned resource with an explicit `Close()` contract, and the production call sites in the current codebase close it as part of session teardown. Reusing the `OpenSessionDB` call context as a lifetime context would be incorrect because that context is only for opening, not for owning the DB for the session lifetime. This report describes hypothetical API misuse, not a demonstrated runtime bug.
