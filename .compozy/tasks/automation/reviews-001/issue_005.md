---
status: resolved
file: internal/api/testutil/apitest.go
line: 142
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:b9e7c36ed732
review_hash: b9e7c36ed732
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 005: Add compile-time verification for StubAutomationManager.
## Review Comment

This new stub is meant to satisfy `core.AutomationManager`, but unlike the other stubs in this file it is not pinned with a `var _ ...` assertion. Adding one will catch interface drift earlier as the automation surface evolves.

As per coding guidelines, "Use compile-time interface verification: var _ Interface = (*Type)(nil)".

## Triage

- Decision: `valid`
- Notes: `StubAutomationManager` is intended to satisfy `core.AutomationManager`, but this file does not pin that contract with a compile-time assertion the way the other test stubs do. I will add the assertion so interface drift fails at compile time instead of later in tests.
