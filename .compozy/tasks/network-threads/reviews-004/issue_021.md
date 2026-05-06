---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/hooks_test.go
line: 1571
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:deb89bedb2ed
review_hash: deb89bedb2ed
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 021: Add a non-matching compaction case here.
## Review Comment

This only exercises `CompactionMatcher` with the matching `"token_limit"` reason, so the test still passes if the new nested matcher is ignored completely. Add a second dispatch with a different `Reason` and assert the payload stays unchanged.

Also applies to: 1652-1655

## Triage

- Decision: `VALID`
- Root cause: `TestHooksDispatchPermissionHooksAndContextCompactionMatcher` only proves the positive `CompactionMatcher{Reason:"token_limit"}` path. If the nested compaction matcher were ignored entirely, the matching dispatch would still patch the payload and this test would keep passing.
- Fix approach: add a second compaction dispatch with a different reason and assert the payload remains unchanged, so the test fails if matcher filtering is bypassed.
- Verification: fixed in scoped code and validated with fresh `make verify`.
