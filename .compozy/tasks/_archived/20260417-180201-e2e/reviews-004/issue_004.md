---
status: resolved
file: internal/session/manager_prompt.go
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130947363,nitpick_hash:cc6821abecd4
review_hash: cc6821abecd4
source_review_id: "4130947363"
source_review_submitted_at: "2026-04-17T17:50:20Z"
---

# Issue 004: Consider using a pointer parameter instead of variadic for optional metadata.
## Review Comment

The variadic approach for an optional single value requires defensive length checking and is non-idiomatic Go. Refactoring to `meta *acp.PromptNetworkMeta` would simplify the API, though this requires updating the interface definitions in `deliveryPrompter` and `networkBindableSessionManager`, all implementations, test fakes, and removing the multiple-metadata validation test case (lines 943–955 in manager_test.go).

## Triage

- Decision: `invalid`
- Notes:
  - This comment proposes an API redesign, not a bug fix in the current behavior.
  - The variadic parameter intentionally models an optional metadata argument while preserving the existing `PromptNetwork(...)` call sites across `internal/network`, `internal/daemon`, and their fakes/tests.
  - Switching to `*acp.PromptNetworkMeta` would force wider interface churn for no behavior change, while the current implementation already validates the "at most one metadata value" contract explicitly and has dedicated test coverage.
  - No code change is warranted for this batch.
