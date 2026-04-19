---
status: resolved
file: internal/extension/host_api_test.go
line: 1792
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:238426dd550f
review_hash: 238426dd550f
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 022: Assert the fallback text as well.
## Review Comment

This test only checks the envelope fields. A regression that stops populating `projected.Text` from stored `Content` would still pass.

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls".

## Triage

- Decision: `valid`
- Root cause: `promptProjectionEventFromStoredEvent` already projects fallback text from the stored canonical payload, but this regression test only checks envelope fields. A future bug that drops `decoded.Text -> projected.Text` would pass unnoticed.
- Fix approach: Extend the test to assert the recovered fallback text from `storedEvent.Content`, in addition to the existing type/turn/timestamp assertions.
