---
status: resolved
file: extensions/bridges/github/provider_test.go
line: 1050
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:e4870173171b
review_hash: e4870173171b
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 002: Test name no longer matches behavior
## Review Comment

The test is still named `TestGitHubProviderReconcileRejectsSharedWebhookPaths`, but it now asserts shared paths are accepted. Please rename it to reflect the new contract (e.g., `...AllowsSharedWebhookPaths`) to avoid future misreads.

## Triage

- Decision: `VALID`
- Notes: The test name still says shared webhook paths are rejected, but the assertions explicitly require both configs to be accepted. That mismatch is real and will be corrected while fixing the shared-path signature-routing bug covered by issue 003.
