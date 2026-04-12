---
status: resolved
file: internal/daemon/channels.go
line: 100
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbf,comment:PRRC_kwDOR5y4QM624L_L
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't return a hard failure after the registry mutation already committed.**

Both paths write the new instance/state before calling `reloadExtensions`. If reload fails, the durable change has already happened, but the caller sees an error and may retry or assume nothing changed. Please either roll the mutation back or surface this as partial success instead of a plain failure.




Also applies to: 204-217

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/channels.go` around lines 91 - 100, The code currently
commits the new channel via r.Service.CreateInstance and then returns a hard
error if reloadExtensions fails, which misleads callers; update the
CreateInstance flow to avoid a plain failure after the durable mutation by
either attempting a rollback (call r.Service.DeleteInstance(ctx, created.ID) or
an explicit rollback method when reloadExtensions returns an error, and if
rollback fails include both errors in the returned wrapped error) or by
returning a partial-success result (return the created instance plus a wrapped
error describing the failed reload) instead of nil; apply the same change to the
similar block around reloadExtensions at lines referenced (the other instance
handling block) and use the symbols Service.CreateInstance, reloadExtensions,
and Service.DeleteInstance (or the service's rollback API) to locate and
implement the fix.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The current runtime persists the new instance/state before `reloadExtensions`, then returns a plain failure if reload fails. That leaves durable state changed even though the requested operational transition did not finish, which is misleading for callers and makes retries ambiguous.
  Resolved in `internal/daemon/channels.go` by compensating reload failures: lifecycle transitions now restore the previously persisted instance record, and enabled creates now roll the new instance back to `disabled` before returning an error because the current registry surface does not expose deletion. Regression coverage was added in `internal/daemon/channels_test.go`. Verified with `go test ./internal/daemon -count=1`, `go test -tags integration ./internal/daemon -count=1`, and the final `make verify` pass.
