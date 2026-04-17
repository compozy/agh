---
status: resolved
file: internal/bundles/resource_store.go
line: 600
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:565fa3220041
review_hash: 565fa3220041
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 023: Potential issue: scope comparison in sameEncodedSpec is always true.
## Review Comment

The comparison `scope == scope.Normalize()` on line 614 compares the scope to itself (normalized), which will always be true if the scope was already normalized. This seems like it should compare `record.Scope` to the expected scope for the desired spec instead.

The scope comparison is redundant here since callers already compare scopes before calling this function (e.g., lines 428, 461, 494).

## Triage

- Decision: `INVALID`
- Notes:
  - `internal/bundles/resource_store.go` is not present in the current tree, and a repo-wide search found no surviving `sameEncodedSpec` helper.
  - The current bundle reconcile path in `internal/bundles/service.go` does not compare bundle resource scope through a helper like the one described here.
  - This review comment is stale against a pre-rebase file layout.
  - Result: resolved as stale after current-tree inspection; no code change required.
