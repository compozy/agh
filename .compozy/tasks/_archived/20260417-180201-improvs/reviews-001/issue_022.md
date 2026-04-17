---
status: resolved
file: internal/workspace/resolver_crud.go
line: 25
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:25f7147bfb97
review_hash: 25f7147bfb97
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 022: Use a bounded detached context for rollback delete.
## Review Comment

Line 25 has the same unbounded rollback-context issue; if store deletion stalls, this can hang registration flow.

## Triage

- Decision: `valid`
- Root cause: `Register` repeats the same rollback pattern as `ResolveOrRegister`, calling `DeleteWorkspace` with `context.WithoutCancel(ctx)` and no deadline after a failed resolve.
- Why this is a bug: if the store delete blocks, registration cleanup can hang permanently even though the operation is already failing. That turns a recoverable rollback into an unbounded wait.
- Fix approach: reuse the same bounded detached rollback-delete helper in this code path so both registration flows share one timeout policy and one implementation.
- Resolution: `Register` now uses the shared rollback-delete helper, so both registration rollback paths share the same bounded detached cleanup behavior.
- Verification: `go test ./internal/workspace` and `make verify` both passed after the fix.
