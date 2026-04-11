---
status: resolved
file: internal/network/manager.go
line: 217
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZi,comment:PRRC_kwDOR5y4QM623eZz
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't drop transport shutdown failures during constructor rollback.**

The three rollback branches here cancel the lifecycle and then call `transport.Shutdown(context.Background())` while discarding the result. If shutdown fails, `NewManager` returns the original error with a partially alive transport still running, and the cleanup path ignores the caller's context entirely.

As per coding guidelines, "Never ignore errors with _ — every error must be handled or have a written justification" and "Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager.go` around lines 189 - 217, The rollback branches
inside NewManager currently call cancel() and call
transport.Shutdown(context.Background()) while discarding its error; update each
failure path (after NewPeerRegistry, NewRouter, and NewAuditWriter) to call
transport.Shutdown using the incoming context (not context.Background()),
capture its error, and return a combined error that includes both the original
constructor error and any shutdown error (e.g., wrap with fmt.Errorf or
multierror) instead of ignoring it; ensure you reference the same symbols
(NewPeerRegistry, NewRouter, NewAuditWriter, transport.Shutdown, cancel) so the
cleanup uses the caller context and surface shutdown failures to the caller.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the constructor rollback paths cancel the lifecycle and call `transport.Shutdown(context.Background())`, then discard the shutdown error, so cleanup failures are silently lost and ignore the caller context.
- Fix approach: centralize rollback cleanup into a helper that uses the caller context and combines the constructor error with any transport shutdown failure.
- Scope note: this also required a minimal verification addition in `internal/network/manager_test.go` because the scoped file list did not include an existing manager-constructor rollback test.
