---
status: resolved
file: internal/workspace/resolver.go
line: 197
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:9fb029bfbe86
review_hash: 9fb029bfbe86
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 021: Bound rollback deletion with a timeout.
## Review Comment

Line 197 detaches cancellation via `context.WithoutCancel(ctx)`, but without a deadline this rollback path can block indefinitely if `DeleteWorkspace` hangs.

## Triage

- Decision: `valid`
- Root cause: `ResolveOrRegister` uses `context.WithoutCancel(ctx)` for rollback deletion after `Resolve` fails. That preserves values but drops both cancellation and any deadline, so a stalled store delete can hang the rollback path indefinitely.
- Why this is a bug: rollback runs on an error path that should remain bounded. Other packages in this repo already wrap detached cleanup work with `context.WithTimeout(context.WithoutCancel(ctx), ...)` for the same reason.
- Fix approach: introduce a package-local bounded rollback-delete helper in the resolver path and use it for this rollback call so cleanup still ignores the caller cancellation but cannot block forever.
- Resolution: `ResolveOrRegister` now routes rollback deletion through a shared helper that wraps `context.WithoutCancel(ctx)` in `context.WithTimeout(..., 2*time.Second)`.
- Verification: `go test ./internal/workspace` and `make verify` both passed after the fix.
