---
status: resolved
file: internal/automation/manager.go
line: 1254
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaE,comment:PRRC_kwDOR5y4QM623-TM
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Remove overlays and trigger secrets when config-backed definitions disappear.**

These removal paths delete only the base job/trigger row. Any persisted enabled overlay survives, so removing and later re-adding the same config ID silently resurrects stale enabled state. Triggers can also leave their stored webhook secret behind because this path bypasses the explicit secret cleanup used in `DeleteTrigger`.



<details>
<summary>Suggested fix</summary>

```diff
 for id := range existingByID {
 	if _, ok := desiredByID[id]; ok {
 		continue
 	}
+	if err := m.store.DeleteJobEnabledOverlay(ctx, id); err != nil && !errors.Is(err, ErrJobOverlayNotFound) {
+		return 0, 0, fmt.Errorf("automation: delete job overlay %q: %w", id, err)
+	}
 	if err := m.store.DeleteJob(ctx, id); err != nil {
 		return 0, 0, err
 	}
 	removed++
 }
```

```diff
 for id := range existingByID {
 	if _, ok := desiredByID[id]; ok {
 		continue
 	}
+	if err := m.store.DeleteTriggerEnabledOverlay(ctx, id); err != nil && !errors.Is(err, ErrTriggerOverlayNotFound) {
+		return 0, 0, fmt.Errorf("automation: delete trigger overlay %q: %w", id, err)
+	}
+	if err := m.store.DeleteTriggerWebhookSecret(ctx, id); err != nil && !errors.Is(err, ErrTriggerWebhookSecretNotFound) {
+		return 0, 0, fmt.Errorf("automation: delete trigger webhook secret %q: %w", id, err)
+	}
 	if err := m.store.DeleteTrigger(ctx, id); err != nil {
 		return 0, 0, err
 	}
 	removed++
 }
```
</details>


Also applies to: 1292-1299

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager.go` around lines 1247 - 1254, The loop that calls
m.store.DeleteJob (and the similar trigger removal at lines ~1292-1299) only
removes base rows and leaves persisted overlays and webhook secrets behind;
replace direct store.DeleteJob/DeleteTrigger calls with the manager-level
deletion routines that perform full cleanup (e.g., call m.DeleteJob(ctx, id) and
m.DeleteTrigger(ctx, id) or the manager methods that encapsulate overlay and
secret removal) so overlays and trigger secrets are explicitly removed when
config-backed definitions disappear. Ensure the removed++ accounting stays
correct and propagate errors from the manager-level calls as before.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The concern does not hold against the current persistence schema. `automation_job_overlays`, `automation_trigger_overlays`, and `automation_trigger_webhook_secrets` all reference their parent definition rows with `ON DELETE CASCADE` in [internal/store/globaldb/global_db.go](/Users/pedronauck/Dev/projects/_worktrees/automation/internal/store/globaldb/global_db.go).
  - SQLite foreign-key enforcement is enabled during store setup, so deleting the base config-backed job/trigger row already removes the overlay and stored webhook secret rows automatically.
  - The suggested switch to manager-level `DeleteJob` / `DeleteTrigger` is also not correct here because those manager methods intentionally reject config-backed definitions with `ErrDefinitionReadOnly`. No production code change is required for this issue.
  - Resolution: closed as invalid after confirming the existing cascade behavior and preserving the current implementation; final repo verification still passed with no change required.
