---
status: resolved
file: internal/bridges/managed_sync.go
line: 538
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:428cb847ce70
review_hash: 428cb847ce70
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 021: JSON equality check may produce false negatives for semantically equivalent JSON.
## Review Comment

The `managedSyncJSONEqual` function compares JSON as trimmed strings. This works for identical serializations but will return `false` for semantically equivalent JSON with different formatting (e.g., `{"a":1}` vs `{"a": 1}`). If provider configs or delivery defaults are serialized differently across sources, this could cause unnecessary updates.

If this is intentional for detecting any serialization differences, consider adding a comment. Otherwise, consider canonical JSON comparison:

## Triage

- Decision: `VALID`
- Notes:
  - Current `internal/bridges/managed_sync.go` still compares `DeliveryDefaults` with trimmed string equality in `managedSyncJSONEqual`, so formatting-only JSON differences trigger false updates.
  - Root cause: the reconcile diff path never normalizes JSON before comparing values that are already validated as JSON elsewhere in the bridge model.
  - Intended fix: compare normalized JSON in `managed_sync.go` and add a regression test in the current `internal/bridges/managed_sync_test.go`.
  - Result: implemented in the live managed sync code and covered by `TestManagedSyncerIgnoresEquivalentDeliveryDefaultsFormatting`; verified with `go test ./internal/bridges` and `make verify`.
