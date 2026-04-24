---
status: resolved
file: internal/api/core/coverage_helpers_test.go
line: 354
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151559901,nitpick_hash:ddfa247ed8db
review_hash: ddfa247ed8db
source_review_id: "4151559901"
source_review_submitted_at: "2026-04-22T01:22:21Z"
---

# Issue 005: Split the new helper assertions into named subtests.
## Review Comment

The preview and payload cases are independent now, but they are still bundled into one large test body. Breaking them into `t.Run("Should...")` cases will make failures much easier to localize.

As per coding guidelines, "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Reasoning: `TestStatusForBundleErrorAndChannelHelpers` currently bundles the network helper assertions into one linear body, so a failure does not identify which helper path regressed. Splitting them into named `t.Run("Should...")` cases matches the project test style and pairs naturally with the additional persisted-channel coverage.
- Fix plan: split the helper assertions into focused subtests for session visibility, peer visibility, persisted metadata, missing channels, and not-found error detection.
- Resolution: split the bundled network helper assertions into focused named subtests for session visibility, peer visibility, persisted metadata, missing channels, and not-found error detection.
- Verification: `go test ./internal/api/core` and `make verify`
