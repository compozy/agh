---
status: resolved
file: internal/bundles/service.go
line: 719
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130784468,nitpick_hash:2d320a249eb1
review_hash: 2d320a249eb1
source_review_id: "4130784468"
source_review_submitted_at: "2026-04-17T17:23:10Z"
---

# Issue 002: Normalize indexed lookup keys to the same case-insensitive policy.
## Review Comment

`findBundleResourceRecordIndexed` still falls back to a full scan whenever activation casing differs from the stored bundle casing, because `newBundleRecordKey` only trims. Lowercasing the key at construction time keeps the fast path effective for mixed-case inputs and matches the behavior of `findBundleResourceRecord`.

## Triage

- Decision: `valid`
- Root cause: `newBundleRecordKey` trims keys but preserves case, so the indexed lookup misses mixed-case activation inputs and falls back to a linear scan even though the non-indexed matcher is case-insensitive.
- Fix plan: normalize key construction to the same trimmed, case-insensitive policy used by `findBundleResourceRecord` and add a focused lookup test that proves the normalized map path works without the fallback scan.
- Resolution: `newBundleRecordKey` now lowercases trimmed extension and bundle names, and `TestFindBundleResourceRecordIndexedNormalizesLookupKeys` proves the indexed lookup succeeds without the scan fallback.
- Verification: `go test ./internal/bundles ./internal/environment/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
