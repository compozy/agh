---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/extension/manifest_test.go
line: 150
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:ffbfbba74fec
review_hash: ffbfbba74fec
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 018: Assert the full matcher mapping here.
## Review Comment

This only checks `Channel` and `WorkState` after `hookConfigMatcher(...)`. If the conversion drops `Surface`, `Kind`, or `Direction`, the new test still passes even though those are part of the network matcher path added in this PR.

## Triage

- Decision: `valid`
- Notes: `TestLoadManifestParsesHookMatcherFields` verifies only `Channel` and `WorkState` after `hookConfigMatcher(...)`. That misses regressions in `Surface`, `Kind`, and `Direction`, which are also part of the network matcher translation. Expand the assertion to cover the full mapping.
