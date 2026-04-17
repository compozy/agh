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

# Issue 023: JSON equality check may produce false negatives for semantically equivalent JSON.
## Review Comment

The `managedSyncJSONEqual` function compares JSON as trimmed strings. This works for identical serializations but will return `false` for semantically equivalent JSON with different formatting (e.g., `{"a":1}` vs `{"a": 1}`). If provider configs or delivery defaults are serialized differently across sources, this could cause unnecessary updates.

If this is intentional for detecting any serialization differences, consider adding a comment. Otherwise, consider canonical JSON comparison:

## Triage

- Decision: `VALID`
- Notes: `managedSyncJSONEqual` still treats compacted JSON as raw strings, so semantically identical objects with different key order or whitespace compare unequal and force unnecessary managed bridge updates. Bridge JSON normalization compacts but does not canonicalize object order, so this is a real false-positive path. The fix is to compare decoded JSON values semantically. The regression coverage requires a minimal out-of-scope update to `internal/bridges/managed_sync_test.go` because no scoped test file exercises the managed-sync path.
