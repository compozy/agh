---
status: resolved
file: internal/bundles/service.go
line: 533
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bu,comment:PRRC_kwDOR5y4QM63zbyO
---

# Issue 007: _⚠️ Potential issue_ | _🔴 Critical_

## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Do not continue with destructive sync after `resolveActivation` fails.**

The loop drops unresolved activations from `desiredJobs`, `desiredTriggers`, and `desiredBridges`, then still reconciles the remaining set. On a transient extension/bundle load failure, that will delete the already-materialized resources for the failing activation before returning the error, and the activation rollback path here does not restore them.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bundles/service.go` around lines 480 - 533, The loop over
activations can continue and perform destructive syncs even if some activations
failed to resolve; change the flow so that if any resolveActivation(ctx,
activation) returned an error (i.e., errs is non-empty after the activations
loop) you short-circuit and return that aggregated error instead of proceeding
to call s.automation.SyncManagedDefinitions or s.bridges.SyncManagedInstances;
in practice, after the for ... range loop check if len(errs) > 0 and return the
composed error (or errors.Join(errs)) so no destructive reconciliation of
desiredJobs/desiredTriggers/desiredBridges occurs when any activation resolution
failed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `reconcileLocked` accumulates activation-resolution errors in `errs` but still proceeds into automation/bridge reconciliation, which can delete already-materialized managed resources for any activation that failed to resolve in this pass.
- Fix plan: stop before any destructive sync once the activation loop produces resolve/default-channel errors, returning the aggregated error immediately instead of reconciling a partial desired set.
- Resolution: added an early return before automation/bridge sync when activation resolution or default-channel validation already failed.
- Verification: added the regression in `internal/bundles/service_test.go` and passed `go test ./internal/bundles` plus `make verify`.
