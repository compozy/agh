---
status: resolved
file: internal/session/manager_hooks_test.go
line: 237
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:6c9a1c1f241e
review_hash: 6c9a1c1f241e
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 027: Wrap this case in t.Run("Should...") to match test conventions.
## Review Comment

The assertions are solid; just align structure with the suite’s required subtest pattern so future case expansion stays uniform.

As per coding guidelines, "MUST use `t.Run(\"Should...\")` pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default".

## Triage

- Decision: `valid`
- Notes:
  `TestPromptNetworkUsesNetworkInputClass` is behaviorally correct, but this repository’s Go test convention requires subtests in `t.Run("Should...")` form even for single-scenario cases. This change is mechanical but valid for consistency with the enforced suite style.
  Resolved by wrapping the case in `t.Run("ShouldUseNetworkInputClass")` inside `internal/session/manager_hooks_test.go`. Verified with package tests and a clean `make verify`.
