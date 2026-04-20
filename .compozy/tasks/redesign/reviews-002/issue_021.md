---
status: resolved
file: packages/ui/src/components/kind-chip.test.tsx
line: 7
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:151dae3eb842
review_hash: 151dae3eb842
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 021: Test title is misleading vs asserted behavior.
## Review Comment

Line 7 says “render the kind lowercase”, but Line 11 asserts `"Greet"` (original casing). Consider renaming the test to reflect that lowercase is CSS-driven (`className`) rather than text normalization.

## Triage

- Decision: `valid`
- Notes:
  - The test title says the component renders lowercase text, but the assertion checks the original text content (`"Greet"`). Lowercasing is presentation-only via the `lowercase` class, not text normalization.
  - Fix by renaming the test to describe CSS-driven lowercase presentation while keeping the current content assertion.
