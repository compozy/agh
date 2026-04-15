---
status: resolved
file: internal/bridges/managed_sync.go
line: 145
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bi,comment:PRRC_kwDOR5y4QM63zbx9
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap store failures with reconcile context.**

The list/insert/update/delete branches all return bare store errors, so a failed sync loses which source/id and phase failed. That will make bundle-reconcile failures much harder to diagnose once this is wired into activation flows. As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bridges/managed_sync.go` around lines 96 - 145, The store calls in
syncManaged (s.store.ListBridgeInstances, s.store.InsertBridgeInstance,
s.store.UpdateBridgeInstance, s.store.DeleteBridgeInstance) return raw
errors—wrap each returned error with contextual reconcile info using fmt.Errorf,
e.g. include the normalized source (normalizedSource), the bridge id (next.ID or
id) and the phase ("list", "insert", "update", "delete") so the function
(syncManaged) returns errors like fmt.Errorf("bridges: reconcile %s %q: %w",
"<phase>", "<source|id>", err); update the error returns in the
ListBridgeInstances failure, and in the branches that call InsertBridgeInstance,
UpdateBridgeInstance and DeleteBridgeInstance to include that context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `ManagedSyncService.SyncManagedInstances` still returns raw store errors from list/insert/update/delete operations, so reconcile failures lose the managed-source phase and instance identifier that triggered them.
- Fix plan: wrap each store call with reconcile-specific context that includes the phase (`list`/`insert`/`update`/`delete`), the normalized source, and the relevant bridge ID.
- Resolution: wrapped all managed-sync store failures with phase/source/instance context.
- Verification: expanded `internal/bridges/managed_sync_test.go` and passed `go test ./internal/bridges` plus `make verify`.
