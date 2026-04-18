---
status: resolved
file: internal/daemon/prompt_input_composite_integration_test.go
line: 146
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:3920dff203a7
review_hash: 3920dff203a7
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 019: Clone EnableAugmenters before appending.
## Review Comment

Appending directly to the slice returned by `r.base.ResolvePrompt` can mutate shared backing storage if that policy is reused, which makes this helper stateful across calls.

## Triage

- Decision: `valid`
- Notes:
  - `append(resolved.Policy.EnableAugmenters, additional...)` can reuse the slice backing array returned by the base resolver.
  - That makes the overlay resolver stateful across calls if the base resolver reuses the same slice, which is a real test-helper hazard.
  - I will clone the base slice before appending so each resolution result is isolated.
