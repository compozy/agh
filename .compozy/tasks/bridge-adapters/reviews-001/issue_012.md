---
status: resolved
file: extensions/bridges/github/provider.go
line: 1018
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57LwnB,comment:PRRC_kwDOR5y4QM64DQ0y
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Completed delivery state is never evicted.**

This map is keyed by delivery ID, and every successful delivery stores an entry, but nothing removes old state after terminal events. A long-lived bridge will keep growing this map indefinitely. Drop terminal deliveries or back the store with a bounded TTL/LRU cache.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/github/provider.go` around lines 1014 - 1018,
storeDeliveryState currently always inserts into p.deliveries and never removes
terminal entries, causing unbounded growth; update
githubProvider.storeDeliveryState to compute key := deliveryStateKey(instanceID,
deliveryID), then if the incoming state is a terminal state (implement or use an
isTerminalDeliveryState helper) remove p.deliveries[key] under p.mu (or skip
insertion) instead of storing it, otherwise store/overwrite as before; ensure
p.mu locking remains around the delete/insert to avoid races.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `storeDeliveryState()` unconditionally retains every GitHub delivery entry and never evicts state after terminal events.
  - Since the map key is `(instanceID, deliveryID)`, a long-lived adapter will keep accumulating completed deliveries indefinitely.
  - Planned fix: stop retaining terminal delivery state entries, preserve only in-flight/non-terminal state, and add a regression test covering insert vs eviction behavior.
  - Resolution: `storeDeliveryState()` now evicts entries when the incoming event is terminal via `isTerminalGitHubDeliveryEvent`, preserving only active delivery state; focused unit coverage now checks insert-versus-eviction behavior.
  - Verification: `go test ./extensions/bridges/github -count=1` and `make verify` both passed after the fix.
