---
status: resolved
file: internal/cli/cli_integration_test.go
line: 1311
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:4ad6aafdc0f2
review_hash: 4ad6aafdc0f2
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 026: Break this lifecycle coverage into focused subtests.
## Review Comment

This single test exercises claim, reconnect, coordination messaging, completion, no-work, and stale-token recovery. Splitting those into `t.Run("Should...")` cases will make failures actionable and keep the setup/recovery flow easier to maintain.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes: `TestCLIAgentTaskLeaseLifecycleIntegration` exercises multiple independent lifecycle behaviors in one long assertion chain. Failures currently do not identify whether claim, reconnect, messaging, completion, no-work, or stale-token recovery regressed.
- Fix: Keep the single integration harness but split the scenario into focused `t.Run("Should...")` phases that preserve the sequential lifecycle dependencies while localizing failures.
