---
status: resolved
file: internal/daemon/channels.go
line: 115
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tt63,comment:PRRC_kwDOR5y4QM624Xyv
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Guard lifecycle writes against stale-snapshot races.**

These paths read instance state, then later persist a forward transition or compensating rollback without proving the row is still unchanged. A concurrent stop/restart can be lost here; for example, a failed restart can restore an older `ready` snapshot after another request already disabled the instance. Please serialize lifecycle operations per instance or make both the forward write and rollback conditional on a last-seen version/`UpdatedAt`.



Also applies to: 181-186, 220-257

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/channels.go` around lines 100 - 115, This code performs
lifecycle writes after reading instance state (e.g., in the create flow around
reloadExtensions and persistCompensatingInstance) without guarding against
stale-snapshot races; either serialize per-instance lifecycle operations or make
writes conditional on a last-seen version. Fix by adding one of two options: (A)
serialize operations per instance ID by introducing an instance-level lock
(e.g., a map[string]*sync.Mutex or a singleflight/keyed mutex held around the
read -> reloadExtensions -> persist path using the channel ID) and use it in the
create/reload/rollback flows, or (B) convert the forward write and the
compensating rollback (persistCompensatingInstance and the normal persist/save
call used after reloadExtensions) into optimistic-concurrency updates that
include the prior UpdatedAt/version in the WHERE clause and return a conflict
error if no row matched; then handle that conflict by failing the operation (or
retrying) instead of blindly overwriting. Apply the same guard to the other
similar spots called out (the other lifecycle write paths around the
create/rollback and the block that currently calls persistCompensatingInstance)
so all lifecycle transitions for a given channel ID are either serialized or
conditional on the last-seen version.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/daemon/channels.go` reads a prior instance snapshot in `transitionInstance`, persists a new lifecycle state, and on reload failure unconditionally restores the older snapshot through `persistCompensatingInstance`.
  - That rollback is not conditioned on the row staying unchanged, so a concurrent lifecycle operation can persist a newer state and then get overwritten by the stale rollback.
  - `CreateInstance` has the same create -> reload -> compensating persist window for enabled instances.
  - Fix approach: serialize lifecycle operations per channel instance ID inside `channelRuntime` across create/start/stop/restart/launch paths so reload and rollback operate on one linearized lifecycle stream.

## Resolution

- Added per-instance lifecycle locking in `channelRuntime` and applied it to `CreateInstance` plus all lifecycle transitions before any reload/rollback work.
- Added a concurrent restart/stop regression test that proves a failed restart rollback cannot overwrite a newer stop request.
- Verified with `go test ./internal/daemon` and `make verify`.
