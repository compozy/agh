---
status: resolved
file: internal/automation/resource_projection.go
line: 474
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123260080,nitpick_hash:8d7dc6b6a789
review_hash: 8d7dc6b6a789
source_review_id: "4123260080"
source_review_submitted_at: "2026-04-16T17:55:39Z"
---

# Issue 017: Consider tolerating not-found errors when deleting webhook secrets.
## Review Comment

`DeleteTriggerWebhookSecret` is called unconditionally, but non-webhook triggers won't have an associated secret. If the store returns an error for missing secrets, this will fail unnecessarily.

Compare with line 700-703 where `ErrTriggerWebhookSecretNotFound` is explicitly tolerated during managed sync cleanup.

---

## Triage

- Decision: `VALID`
- Root cause: The referenced file was refactored; the equivalent logic now lives in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go#L1704). `syncManagedTriggerWebhookSecret` deletes the stored secret for non-webhook triggers without tolerating `ErrTriggerWebhookSecretNotFound`, so reconciling a non-webhook trigger can fail even when there is nothing to delete.
- Fix approach: Make the managed-sync delete path idempotent by accepting the not-found sentinel, matching the existing cleanup behavior elsewhere in the manager.

## Resolution

- Added a shared delete helper in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go) so non-webhook secret cleanup treats `ErrTriggerWebhookSecretNotFound` as idempotent.
- Added regression coverage in [internal/automation/manager_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager_test.go) for the missing-secret delete path.
