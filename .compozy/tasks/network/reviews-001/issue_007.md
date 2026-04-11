---
status: resolved
file: internal/cli/network_test.go
line: 186
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:f1d91a3616d4
review_hash: f1d91a3616d4
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 007: Consider using t.Run subtests for validation cases.
## Review Comment

This would provide better test isolation and clearer failure reporting. As per coding guidelines, table-driven tests with subtests (`t.Run`) are the default pattern.

## Triage

- Decision: `valid`
- Root cause: `TestNetworkSendParsersRejectInvalidFlags` bundles several validation paths in a single function without subtests, which makes failures noisier and diverges from local test style.
- Fix approach: convert the invalid-flag cases into table-driven `t.Run("Should...")` subtests.
