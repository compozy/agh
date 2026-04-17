---
status: resolved
file: internal/bundles/service.go
line: 665
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:4378d9753d34
review_hash: 4378d9753d34
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 025: Preload bundle resources once per request/reconcile instead of listing them per activation.
## Review Comment

This lookup path does a fresh `ListBundleResources(ctx)` for every activation resolution. On larger activation sets that turns reconcile into repeated full-store scans, and different activations can be resolved against different bundle snapshots inside the same operation.

## Triage

- Decision: `INVALID`
- Notes:
  - The current `internal/bundles/service.go` does not call `ListBundleResources` during activation resolution or reconcile.
  - Desired bundle state is collected once into `reconcileState` in `collectDesiredState`, so the repeated full-store-scan path described in the comment is gone.
  - This review comment no longer applies to the live implementation.
  - Result: resolved as stale after current-tree inspection; no code change required.
