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

# Issue 027: Potential issue: scope comparison in sameEncodedSpec is always true.
## Review Comment

The comparison `scope == scope.Normalize()` on line 614 compares the scope to itself (normalized), which will always be true if the scope was already normalized. This seems like it should compare `record.Scope` to the expected scope for the desired spec instead.

The scope comparison is redundant here since callers already compare scopes before calling this function (e.g., lines 428, 461, 494).

## Triage

- Decision: `INVALID`
- Notes: `sameEncodedSpec` is only called after each caller has already compared `existing.Scope` against the expected normalized resource scope for the desired spec. That makes `scope == scope.Normalize()` redundant, but not incorrect, and removing it does not fix a behavioral bug or unblock verification for this batch. I am leaving the code as-is and resolving this as a non-actionable cleanup comment.
