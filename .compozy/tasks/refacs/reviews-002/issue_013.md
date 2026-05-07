---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/bundles/lookup.go
line: 47
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4247165327,nitpick_hash:2d0ec6bd7b48
review_hash: 2d0ec6bd7b48
source_review_id: "4247165327"
source_review_submitted_at: "2026-05-07T19:37:05Z"
---

# Issue 013: Fallback scan is dead code for non-empty keys and creates an inconsistency for empty keys.
## Review Comment

Because `newBundleRecordKey` normalises identically at both index-build time and lookup time, any record reachable by the fallback scan would already be present in `lookup.exact`. The fallback is therefore unreachable for any non-empty `(extensionName, bundleName)` pair.

For the empty-key case there is a subtle inconsistency: records whose `extensionName` or `bundleName` is empty are **explicitly excluded** from the index (lines 24–26), but the fallback scan would still surface them if the caller passes empty strings. If those records are intentionally invalid, the fallback should mirror that exclusion; if they are valid, they should be indexed.

## Triage

- Decision: `valid`
- Notes:
  - `internal/bundles/lookup.go:42-52` performs a fallback linear scan after the normalized-index lookup misses, but the scan uses the same normalization as `newBundleRecordKey` for non-empty keys.
  - The only distinct behavior left is that empty normalized keys are skipped during index construction yet can still be found by the fallback scan, which is inconsistent.
  - Fix plan: remove the dead fallback path so lookups obey the same non-empty-key rule as the index; this requires a minimal out-of-scope regression test update in `internal/bundles/service_test.go` because `lookup.go` has no in-scope test file.
  - Resolved: the dead fallback scan was removed and the bundle lookup regression coverage now asserts that empty keys do not match.
