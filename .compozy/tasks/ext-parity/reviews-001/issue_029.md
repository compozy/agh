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

# Issue 029: Preload bundle resources once per request/reconcile instead of listing them per activation.
## Review Comment

This lookup path does a fresh `ListBundleResources(ctx)` for every activation resolution. On larger activation sets that turns reconcile into repeated full-store scans, and different activations can be resolved against different bundle snapshots inside the same operation.

## Triage

- Decision: `VALID`
- Notes: `collectDesiredState` still resolves each activation through `resolveActivation`, and `resolveActivationDefinition` performs a fresh `ListBundleResources` call every time. On larger activation sets that causes repeated full bundle scans and allows different activations in one reconcile to observe different bundle snapshots. The fix is to preload bundle records once per reconcile and resolve against that shared snapshot, with a regression test that counts store lookups.
