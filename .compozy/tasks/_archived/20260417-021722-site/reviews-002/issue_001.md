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

# Issue 001: Test name no longer matches behavior
## Review Comment

The test is still named `TestGitHubProviderReconcileRejectsSharedWebhookPaths`, but it now asserts shared paths are accepted. Please rename it to reflect the new contract (e.g., `...AllowsSharedWebhookPaths`) to avoid future misreads.

## Triage

- Decision: `INVALID`
- Reason: The review comment is stale against the current implementation. The production code still rejects duplicate webhook paths in [extensions/bridges/github/provider.go](/Users/pedronauck/Dev/compozy/_worktrees/site/extensions/bridges/github/provider.go#L695), so the existing test name `TestGitHubProviderReconcileRejectsSharedWebhookPaths` accurately matches the current contract.

## Resolution

- Analysis complete. No code change was required because the reviewed behavior and test name still match the current contract.
