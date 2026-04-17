---
status: resolved
file: internal/daemon/boot.go
line: 1070
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:8d0a755eab64
review_hash: 8d0a755eab64
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 036: Consider consolidating the repeated sync pattern.
## Review Comment

The `syncExtensionResourcePublishers` function has a repetitive pattern of nil-check-then-sync. Consider extracting a helper to reduce duplication.

---

## Triage

- Decision: `INVALID`
- Notes:
  - `syncExtensionResourcePublishers` does not exist in the current `internal/daemon/boot.go`.
  - The repeated nil-check sync helper described in the review has already been removed or consolidated away in this checkout.
  - There is no live code path to refactor for this item.
  - Result: resolved as stale after current-tree inspection; no code change required.
