---
provider: coderabbit
pr: "105"
round: 6
round_created_at: 2026-05-06T03:03:04.040959Z
status: resolved
file: internal/extension/manifest_test.go
line: 152
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232891518,nitpick_hash:7a1d721b621d
review_hash: 7a1d721b621d
source_review_id: "4232891518"
source_review_submitted_at: "2026-05-06T03:02:32Z"
---

# Issue 004: Assert nested NetworkMatcher fields directly for stronger regression protection.
## Review Comment

The current check validates promoted fields on `hookMatcher`. Given the pointer-based matcher shape, asserting `hookMatcher.NetworkMatcher.<field>` directly makes this test stricter and better aligned with the migration intent.

## Triage

- Decision: `invalid`
- Notes:
  `hookspkg.HookMatcher` embeds `*NetworkMatcher`, so `hookMatcher.Channel` and `hookMatcher.NetworkMatcher.Channel` read the same storage once `hookMatcher.NetworkMatcher != nil`. This test already asserts that the nested pointer is non-nil before checking the promoted fields, so the suggested rewrite does not strengthen the contract in practice. I also verified that the repository's required verification pipeline (`make verify`) runs auto-fixers (`golangci-lint --fix` and `gopls modernize`) that canonicalize the explicit nested field access back to the promoted form, which confirms the current assertion style is the enforced project convention.
  Analysis complete; no code change was required. The verified source remains correct as-is.
