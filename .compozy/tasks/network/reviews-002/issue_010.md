---
status: resolved
file: internal/network/delivery_integration_test.go
line: 24
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:b37303161547
review_hash: b37303161547
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 010: Consider adding t.Parallel() to independent integration tests.
## Review Comment

Per coding guidelines, independent tests should use `t.Parallel()` for parallel execution. Since these integration tests use isolated resources (separate managers, temp directories), they should be safe to run in parallel.

```diff
func TestDeliveryCoordinatorIntegrationDrainsOneQueuedPromptPerTurn(t *testing.T) {
+ t.Parallel()
manager, driver := newDeliveryIntegrationHarness(t)
```

## Triage

- Decision: `valid`
- Root cause: The integration harness uses isolated temp directories and per-test managers, so the test is independent but does not opt into parallel execution.
- Fix plan: Add `t.Parallel()` to the test and keep the existing isolated setup unchanged.
