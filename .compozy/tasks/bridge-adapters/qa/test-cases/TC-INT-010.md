## TC-INT-010: Managed Instance Sync Reconciliation

**Priority:** P1
**Type:** Integration
**Systems:** bridges.ManagedSyncService, bridges.ManagedSyncStore, bridges.BridgeInstanceSource, extension.Manager (install_managed), extension.Manifest, store/globaldb
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-15

---

### Objective
Validate the managed bridge instance reconciliation pipeline: extension manifests declare bridge instances (source=package), the `ManagedSyncService.SyncManagedInstances` method compares the desired set against the persisted set in globaldb, inserts missing instances, updates changed instances (preserving `created_at`), and deletes orphaned instances. Confirms the `sameManagedInstance` comparison function and the `ManagedSyncStats` counters.

### Preconditions
- [ ] globaldb initialized via `t.TempDir()` with a clean schema
- [ ] ManagedSyncStore implementation backed by the real globaldb (not mocked)
- [ ] Clock overridden via `WithManagedSyncNow` for deterministic timestamps

### Test Steps
1. **Initial sync: insert 3 package-sourced instances**
   - Input: Desired set = 3 `BridgeInstance` entries: `brg-pkg-1` (scope=global, platform=telegram, display_name="TG Bot 1", enabled=true, status=starting, source=package), `brg-pkg-2` (scope=global, platform=telegram, display_name="TG Bot 2", enabled=true, status=starting, source=package), `brg-pkg-3` (scope=workspace, workspace_id=ws-1, platform=slack, display_name="Slack Bot", enabled=false, status=disabled, source=package)
   - **Expected:** `SyncManagedInstances(ctx, BridgeInstanceSourcePackage, desired)` returns `ManagedSyncStats{InstancesSynced: 3, InstancesRemoved: 0, SyncedAt: <clock>}`

2. **Verify all 3 instances persisted in globaldb**
   - Input: Call `store.ListBridgeInstances(ctx)`
   - **Expected:** Returns 3 instances; all have `source=package`; IDs match `brg-pkg-1`, `brg-pkg-2`, `brg-pkg-3`; `created_at` is set to the overridden clock time

3. **Re-sync with no changes (idempotent)**
   - Input: Call `SyncManagedInstances` with the same desired set
   - **Expected:** `ManagedSyncStats{InstancesSynced: 3, InstancesRemoved: 0}`; no database writes (instances unchanged per `sameManagedInstance`)

4. **Sync with one instance modified (display_name changed)**
   - Input: Change `brg-pkg-1` desired entry to `display_name="Updated TG Bot 1"`; advance the clock by 1 minute
   - **Expected:** `ManagedSyncStats{InstancesSynced: 3, InstancesRemoved: 0}`; globaldb row for `brg-pkg-1` has `display_name="Updated TG Bot 1"`; `created_at` is preserved from the original insert; `updated_at` reflects the new clock time

5. **Sync with one instance removed from desired set**
   - Input: Remove `brg-pkg-3` from the desired set (now only 2 entries)
   - **Expected:** `ManagedSyncStats{InstancesSynced: 2, InstancesRemoved: 1}`; `brg-pkg-3` is deleted from globaldb; only `brg-pkg-1` and `brg-pkg-2` remain

6. **Sync with a new instance added to desired set**
   - Input: Add `brg-pkg-4` (scope=global, platform=discord, display_name="Discord Bot", source=package) to the desired set
   - **Expected:** `ManagedSyncStats{InstancesSynced: 3, InstancesRemoved: 0}`; globaldb now has `brg-pkg-1`, `brg-pkg-2`, `brg-pkg-4`

7. **Verify dynamic instances are not affected by managed sync**
   - Input: Insert a `source=dynamic` instance `brg-dyn-1` directly into globaldb; run sync with package desired set
   - **Expected:** `brg-dyn-1` is untouched; sync only affects `source=package` instances

8. **Verify duplicate desired IDs are rejected**
   - Input: Desired set contains two entries with `id=brg-pkg-1`
   - **Expected:** `SyncManagedInstances` returns error: "bridges: duplicate desired managed instance"

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| Desired.Source | BridgeInstanceSourcePackage | Persisted source=package | |
| Desired.ID | `brg-pkg-1` | globaldb bridge_instances.id = `brg-pkg-1` | |
| Modified display_name | `Updated TG Bot 1` | globaldb bridge_instances.display_name updated | |
| Preserved created_at | Original insert timestamp | Unchanged after update | |
| Orphaned instance | `brg-pkg-3` removed from desired | Deleted from globaldb | |
| Dynamic instance | `brg-dyn-1` with source=dynamic | Not touched by package sync | |
| ManagedSyncStats.SyncedAt | Clock override | Matches `now()` at sync time | |

### Error Scenarios
- [ ] Desired instance fails validation (e.g., empty platform): `SyncManagedInstances` returns error with instance ID context
- [ ] Store insert fails (e.g., unique constraint): returns wrapped error with insert context
- [ ] Store delete fails: returns wrapped error with delete context
- [ ] Invalid source (e.g., "custom"): `BridgeInstanceSource.Validate()` rejects
- [ ] Nil context: returns "bridges: managed sync context is required"
- [ ] Nil store: returns "bridges: managed sync store is required"
- [ ] DeliveryDefaults changed in desired: `sameManagedInstance` detects the difference via `managedSyncJSONEqual`

### Related Test Cases
- TC-INT-001 (launched provider depends on synced managed instances)
- TC-INT-007 (CRUD operations interact with the same globaldb rows; managed instances are read-only via CRUD)
